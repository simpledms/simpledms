package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/marcobeierer/structs"
	"github.com/mattn/go-sqlite3"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/session"
	"github.com/simpledms/simpledms/db/entmain/tenant"
	"github.com/simpledms/simpledms/db/entmain/tenantaccountassignment"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/i18n"
	"github.com/simpledms/simpledms/model/account"
	tenant2 "github.com/simpledms/simpledms/model/tenant"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/cookiex"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type handlerFn func(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error
type Actionable interface {
	Route() string
	Endpoint() string
	FormRoute() string
	IsReadOnly() bool
	UseManualTxManagement() bool
	// Handler(httpx.ResponseWriter, *httpx.Request, *ent.Tx) error
	Handler(httpx.ResponseWriter, *httpx.Request, ctxx.Context) error
}
type FormActionable interface {
	// not always implemented
	//
	// usually you don't want to overwrite the default implementation in util.FormHelper
	// and create a separate form action instead
	FormHandler(httpx.ResponseWriter, *httpx.Request, ctxx.Context) error
}

var tenantIDRegex = regexp.MustCompile(`/org/(?P<id>[a-z0-9]+)`)
var spaceIDRegex = regexp.MustCompile(`/space/(?P<id>[a-z0-9]+)`)

type Router struct {
	*http.ServeMux
	mainDB *sqlx.MainDB
	// TODO if performance problems are observed, dbs can be closed when not needed for some time
	//		should be implementable with go-cache
	tenantDBs  *tenantdbs.TenantDBs
	infra      *common.Infra
	handlerMap map[string]Actionable
	devMode    bool
	metaPath   string
	i18n       *i18n.I18n
}

func NewRouter(
	mainDB *sqlx.MainDB,
	tenantDBs *tenantdbs.TenantDBs,
	infra *common.Infra,
	devMode bool,
	metaPath string,
	i18n *i18n.I18n,
) *Router {
	return &Router{
		ServeMux:   http.NewServeMux(),
		mainDB:     mainDB,
		tenantDBs:  tenantDBs,
		infra:      infra,
		handlerMap: map[string]Actionable{},
		devMode:    devMode,
		metaPath:   metaPath,
		i18n:       i18n,
	}
}

func (qq *Router) RegisterPage(pattern string, handlerFn handlerFn) {
	qq.HandleFunc(pattern, qq.wrapTx(handlerFn, true))
}

func (qq *Router) RegisterActions(actions any) {
	for _, field := range structs.Fields(actions) {
		if actionx, ok := field.Value().(Actionable); ok {
			qq.RegisterAction(actionx)
			log.Println("Registered action", field.Name())
		} else if field.Tag("actions") != "" {
			qq.RegisterActions(field.Value())
		} else {
			// could for example be ListDir, which is just used as partial
			// TODO should that be refactored?
			log.Println("Field is not Actionable", field.Name())
		}
	}
}

func (qq *Router) RegisterAction(
	action Actionable,
) {
	// wrapCommand is also necessary when readOnly because there are read only commands that
	// just manipulate the state, for example ToggleDocumentTypeFilter
	qq.HandleFunc(action.Route(), qq.wrapTx(qq.wrapCommand(action.Handler), action.IsReadOnly()))

	// TODO route or endpoint? does method (POST OR GET) matter? currently not, maybe later?
	qq.handlerMap[action.Endpoint()] = action

	if actionWithForm, ok := action.(FormActionable); ok {
		if action.FormRoute() == "" {
			panic("form route is required, main route was " + action.Route())
		}
		qq.HandleFunc(action.FormRoute(), qq.wrapTx(actionWithForm.FormHandler, true))
	}
}

