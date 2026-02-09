package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"filippo.io/age"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/gorilla/handlers"
	"github.com/marcobeierer/go-tika"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"golang.org/x/crypto/acme/autocert"

	"github.com/simpledms/simpledms/action"
	"github.com/simpledms/simpledms/action/download"
	trashaction "github.com/simpledms/simpledms/action/trash"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/migrate"
	"github.com/simpledms/simpledms/db/entmain/systemconfig"
	"github.com/simpledms/simpledms/db/entmain/tenant"
	migrate2 "github.com/simpledms/simpledms/db/enttenant/migrate"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/encryptor"
	"github.com/simpledms/simpledms/i18n"
	"github.com/simpledms/simpledms/model/common/country"
	"github.com/simpledms/simpledms/model/common/language"
	"github.com/simpledms/simpledms/model/common/mainrole"
	"github.com/simpledms/simpledms/model/filesystem"
	"github.com/simpledms/simpledms/model/modelmain"
	tenant2 "github.com/simpledms/simpledms/model/tenant"
	"github.com/simpledms/simpledms/scheduler"
	"github.com/simpledms/simpledms/ui"
	"github.com/simpledms/simpledms/ui/uix/partial"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/recoverx"
)

// TODO move to own package in cmd?
type Server struct {
	metaPath                 string
	devMode                  bool
	unsafePort               int // unsafe because it can be 0, use qq.port()
	assetsFS                 fs.FS
	migrationsMainFS         fs.FS
	migrationsTenantFS       fs.FS
	isSaaSModeEnabled        bool
	commercialLicenseEnabled bool
}

type listenMode int

const (
	listenModeHTTP listenMode = iota
	listenModeTLSFiles
	listenModeTLSAutocert
)

func shouldUseAutocert(enableAutocert, devMode bool) bool {
	return enableAutocert && !devMode
}

func resolveListenMode(useAutocert bool, tlsCertFilepath, tlsPrivateKeyFilepath string) listenMode {
	if useAutocert {
		return listenModeTLSAutocert
	}
	if tlsCertFilepath == "" || tlsPrivateKeyFilepath == "" {
		return listenModeHTTP
	}
	return listenModeTLSFiles
}

func newMaintenanceModeHandler(
	mainDB *sqlx.MainDB,
	assetsFS fs.FS,
	i18nx *i18n.I18n,
	renderer *ui.Renderer,
	encryptedIdentity []byte,
	commercialLicenseEnabled bool,
	shutdownFn func(context.Context) error,
) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assetsFS))))

	mux.HandleFunc("/-/unlock-cmd", func(rw http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()

		var reqBody struct {
			Passphrase string `json:"passphrase"`
		}

		err := json.NewDecoder(req.Body).Decode(&reqBody)
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusBadRequest)
			_, _ = rw.Write([]byte("Invalid request payload"))
			return
		}

		passphrase := reqBody.Passphrase
		if passphrase == "" {
			rw.WriteHeader(http.StatusBadRequest)
			_, _ = rw.Write([]byte("Passphrase is required"))
			return
		}

		identity, err := modelmain.DecryptMainIdentity(encryptedIdentity, passphrase)
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusBadRequest)
			_, _ = rw.Write([]byte("Invalid passphrase"))
			return
		}

		encryptor.NilableX25519MainIdentity = identity

		if shutdownFn != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err = shutdownFn(ctx)
			if err != nil {
				log.Println(err)
			}
		}
	})

	// TODO recovery handler
	// TODO status code?
	mux.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		mainTx, err := mainDB.Tx(req.Context(), true)
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer func() {
			err = mainTx.Commit()
			if err != nil {
				log.Println(err)
			}
		}()

		visitorCtx := ctxx.NewVisitorContext(
			req.Context(),
			mainTx,
			i18nx,
			req.Header.Get("Accept-Language"),
			req.Header.Get("X-Client-Timezone"),
			false,
			commercialLicenseEnabled,
		)

		titlex := wx.Tuf("%s | SimpleDMS", wx.T("Maintenance mode").String(visitorCtx))
		viewx := partial.NewBase(
			titlex,
			&wx.MainLayout{
				Content: &wx.NarrowLayout{
					Content: &wx.Column{
						GapYSize:         wx.Gap4,
						NoOverflowHidden: true,
						Children: []wx.IWidget{
							wx.H(wx.HeadingTypeHeadlineMd, titlex),
							wx.T("Maintenance mode is enabled. Please wait until the app ready again.").SetWrap(),
							// wx.T("This page automatically refreshes every 60 seconds.").SetWrap(),
						},
					},
				},
			},
		)
		viewx.ShouldRefreshEvery60Seconds = true

		rwx := httpx.NewResponseWriter(rw)
		rwx.WriteHeader(http.StatusServiceUnavailable) // must be before render

		err = renderer.Render(rwx, visitorCtx, viewx)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
			return
		}
	})

	return mux
}