func (qq *Router) wrapCommand(handlerFn handlerFn) handlerFn {
	return func(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
		// TODO wrap in separate TTx or Same? same probably safer for the moment;
		//		user has no inconsistent state in frontend

		err := handlerFn(rw, req, ctx)
		if err != nil {
			return err
		}

		hxCurrentURL := rw.Header().Get("HX-Current-Url")
		hxReplaceURL := rw.Header().Get("HX-Replace-Url")
		hxPushURL := rw.Header().Get("HX-Push-Url")

		// update CurrentURL in new request; this is necessary, if command manipulates state
		// in URL and query depends on it
		if hxReplaceURL != "" || hxPushURL != "" {
			hxCurrentURLx, err := url.Parse(hxCurrentURL)
			if err != nil {
				log.Println(err)
				return err
			}
			newCurrentURL := hxCurrentURLx

			// TODO which order is better? Can they be set at same time? how would htmx handle it?
			if hxPushURL != "" {
				newCurrentURL, err = hxCurrentURLx.Parse(hxPushURL)
				if err != nil {
					log.Println(err)
					return err
				}
			} else if hxReplaceURL != "" {
				newCurrentURL, err = hxCurrentURLx.Parse(hxReplaceURL)
				if err != nil {
					log.Println(err)
					return err
				}
			}
			req.Header.Set("HX-Current-URL", newCurrentURL.String())
		}

		// TODO path url?
		endpoint := req.Header.Get("X-Query-Endpoint")
		if endpoint == "" {
			err = qq.infra.Renderer().Render(rw, ctx) // render Renderables
			if err != nil {
				log.Println(err)
				return err
			}
			return nil
		}
		data := req.Header.Get("X-Query-Data") // url.Values encoded

		// TODO reuse request or create new one or clone?
		req.Body = io.NopCloser(strings.NewReader(data))

		// reset, otherwise req.ParseForm wouldn't do anything
		req.PostForm = nil
		req.Form = nil

		// IMPORTANT
		// be careful if new middleware, for example for permissions,
		// get added; this could be a vulnerability depending on the
		// implementation
		partialHandlerFn := qq.handlerMap[endpoint].Handler
		if partialHandlerFn == nil {
			log.Println("no handler for", endpoint)
			return errors.New("no handler for " + endpoint)
		}
		err = partialHandlerFn(rw, req, ctx)
		if err != nil {
			return err
		}

		return nil
	}
}

// TODO is this the best place?
func (qq *Router) wrapTx(handlerFn handlerFn, isReadOnly bool) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		// workaround for `open with` function // TODO find a better solution
		if strings.Contains(req.URL.Path, "/inbox/") && req.URL.Query().Has("upload_token") {
			isReadOnly = false
		}

		mainTx, err := qq.mainDB.Tx(req.Context(), isReadOnly)
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		var nilableTenantTx *enttenant.Tx

		rwx := httpx.NewResponseWriter(rw)
		reqx := httpx.NewRequest(req)

		acceptLanguage := req.Header.Get("Accept-Language")
		clientTimezone := req.Header.Get("X-Client-Timezone") // set for all HTMX requests
		isHTMXRequest := req.Header.Get("HX-Request") != ""
		visitorCtx := ctxx.NewVisitorContext(
			req.Context(),
			mainTx,
			qq.i18n,
			acceptLanguage,
			clientTimezone,
			isHTMXRequest,
			qq.infra.SystemConfig().CommercialLicenseEnabled(),
		)

		defer func() {
			// tested and works
			if r := recover(); r != nil {
				// cannot use errors.As because r is `any` not `error`
				// TODO added on 2 April 2025, not sure if it makes sense...
				if err, isErr := r.(error); isErr {
					qq.handleError(rwx, visitorCtx, err, mainTx, nilableTenantTx)
					return
				}

				log.Printf("%v: %s", r, debug.Stack())
				log.Println("trying to recover and rollback transaction")

				qq.handleError(rwx, visitorCtx, fmt.Errorf("Internal error, please contact support."), mainTx, nilableTenantTx)
				return
			}
		}()

		// FIXME is nilableTenantTx assigned to var defined above?
		ctx, nilableTenantTx, isRedirected, err := qq.context(rwx, reqx, mainTx, visitorCtx, isReadOnly)
		if err != nil {
			log.Println(err)
			qq.handleError(rwx, visitorCtx, err, mainTx, nilableTenantTx)
			return
		}
		if isRedirected {
			return
		}

		err = handlerFn(rwx, reqx, ctx)
		if err != nil {
			log.Println(err)
			qq.handleError(rwx, ctx, err, mainTx, nilableTenantTx)
			return
		}

		/*
			seems not to work, also difficult to find best location
			may not be necessary on errors, but status header is already written when reaching this location...

			if reqx.Header.Get("X-Refresh-Page") != "" {
				rwx.Header().Set("HX-Refresh", "true")
			}
		*/

		err = mainTx.Commit()
		if err != nil {
			log.Println(err)
			rwx.WriteHeader(http.StatusInternalServerError)
			return
		}
		if nilableTenantTx != nil {
			err = nilableTenantTx.Commit()
			if err != nil {
				log.Println(err)
				rwx.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}
}

func (qq *Router) handleError(
	rw httpx.ResponseWriter,
	ctx ctxx.Context,
	err error,
	mainTx *entmain.Tx,
	nilableTenantTx *enttenant.Tx,
) {
	var httpErr *e.HTTPError
	isHTTPErr := errors.As(err, &httpErr)

	if !isHTTPErr {
		// check if db foreign key or unique violation
		// TODO handle other ent errors too?

		var entErr *e.HTTPError

		var entMainCErr *entmain.ConstraintError
		var entTenantCErr *enttenant.ConstraintError
		// errors.Is doesn't work for some reason...
		isEntMainCErr := errors.As(err, &entMainCErr)
		isEntTenantCErr := errors.As(err, &entTenantCErr)
		if isEntMainCErr || isEntTenantCErr {
			var errx error
			if isEntMainCErr {
				errx = entMainCErr.Unwrap()
			} else {
				errx = entTenantCErr.Unwrap()
			}
			var sqlErr *sqlite3.Error // was *sqlite.Error
			if errors.As(errx, &sqlErr) {
				switch sqlErr.ExtendedCode { // has Code and ExtendedCode
				// case sqlitelib.SQLITE_CONSTRAINT_UNIQUE:
				case sqlite3.ErrConstraintUnique:
					// TODO good error message?
					entErr = e.NewHTTPErrorf(
						http.StatusBadRequest,
						"A similar entity already exists.",
					)
				// case sqlitelib.SQLITE_CONSTRAINT_FOREIGNKEY:
				case sqlite3.ErrConstraintForeignKey:
					entErr = e.NewHTTPErrorf(
						http.StatusBadRequest,
						"Cannot delete an entity still in use.",
					)
				}
				/*
					const SQLITE_CONSTRAINT_CHECK = 275
					const SQLITE_CONSTRAINT_COMMITHOOK = 531
					const SQLITE_CONSTRAINT_DATATYPE = 3091
					const SQLITE_CONSTRAINT_FOREIGNKEY = 787
					const SQLITE_CONSTRAINT_FUNCTION = 1043
					const SQLITE_CONSTRAINT_NOTNULL = 1299
					const SQLITE_CONSTRAINT_PINNED = 2835
					const SQLITE_CONSTRAINT_PRIMARYKEY = 1555
					const SQLITE_CONSTRAINT_ROWID = 2579
					const SQLITE_CONSTRAINT_TRIGGER = 1811
					const SQLITE_CONSTRAINT_UNIQUE = 2067
					const SQLITE_CONSTRAINT_VTAB = 2323
				*/
			}
			if entErr == nil {
				log.Println(errx)
				entErr = e.NewHTTPErrorf(
					http.StatusInternalServerError,
					"A database constraint violation happened. Please contact the support.",
				)
			}
		}
		var entMValErr *entmain.ValidationError
		var entTValErr *enttenant.ValidationError
		// errors.Is doesn't work for some reason...
		isEntMValErr := errors.As(err, &entMValErr)
		isEntTValErr := errors.As(err, &entTValErr)
		if isEntMValErr || isEntTValErr {
			// logged because I currently don't know if this is triggered in simpledms
			// and if so, if handling can be improved
			log.Println(err)

			// TODO error message?
			entErr = e.NewHTTPErrorf(http.StatusBadRequest, "Data validation failed.")
		}
		if entErr != nil {
			isHTTPErr = true
			httpErr = entErr
		}
	}

	if isHTTPErr {
		rw.WriteHeader(httpErr.StatusCode())
		rw.AddRenderables(httpErr.Snackbar())
	} else {
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		rw.AddRenderables(wx.NewSnackbarf("Something went wrong. Please try again.").SetIsError(true))
	}

	// render error messages in snackbars, no RenderX because of rollback
	err = qq.infra.Renderer().Render(rw, ctx)
	if err != nil {
		log.Println(err)
		// no return, rollback is important
	}

	if err := mainTx.Rollback(); err != nil {
		log.Println(err)
	}
	if nilableTenantTx != nil {
		if err := nilableTenantTx.Rollback(); err != nil {
			log.Println(err)
		}
	}

	return
}