func NewServer(
	metaPath string,
	devMode bool,
	unsafePort int,
	assetsFS fs.FS,
	isSaaSModeEnabled bool,
	commercialLicenseEnabled bool,
) *Server {
	// TODO should path outside PWD be allowed? probably if someone likes
	//		to manage all db files / meta data centrally
	metaPath = filepath.Clean(metaPath)

	pwd, err := os.Getwd()
	if err != nil {
		log.Println(err)
		panic(err)
	}
	metaPath, err = securejoin.SecureJoin(pwd, metaPath)
	if err != nil {
		log.Println(err)
		panic(err)
	}
	err = os.MkdirAll(metaPath, 0777)
	if err != nil {
		log.Fatalln(err)
	}

	migrationsMainFS, err := migrate.NewMigrationsMainFS()
	if err != nil {
		log.Fatalln(err)
	}
	migrationsTenantFS, err := migrate2.NewMigrationsTenantFS()
	if err != nil {
		log.Fatalln(err)
	}

	return &Server{
		metaPath:                 metaPath,
		devMode:                  devMode,
		unsafePort:               unsafePort,
		assetsFS:                 assetsFS,
		migrationsMainFS:         migrationsMainFS,
		migrationsTenantFS:       migrationsTenantFS,
		isSaaSModeEnabled:        isSaaSModeEnabled,
		commercialLicenseEnabled: commercialLicenseEnabled,
	}
}