// bool is isRedirected
// TODO better name, does more than just context
func (qq *Router) context(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	mainTx *entmain.Tx,
	visitorCtx *ctxx.VisitorContext,
	isReadOnly bool,
) (ctxx.Context, *enttenant.Tx, bool, error) {
	accountm, isAuthenticated, err := qq.authenticateAccount(rw, req, mainTx)
	if err != nil {
		log.Println(err)
		if errors.Is(err, ErrSessionNotFound) {
			http.Redirect(rw, req.Request, "/", http.StatusSeeOther)
			return visitorCtx, nil, true, nil
		}
		return visitorCtx, nil, false, err
	}
	// TODO is "/" robust enough? should also work if URL is `/?test=123`
	if !isAuthenticated {
		// FIXME should be dynamicly generated list (by declaration)...
		if slices.Contains([]string{
			"/",
			"/pages/about/",
			"/pages/imprint/",
			"/pages/privacy-policy/",
			"/pages/terms-of-service/",
			"/-/auth/reset-password-cmd",
			"/-/auth/reset-password-cmd-form",
			"/-/auth/sign-up-cmd",
			"/-/auth/sign-up-cmd-form",
			"/-/auth/sign-in-cmd",
		}, req.URL.Path) {
			return visitorCtx, nil, false, nil
		} else {
			// no cookie set

			// FIXME message is not shown...
			rw.AddRenderables(wx.NewSnackbarf("You are not signed in. Please sign in to continue."))

			http.Redirect(rw, req.Request, "/", http.StatusSeeOther)
			return visitorCtx, nil, true, nil
		}
	}
	if req.URL.Path == "/" {
		// authenticated, but accessing login page

		// duplicate code in SignIn
		tenantx, err := accountm.Data.
			QueryTenantAssignment().
			Where(tenantaccountassignment.IsDefault(true)).
			QueryTenant().
			Where(tenant.InitializedAtNotNil()).
			Only(visitorCtx)
		if err != nil {
			http.Redirect(rw, req.Request, route2.Dashboard(), http.StatusSeeOther) // 302 or 303?
		} else {
			http.Redirect(rw, req.Request, route2.SpacesRoot(tenantx.PublicID.String()), http.StatusSeeOther) // 302 or 303?
		}
		return visitorCtx, nil, true, nil
	}

	tenantID := req.PathValue("tenant_id")
	spaceID := req.PathValue("space_id")

	// GET check necessary for links to pages via HxGet, for example to switch between spaces in main menu;
	// hx-current-url is old url in these cases
	if req.Header.Get("HX-Request") != "" && req.Method != http.MethodGet {
		currentURLStr := req.Header.Get("HX-Current-URL")
		currentURL, err := url.Parse(currentURLStr)
		if err != nil {
			log.Println(err)
			return visitorCtx, nil, false, e.NewHTTPErrorf(http.StatusBadRequest, "Could not parse url.")
		}

		matchesOrg := tenantIDRegex.FindStringSubmatch(currentURL.Path)
		if len(matchesOrg) > 1 {
			// TODO is it also possible to get value by group name?
			tenantID = matchesOrg[1]
		} else {
			// not found, can be a valid request
		}

		matches := spaceIDRegex.FindStringSubmatch(currentURL.Path)
		if len(matches) > 1 {
			// TODO is it also possible to get value by group name?
			spaceID = matches[1]
		} else {
			// not found, can be a valid request
		}
	}

	mainCtx := ctxx.NewMainContext(visitorCtx, accountm.Data, qq.i18n, qq.mainDB, qq.tenantDBs, isReadOnly)
	if tenantID == "" { // spaceID doesn't have to be checked, can only be set if Tenant is set
		return mainCtx, nil, false, nil
	}

	tenantx := mainTx.Tenant.Query().Where(tenant.PublicID(entx.NewCIText(tenantID))).OnlyX(mainCtx)
	tenantClient, ok := qq.tenantDBs.Load(tenantx.ID)
	if !ok {
		// IMPORTANT don't initialize here because this could trigger concurrency issues...

		tenantm := tenant2.NewTenant(tenantx)

		tenantClient, err = tenantm.OpenDB(qq.devMode, qq.metaPath)
		if err != nil {
			log.Println(err)
			return mainCtx, nil, false, err
		}

		qq.tenantDBs.Store(tenantx.ID, tenantClient)
	}

	tenantTx, err := tenantClient.Tx(mainCtx, isReadOnly)
	if err != nil {
		log.Println(err)
		return mainCtx, nil, false, err
	}

	// verify that account belangs to tenant
	tenantm := tenant2.NewTenant(tenantx)
	// if !accountm.BelongsToTenant(mainCtx, tenantm) {
	if !tenantm.HasAccount(mainCtx, accountm) {
		// TODO does this render?
		// rwx.AddRenderables(wx.NewSnackbarf("You are not allowed to access this tenant."))
		// rwx.WriteHeader(http.StatusForbidden)
		return mainCtx, tenantTx, false, e.NewHTTPErrorf(http.StatusForbidden, "You are not allowed to access this tenant.")
	}

	tenantCtx := ctxx.NewTenantContext(mainCtx, tenantTx, tenantx, isReadOnly)
	if spaceID == "" {
		return tenantCtx, tenantTx, false, nil
	}

	spacex := tenantTx.Space.
		Query().
		Where(space.PublicID(entx.NewCIText(spaceID))).
		OnlyX(tenantCtx)

	// impl in enttentant.Space.Policy(); query above fails if not permission
	/*
		// check if user is assigned to space

		isAssigned := spacex.
			QueryUsers().
			Where(user.AccountID(accountm.Data.ID)).
			ExistX(tenantCtx)
		if !isAssigned {
			// rwx.AddRenderables(wx.NewSnackbarf("You are not allowed to access this space."))
			// rwx.WriteHeader(http.StatusForbidden)
			return tenantCtx, tenantTx, false, e.NewHTTPErrorf(http.StatusForbidden, "You are not allowed to access this space.")
		}
	*/

	spaceCtx := ctxx.NewSpaceContext(
		tenantCtx,
		spacex,
	)

	return spaceCtx, tenantTx, false, nil
}