// TODO way to long, needs refactoring
func (qq *Server) Start() error {
	var err error

	// TODO close all clients
	mainDB := dbMigrationsMainDB(qq.devMode, qq.metaPath, qq.migrationsMainFS)
	// FIXME is it okay to keep all databases open all the time?
	defer func() {
		err = mainDB.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	overrideDBConfig := os.Getenv("SIMPLEDMS_OVERRIDE_DB_CONFIG") == "true"

	ctx := context.Background()
	configCount := mainDB.ReadOnlyConn.SystemConfig.Query().CountX(ctx)
	if configCount == 0 {
		mailerPort := 25
		mailerPortStr := os.Getenv("SIMPLEDMS_MAILER_PORT")
		if mailerPortStr != "" {
			mailerPort, err = strconv.Atoi(mailerPortStr)
			if err != nil {
				log.Fatalln(err)
			}
		}
		initAppTx, err := mainDB.Tx(ctx, false)
		if err != nil {
			log.Fatalln(err)
		}
		err = modelmain.InitAppWithoutCustomContext(
			ctx,
			initAppTx,
			// true,
			"",
			true,
			modelmain.S3Config{
				S3Endpoint:        os.Getenv("SIMPLEDMS_S3_ENDPOINT"),
				S3AccessKeyID:     os.Getenv("SIMPLEDMS_S3_ACCESS_KEY_ID"),
				S3SecretAccessKey: os.Getenv("SIMPLEDMS_S3_SECRET_ACCESS_KEY"),
				S3BucketName:      os.Getenv("SIMPLEDMS_S3_BUCKET_NAME"),
				S3UseSSL:          os.Getenv("SIMPLEDMS_S3_USE_SSL") == "true",
			},
			modelmain.TLSConfig{
				TLSEnableAutocert:     os.Getenv("SIMPLEDMS_TLS_ENABLE_AUTOCERT") == "true",
				TLSCertFilepath:       os.Getenv("SIMPLEDMS_TLS_CERT_FILEPATH"),
				TLSPrivateKeyFilepath: os.Getenv("SIMPLEDMS_TLS_PRIVATE_KEY_FILEPATH"),
				TLSAutocertEmail:      os.Getenv("SIMPLEDMS_TLS_AUTOCERT_EMAIL"),
				TLSAutocertHosts:      strings.Split(os.Getenv("SIMPLEDMS_TLS_AUTOCERT_HOSTS"), ","),
			},
			modelmain.MailerConfig{
				MailerHost:               os.Getenv("SIMPLEDMS_MAILER_HOST"),
				MailerPort:               mailerPort,
				MailerUsername:           os.Getenv("SIMPLEDMS_MAILER_USERNAME"),
				MailerPassword:           os.Getenv("SIMPLEDMS_MAILER_PASSWORD"),
				MailerFrom:               os.Getenv("SIMPLEDMS_MAILER_FROM"),
				MailerInsecureSkipVerify: os.Getenv("SIMPLEDMS_MAILER_INSECURE_SKIP_VERIFY") == "true",
				MailerUseImplicitSSLTLS:  os.Getenv("SIMPLEDMS_MAILER_USE_IMPLICIT_SSL_TLS") == "true",
			},
			modelmain.OCRConfig{
				TikaURL: os.Getenv("SIMPLEDMS_OCR_TIKA_URL"),
			},
		)
		if err != nil {
			erry := initAppTx.Rollback()
			if erry != nil {
				log.Println(erry)
			}
			log.Fatalln(err)
		}
		err = initAppTx.Commit()
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		if overrideDBConfig {
			// IMPORTANT
			// only TLS is overridden here because all encrypted fields can just
			// be overridden when encryptor.NilableX25519MainIdentity is set;
			// TLS config is read before that is the case
			// END IMPORTANT

			updateQuery := mainDB.ReadWriteConn.SystemConfig.Query().FirstX(ctx).Update()

			if val, set := os.LookupEnv("SIMPLEDMS_TLS_ENABLE_AUTOCERT"); set {
				updateQuery.SetTLSEnableAutocert(val == "true")
			}
			if val, set := os.LookupEnv("SIMPLEDMS_TLS_CERT_FILEPATH"); set {
				updateQuery.SetTLSCertFilepath(val)
			}
			if val, set := os.LookupEnv("SIMPLEDMS_TLS_PRIVATE_KEY_FILEPATH"); set {
				updateQuery.SetTLSPrivateKeyFilepath(val)
			}
			if val, set := os.LookupEnv("SIMPLEDMS_TLS_AUTOCERT_EMAIL"); set {
				updateQuery.SetTLSAutocertEmail(val)
			}
			if val, set := os.LookupEnv("SIMPLEDMS_TLS_AUTOCERT_HOSTS"); set {
				updateQuery.SetTLSAutocertHosts(strings.Split(val, ","))
			}

			updateQuery.SaveX(ctx)
		}
	}

	initialAccountEmail := os.Getenv("SIMPLEDMS_INITIAL_ACCOUNT_EMAIL")
	initialTemporaryPassword := os.Getenv("SIMPLEDMS_INITIAL_TEMPORARY_PASSWORD")
	initialTenantName := os.Getenv("SIMPLEDMS_INITIAL_TENANT_NAME")
	if initialAccountEmail != "" && initialTenantName != "" {
		if mainDB.ReadOnlyConn.Account.Query().CountX(ctx) > 0 {
			log.Println("an account already exists, skipping creation of initial account")
		} else {
			err = qq.initInitialUser(
				ctx,
				mainDB,
				initialAccountEmail,
				initialTemporaryPassword,
				initialTenantName,
			)
			if err != nil {
				log.Fatalln(err)
			}
		}
	}

	// TODO are there any naming conflicts?
	templates := template.New("app")

	templates.Funcs(ui.TemplateFuncMap(templates))
	templates, err = templates.ParseFS(ui.WidgetFS, "widget/*.gohtml") // , "widget/*.tw.gohtml") // TODO was ui/widget/*.gohtml
	if err != nil {
		log.Fatal(err)
	}
	renderer := ui.NewRenderer(templates)
	i18nx := i18n.NewI18n()

	/*assetsFS, err := fs.Sub(qq.assetsFS, "ui/web/assets")
	if err != nil {
		log.Fatal(err)
	}*/

	// partial request because encrypted fields cannot be decrypted before
	// encryptor.NilableX25519MainIdentity is set
	// TODO FirstX okay?
	systemConfigx := mainDB.ReadOnlyConn.SystemConfig.Query().
		Select(
			systemconfig.FieldX25519Identity,
			systemconfig.FieldIsIdentityEncryptedWithPassphrase,
			systemconfig.FieldTLSEnableAutocert,
			systemconfig.FieldTLSCertFilepath,
			systemconfig.FieldTLSPrivateKeyFilepath,
			systemconfig.FieldTLSAutocertEmail,
			systemconfig.FieldTLSAutocertHosts,
		).
		FirstX(ctx)

	var manager *autocert.Manager
	useAutocert := shouldUseAutocert(systemConfigx.TLSEnableAutocert, qq.devMode)

	// autocert server runs always, in maintenance mode and normal mode
	if useAutocert {
		manager = &autocert.Manager{
			Cache:      autocert.DirCache(qq.metaPath + "/autocert"),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(systemConfigx.TLSAutocertHosts...),
			Email:      systemConfigx.TLSAutocertEmail,
		}

		go func() {
			recoverx.Recover("autocert server")

			errx := http.ListenAndServe(":http", manager.HTTPHandler(nil))
			if errx != nil {
				log.Println(errx)
			}
		}()
	}

	if systemConfigx.IsIdentityEncryptedWithPassphrase {
		maintenanceModeServer := http.Server{
			Addr: fmt.Sprintf(":%d", qq.port(
				useAutocert,
				systemConfigx.TLSCertFilepath,
				systemConfigx.TLSPrivateKeyFilepath,
			)),
		}

		maintenanceMux := newMaintenanceModeHandler(
			mainDB,
			qq.assetsFS,
			i18nx,
			renderer,
			systemConfigx.X25519Identity,
			qq.commercialLicenseEnabled,
			maintenanceModeServer.Shutdown,
		)

		handlerChain := handlers.CompressHandler(
			handlers.RecoveryHandler(
				handlers.PrintRecoveryStack(true),
			)(
				maintenanceMux,
			),
		)

		maintenanceModeServer.Handler = handlerChain

		maintenanceListenMode := resolveListenMode(
			useAutocert,
			systemConfigx.TLSCertFilepath,
			systemConfigx.TLSPrivateKeyFilepath,
		)

		switch maintenanceListenMode {
		case listenModeTLSAutocert:
			maintenanceModeServer.TLSConfig = &tls.Config{GetCertificate: manager.GetCertificate}
			err = maintenanceModeServer.ListenAndServeTLS("", "")
		case listenModeHTTP:
			err = maintenanceModeServer.ListenAndServe()
		case listenModeTLSFiles:
			err = maintenanceModeServer.ListenAndServeTLS(
				systemConfigx.TLSCertFilepath,
				systemConfigx.TLSPrivateKeyFilepath,
			)
		default:
			log.Fatalln("unknown maintenance listen mode")
		}
		if err != nil {
			// err is set if the server gets shutdown on unlock, thus no aborting
			log.Println(err)
		}

		// TODO maintenance mode and wait for unlock
		// log.Fatalln("identity encrypted with passphrase")
	} else {
		identityBytes := systemConfigx.X25519Identity
		if len(identityBytes) == 0 {
			// TODO init or maintenance mode?
			log.Fatalln("no identity")
		}
		x25519Identity, err := age.ParseX25519Identity(string(identityBytes))
		if err != nil {
			log.Fatalln(err, "could not parse identity")
		}

		encryptor.NilableX25519MainIdentity = x25519Identity
	}

	if overrideDBConfig {
		// TLS config is processed above because required earlier;
		// It's important that this is after encryptor.NilableX25519MainIdentity
		// is set

		updateQuery := mainDB.ReadWriteConn.SystemConfig.Query().FirstX(ctx).Update()

		if val, set := os.LookupEnv("SIMPLEDMS_S3_ENDPOINT"); set {
			updateQuery.SetS3Endpoint(val)
		}
		if val, set := os.LookupEnv("SIMPLEDMS_S3_ACCESS_KEY_ID"); set {
			updateQuery.SetS3AccessKeyID(val)
		}
		if val, set := os.LookupEnv("SIMPLEDMS_S3_SECRET_ACCESS_KEY"); set {
			updateQuery.SetS3SecretAccessKey(entx.NewEncryptedString(val))
		}
		if val, set := os.LookupEnv("SIMPLEDMS_S3_BUCKET_NAME"); set {
			updateQuery.SetS3BucketName(val)
		}
		if val, set := os.LookupEnv("SIMPLEDMS_S3_USE_SSL"); set {
			updateQuery.SetS3UseSsl(val == "true")
		}

		if val, set := os.LookupEnv("SIMPLEDMS_MAILER_HOST"); set {
			updateQuery.SetMailerHost(val)
		}
		if val, set := os.LookupEnv("SIMPLEDMS_MAILER_PORT"); set {
			mailerPort, err := strconv.Atoi(val)
			if err != nil {
				log.Fatalln(err)
			}
			updateQuery.SetMailerPort(mailerPort)
		}
		if val, set := os.LookupEnv("SIMPLEDMS_MAILER_USERNAME"); set {
			updateQuery.SetMailerUsername(val)
		}
		if val, set := os.LookupEnv("SIMPLEDMS_MAILER_PASSWORD"); set {
			updateQuery.SetMailerPassword(entx.NewEncryptedString(val))
		}
		if val, set := os.LookupEnv("SIMPLEDMS_MAILER_FROM"); set {
			updateQuery.SetMailerFrom(val)
		}
		if val, set := os.LookupEnv("SIMPLEDMS_MAILER_INSECURE_SKIP_VERIFY"); set {
			updateQuery.SetMailerInsecureSkipVerify(val == "true")
		}
		if val, set := os.LookupEnv("SIMPLEDMS_MAILER_USE_IMPLICIT_SSL_TLS"); set {
			updateQuery.SetMailerUseImplicitSslTLS(val == "true")
		}

		if val, set := os.LookupEnv("SIMPLEDMS_OCR_TIKA_URL"); set {
			updateQuery.SetOcrTikaURL(val)
		}

		updateQuery.SaveX(ctx)
	}

	allowInsecureCookiesStr := os.Getenv("SIMPLEDMS_ALLOW_INSECURE_COOKIES")
	allowInsecureCookies := false
	if allowInsecureCookiesStr != "" {
		allowInsecureCookies, err = strconv.ParseBool(allowInsecureCookiesStr)
		if err != nil {
			log.Fatalln(err)
		}
	}

	// TODO FirstX okay?
	systemConfigx = mainDB.ReadOnlyConn.SystemConfig.Query().FirstX(ctx)
	systemConfig := modelmain.NewSystemConfig(
		systemConfigx,
		qq.isSaaSModeEnabled,
		qq.commercialLicenseEnabled,
		allowInsecureCookies,
	)

	tenantDBs := dbMigrationsTenantDBs(mainDB, qq.devMode, qq.metaPath)
	defer func() {
		// must run over tenantDBs at time of closing, thus in wrapper func
		tenantDBs.Range(func(tenantID int64, tenantDB *sqlx.TenantDB) bool {
			err = tenantDB.Close()
			if err != nil {
				log.Println(err)
			}
			return true
		})
	}()

	factory := common.NewFactory(
		// client.FileInfo.Query().Where(fileinfo.FullPath(common.InboxPath(metaPath))).OnlyX(context.Background()),
		// client.FileInfo.Query().Where(fileinfo.FullPath(common.StoragePath(metaPath))).OnlyX(context.Background()),
	)
	// storagePath := common.StoragePath(metaPath)
	fileRepo := common.NewFileRepository()
	minioClient := qq.initNilableMinioClient(systemConfig.S3())
	fileSystem := filesystem.NewFileSystem(qq.metaPath)

	disableFileEncryptionStr := os.Getenv("SIMPLEDMS_DISABLE_FILE_ENCRYPTION")
	disableFileEncryption := false
	if disableFileEncryptionStr != "" {
		disableFileEncryption, err = strconv.ParseBool(disableFileEncryptionStr)
		if err != nil {
			log.Fatalln(err)
		}
	}

	infra := common.NewInfra(
		renderer,
		qq.metaPath,
		filesystem.NewS3FileSystem(
			minioClient,
			systemConfig.S3().S3BucketName,
			fileSystem,
			disableFileEncryption,
		),
		factory,
		fileRepo,
		systemConfig,
	)
	router := NewRouter(mainDB, tenantDBs, infra, qq.devMode, qq.metaPath, i18nx)
	actions := action.NewActions(infra, tenantDBs)
	downloadHandler := download.NewDownload(infra)
	trashDownloadHandler := trashaction.NewDownload(infra)

	/*
		indexer := internal.NewFileIndexer(client, infra)
		go func() {
			pwd, err := os.Getwd()
			if err != nil {
				log.Fatalln(err)
			}
			indexer.Index(context.Background(), pwd)
		}()
	*/

	// concept:
	// rpc style API;
	// use GET for all read requests and POST for all write requests?
	// restful for default CRUD on main resource? falls apart quickly, see AddConsumption

	// workaround to prevent route conflict between `GET /` and `/webdav/`
	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == "GET" && req.URL.Path == "/" {
			// router.wrapTx(pages.Browse.Handler)(rw, req)
			// router.wrapTx(actions.Spaces.SpacesPage.Handler)(rw, req)
			router.wrapTx(actions.Auth.SignInPage.Handler, true)(rw, req)
			return
		}
		rw.WriteHeader(http.StatusNotFound)
	})

	// TODO find a better way to handle paths
	// TODO in TTx or not necessary because read only?
	router.RegisterPage(route2.DashboardRoute(), actions.Dashboard.DashboardPage.Handler)
	router.RegisterPage(route2.StaticPageRoute(), actions.StaticPage.StaticPage.Handler)

	router.RegisterPage(route2.BrowseRoute(false), actions.Browse.BrowsePage.Handler)
	router.RegisterPage(route2.BrowseRoute(true), actions.Browse.BrowsePage.Handler)
	router.RegisterPage(route2.BrowseRouteWithSelection(), actions.Browse.BrowseWithSelectionPage.Handler)
	// router.RegisterPage(route.BrowseRouteWithSelection(false), pages.BrowseWithSelection.Handler)

	router.RegisterPage(route2.InboxRoute(false, false), actions.Inbox.InboxRootPage.Handler)
	router.RegisterPage(route2.InboxRoute(true, false), actions.Inbox.InboxWithSelectionPage.Handler)
	// for use with PWA share target
	// router.RegisterPage(route.InboxRoute(false, true), pages.Inbox.Handler)

	router.RegisterPage(route2.TrashRoute(), actions.Trash.TrashRootPage.Handler)
	router.RegisterPage(route2.TrashRouteWithSelection(), actions.Trash.TrashWithSelectionPage.Handler)

	router.RegisterPage(route2.SpacesRoute(), actions.Spaces.SpacesPage.Handler)

	router.RegisterPage(route2.ManageDocumentTypesRoute(), actions.DocumentType.ManageDocumentTypesPage.Handler)
	router.RegisterPage(route2.ManageDocumentTypesRouteWithSelection(), actions.DocumentType.ManageDocumentTypesPage.Handler)

	router.RegisterPage(route2.ManageTagsRoute(), actions.ManageTags.ManageTagsPage.Handler)
	// router.RegisterPage(route.ManageTagsRouteWithSelection(), actions.Tagging.ManageTagsPage.Handler)

	router.RegisterPage(route2.PropertiesRoute(), actions.Property.PropertiesPage.Handler)
	router.RegisterPage(route2.ManageUsersOfSpaceRoute(), actions.ManageSpaceUsers.ManageUsersOfSpacePage.Handler)
	router.RegisterPage(route2.ManageUsersOfTenantRoute(), actions.ManageTenantUsers.ManageUsersOfTenantPage.Handler)

	router.RegisterPage(route2.SelectSpaceRoute(false), actions.OpenFile.SelectSpacePage.Handler)
	router.RegisterPage(route2.SelectSpaceRoute(true), actions.OpenFile.SelectSpacePage.Handler)

	// router.RegisterPage(route.FindRoute(false), actions.Find.Page.Handler)
	// router.RegisterPage(route.FindRoute(true), actions.Find.PageWithSelection.Handler)

	router.RegisterPage(route2.DownloadRoute(), downloadHandler.Handler)
	router.RegisterPage(route2.TrashDownloadRoute(), trashDownloadHandler.Handler)

	router.RegisterActions(actions)

	// router.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./webapp/assets"))))
	// slash suffix is necessary to match all paths with the prefix
	router.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(qq.assetsFS))))

	/*
		// mounting also works with `/webdav`, but if `/webdav` is defined
		// as route, it would work...
		router.Handle("/webdav/", &webdav.Handler{
			Prefix:     "/webdav/",
			FileSystem: webdavx.NewDir(infra, "."),
			LockSystem: webdav.NewMemLS(),
			Logger: func(request *http.Request, err error) {
				// log.Println(request)
				// log.Println(err)
			},
		})
	*/

	tenantsInMaintenanceMode := mainDB.ReadOnlyConn.Tenant.Query().Where(tenant.MaintenanceModeEnabledAtNotNil()).CountX(ctx)
	if tenantsInMaintenanceMode > 0 {
		// TODO abort??
		msg := `

WARNING 
there are tenants in maintenance mode;
the database migrations won't run for them; 
this must be fixed manually;
END WARNING
`
		if qq.devMode {
			log.Fatalln(msg)
		}
		log.Println(msg)
	}

	// migrate all existing tenants to the newest db version
	tenantsToMigrate := mainDB.ReadWriteConn.Tenant.Query().Where(tenant.MaintenanceModeEnabledAtIsNil()).AllX(ctx)
	// FIXME enable only if migration is required... version can be read with migx.Version()
	mainDB.ReadWriteConn.Tenant.Update().
		SetMaintenanceModeEnabledAt(time.Now()).
		Where(tenant.MaintenanceModeEnabledAtIsNil()).
		ExecX(ctx)
	for _, tenantx := range tenantsToMigrate {
		tenantm := tenant2.NewTenant(tenantx)
		tenantDB, found := tenantDBs.Load(tenantm.Data.ID)
		if !found {
			// could happen if initialization fails; retries initialization automatically,
			log.Println("tenant DB not found, could happen if initialization fails")
			continue
		}

		err = tenantm.ExecuteDBMigrations(qq.devMode, qq.metaPath, qq.migrationsTenantFS, tenantDB)
		if err != nil {
			// TODO make this more robust, maybe continue and deactivate tenant till restart or fixed
			log.Println(err, "; tenant is in maintenance mode now, must be fixed manually")
			continue
		}

		tenantx.
			Update().
			ClearMaintenanceModeEnabledAt().
			ExecX(ctx)
	}

	var tikaClientNilable *tika.Client
	if systemConfigx.OcrTikaURL != "" {
		tikaClientNilable = tika.NewDefaultClient(systemConfigx.OcrTikaURL)
	}

	schedulerx := scheduler.NewScheduler(
		infra,
		mainDB,
		tenantDBs,
		minioClient,
		systemConfig.S3().S3BucketName,
		tikaClientNilable,
	)
	schedulerx.Run(qq.devMode, qq.metaPath, qq.migrationsTenantFS)

	handlerChain := handlers.CompressHandler(
		handlers.RecoveryHandler(
			handlers.PrintRecoveryStack(true),
		)(
			// see https://words.filippo.io/csrf/ for implementation details
			http.NewCrossOriginProtection().Handler(router),
			// handlers.LoggingHandler(),
		),
	)

	mainListenMode := resolveListenMode(
		useAutocert,
		systemConfig.TLS().TLSCertFilepath,
		systemConfig.TLS().TLSPrivateKeyFilepath,
	)

	switch mainListenMode {
	case listenModeTLSAutocert:
		server := &http.Server{
			Addr: fmt.Sprintf(":%d", qq.port(
				useAutocert,
				systemConfig.TLS().TLSCertFilepath,
				systemConfig.TLS().TLSPrivateKeyFilepath,
			)),
			TLSConfig: &tls.Config{GetCertificate: manager.GetCertificate},
			Handler:   handlerChain,
		}
		err = server.ListenAndServeTLS("", "")
	case listenModeHTTP:
		err = http.ListenAndServe(
			fmt.Sprintf(":%d", qq.port(
				useAutocert,
				systemConfig.TLS().TLSCertFilepath,
				systemConfig.TLS().TLSPrivateKeyFilepath,
			)),
			handlerChain,
		)
	case listenModeTLSFiles:
		err = http.ListenAndServeTLS(
			fmt.Sprintf(":%d", qq.port(
				useAutocert,
				systemConfig.TLS().TLSCertFilepath,
				systemConfig.TLS().TLSPrivateKeyFilepath,
			)),
			systemConfig.TLS().TLSCertFilepath,
			systemConfig.TLS().TLSPrivateKeyFilepath,
			handlerChain,
		)
	default:
		log.Fatalln("unknown main listen mode")
	}
	if err != nil {
		log.Fatalln(err)
	}

	return nil
}

func (qq *Server) initNilableMinioClient(config *modelmain.S3Config) *minio.Client {
	if config.S3Endpoint == "" {
		log.Println("No storage endpoint configured")
		return nil
	}

	client, err := minio.New(
		config.S3Endpoint,
		&minio.Options{
			Creds: credentials.NewStaticV4(
				config.S3AccessKeyID,
				config.S3SecretAccessKey,
				"",
			),
			Secure: config.S3UseSSL,
			// TODO add region?
		})
	if err != nil {
		log.Fatalln(err)
	}

	if config.S3BucketName == "" {
		log.Fatalln("No storage bucket configured")
	}

	exists, err := client.BucketExists(context.Background(), config.S3BucketName)
	if err != nil {
		log.Fatalln(err)
	}
	if exists {
		return client
	}

	err = client.MakeBucket(context.Background(), config.S3BucketName, minio.MakeBucketOptions{
		// enables retention settings for bucket and legal hold;
		// if understood correctly, does nothing on itself
		// https://min.io/docs/minio/linux/administration/object-management/object-retention.html#minio-object-locking-retention-modes
		ObjectLocking: true,
		// TODO retention?
	})
	if err != nil {
		log.Fatalln(err)
	}

	return client
}

func (qq *Server) initInitialUser(
	ctx context.Context,
	mainDB *sqlx.MainDB,
	initialAccountEmail string,
	initialTemporaryPassword string,
	initialTenantName string,
) error {
	createInitialUserTx, err := mainDB.Tx(ctx, false)
	if err != nil {
		log.Println(err)
		return err
	}
	defer func() {
		if err != nil {
			log.Println(err)
			erry := createInitialUserTx.Rollback()
			if erry != nil {
				log.Fatalln(erry)
			}
		}
	}()

	visitorCtx := ctxx.NewVisitorContext(
		ctx,
		createInitialUserTx,
		i18n.NewI18n(),
		// TODO are there better defaults? or provide config?
		"en",
		"UTC",
		false,
		qq.commercialLicenseEnabled,
	)

	skipSendingMail := initialTemporaryPassword != ""

	initialAccount, err := modelmain.NewSignUpService().SignUp(
		visitorCtx,
		initialAccountEmail,
		initialTenantName,
		"",
		"",
		country.Unknown,
		language.Unknown,
		false,
		skipSendingMail,
	)
	if err != nil {
		log.Println(err)
		return err
	}

	err = initialAccount.Data.Update().SetRole(mainrole.Admin).Exec(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	if initialTemporaryPassword != "" {
		_, err = initialAccount.SetTemporaryPassword(visitorCtx, initialTemporaryPassword)
		if err != nil {
			log.Println(err)
			return err
		}
		log.Println("the initial temporary password you provided was set")
	} else {
		log.Println("the initial temporary password was generated and sent to the provided mail address")
	}

	err = createInitialUserTx.Commit()
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (qq *Server) port(useAutocert bool, tlsCertFilepath, tlsPrivateKeyFilepath string) int {
	if qq.unsafePort > 0 {
		return qq.unsafePort
	}
	if useAutocert || (tlsCertFilepath != "" && tlsPrivateKeyFilepath != "") {
		return 443
	}
	return 80
}