var ErrSessionNotFound = errors.New("session not found")

func (qq *Router) authenticateAccount(rw httpx.ResponseWriter, req *httpx.Request, mainTx *entmain.Tx) (*account.Account, bool, error) {
	// reads only the value, all other fields have zero value
	// this is the correct behavior, as only the name and value are send via HTTP
	cookie, err := req.Cookie(cookiex.SessionCookieName())
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return nil, false, nil
		}
		return nil, false, e.NewHTTPErrorf(http.StatusBadRequest, "Could not read cookie.")
	}

	// doesn't do much because we only read the value...
	if err = cookie.Valid(); err != nil {
		cookiex.InvalidateSessionCookie(rw, qq.infra.SystemConfig().AllowInsecureCookies())
		mainTx.Session.Delete().Where(session.Value(cookie.Value)).ExecX(req.Context())
		return nil, false, e.NewHTTPErrorf(http.StatusBadRequest, "Cookie set but not valid.")
	}
	if cookie.Value == "" {
		// not sure if also checked with cookie.Valid()
		return nil, false, e.NewHTTPErrorf(http.StatusBadRequest, "Cookie set but empty.")
	}

	sessionx, err := mainTx.Session.
		Query().
		Where(
			session.Value(cookie.Value),
			session.Or(
				session.IsTemporarySession(true),
				session.ExpiresAtGT(time.Now()),
			),
		).
		Only(req.Context())
	if err != nil {
		log.Println(err)
		cookiex.InvalidateSessionCookie(
			rw,
			qq.infra.SystemConfig().AllowInsecureCookies(),
		)
		// no need to delete in db, because wasn't found...

		// TODO show message to user
		return nil, false, ErrSessionNotFound
	}

	accountx := sessionx.QueryAccount().OnlyX(req.Context())
	accountm := account.NewAccount(accountx)

	cookie, isRenewed := cookiex.RenewSessionCookie(
		rw,
		cookie.Value,
		sessionx.ExpiresAt,
		qq.infra.SystemConfig().AllowInsecureCookies(),
	)
	if isRenewed {
		// mainTx could be read only
		qq.mainDB.ReadWriteConn.Session.Update().
			SetDeletableAt(cookiex.DeletableAt(cookie)).
			SetExpiresAt(cookie.Expires).
			Where(session.Value(cookie.Value)).
			ExecX(req.Context())
	}

	return accountm, true, nil
}

/*
func (qq *Router) RegisterAction(
	action Actionable,
	handler func(http.ResponseWriter, *http.Request),
	formHandler func(http.ResponseWriter, *http.Request), // can be nil
) {
	qq.HandleFunc(action.Route(), handler)
	if formHandler != nil {
		if action.FormRoute() == "" {
			panic("form route is required, main route was " + action.Route())
		}
		qq.HandleFunc(action.FormRoute(), formHandler)
	}
}
*/
