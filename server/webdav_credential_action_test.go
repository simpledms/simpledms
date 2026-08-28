package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestCreateWebDAVCredentialCmdUsesAuthorizedTenantContext(t *testing.T) {
	t.Setenv("SIMPLEDMS_PUBLIC_ORIGIN", "")
	harness := newActionTestHarness(t)
	const email = "webdav-settings-owner@example.com"
	accountx, tenantx := signUpAccount(t, harness, email)
	tenantDB := initTenantDB(t, harness, tenantx)

	var spacePublicID string
	var archiveSpacePublicID string
	err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(
		_ *entmain.Tx,
		_ *enttenant.Tx,
		tenantCtx *ctxx.TenantContext,
	) error {
		createSpaceViaCmd(t, harness.actions, tenantCtx, "Scanner Inbox")
		createSpaceViaCmd(t, harness.actions, tenantCtx, "Archive")
		spacePublicID = tenantCtx.TTx.Space.Query().
			Where(space.Name("Scanner Inbox")).
			OnlyX(tenantCtx).
			PublicID.String()
		archiveSpacePublicID = tenantCtx.TTx.Space.Query().
			Where(space.Name("Archive")).
			OnlyX(tenantCtx).
			PublicID.String()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	otherAccount, otherTenant := signUpAccount(t, harness, "other-webdav-owner@example.com")
	otherTenantDB := initTenantDB(t, harness, otherTenant)
	var otherSpacePublicID string
	err = withTenantContext(t, harness, otherAccount, otherTenant, otherTenantDB, func(
		_ *entmain.Tx,
		_ *enttenant.Tx,
		tenantCtx *ctxx.TenantContext,
	) error {
		createSpaceViaCmd(t, harness.actions, tenantCtx, "Other Inbox")
		otherSpacePublicID = tenantCtx.TTx.Space.Query().
			Where(space.Name("Other Inbox")).
			OnlyX(tenantCtx).
			PublicID.String()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	exerciseWebDAVCredentialActions(
		t,
		harness,
		accountx,
		tenantx,
		otherTenant,
		spacePublicID,
		archiveSpacePublicID,
		otherSpacePublicID,
	)
	assertWebDAVCredentialsPage(t, harness, email, archiveSpacePublicID)
}

func exerciseWebDAVCredentialActions(
	t *testing.T,
	harness *actionTestHarness,
	accountx *entmain.Account,
	tenantx *entmain.Tenant,
	otherTenant *entmain.Tenant,
	spacePublicID string,
	archiveSpacePublicID string,
	otherSpacePublicID string,
) {
	t.Helper()
	err := withMainContext(t, harness, accountx, func(
		mainTx *entmain.Tx,
		mainCtx *ctxx.MainContext,
	) error {
		if err := assertWebDAVCredentialForm(t, harness, mainCtx); err != nil {
			return err
		}

		form, credentialx, err := createWebDAVCredential(
			t,
			harness,
			mainTx,
			mainCtx,
			accountx,
			tenantx,
			spacePublicID,
		)
		if err != nil {
			return err
		}
		if err := editAndRejectDuplicateWebDAVCredential(
			t,
			harness,
			mainTx,
			mainCtx,
			form,
			credentialx,
		); err != nil {
			return err
		}
		if err := createArchiveAndRevokeWebDAVCredential(
			harness,
			mainCtx,
			form,
			credentialx,
			tenantx,
			archiveSpacePublicID,
		); err != nil {
			return err
		}
		rejectInvalidWebDAVCredentialDestinations(
			t,
			harness,
			mainTx,
			mainCtx,
			form,
			otherTenant,
			otherSpacePublicID,
		)
		return nil
	})
	if err != nil {
		t.Fatalf("create WebDAV credential: %v", err)
	}
}

func assertWebDAVCredentialForm(
	t *testing.T,
	harness *actionTestHarness,
	mainCtx *ctxx.MainContext,
) error {
	t.Helper()
	formReq := httptest.NewRequest(
		http.MethodPost,
		"/-/dashboard/create-webdav-credential-cmd-form?wrapper=dialog",
		nil,
	)
	formRR := httptest.NewRecorder()
	if err := harness.actions.Dashboard.CreateWebDAVCredentialCmd.FormHandler(
		httpx.NewResponseWriter(formRR),
		httpx.NewRequest(formReq),
		mainCtx,
	); err != nil {
		return err
	}
	if count := strings.Count(formRR.Body.String(), `name="Destination"`); count != 2 {
		t.Fatalf("expected two Space selector options, got %d", count)
	}

	emptyOverview, err := harness.actions.Dashboard.WebDAVCredentialListPartial.Widget(
		mainCtx,
		httpx.NewRequest(formReq),
		harness.actions.Dashboard.WebDAVCredentialListPartial.Data(""),
	)
	if err != nil {
		return err
	}
	emptyState, ok := emptyOverview.Child.(*widget.EmptyState)
	if !ok || len(emptyState.Actions) != 1 {
		t.Fatalf("expected empty state with one create action, got %#v", emptyOverview.Child)
	}
	createButton, ok := emptyState.Actions[0].(*widget.Button)
	if !ok || createButton.HxPost == "" || !createButton.LoadInPopover {
		t.Fatalf("expected create action to open the credential form, got %#v", emptyState.Actions[0])
	}
	return nil
}

func createWebDAVCredential(
	t *testing.T,
	harness *actionTestHarness,
	mainTx *entmain.Tx,
	mainCtx *ctxx.MainContext,
	accountx *entmain.Account,
	tenantx *entmain.Tenant,
	spacePublicID string,
) (url.Values, *entmain.WebDAVCredential, error) {
	t.Helper()
	form := url.Values{}
	form.Set("Destination", tenantx.PublicID.String()+":"+spacePublicID)
	form.Set("Label", "Office scanner")
	req := newWebDAVCredentialRequest("/-/dashboard/create-webdav-credential-cmd", form)
	rr := httptest.NewRecorder()

	if err := harness.actions.Dashboard.CreateWebDAVCredentialCmd.Handler(
		httpx.NewResponseWriter(rr),
		httpx.NewRequest(req),
		mainCtx,
	); err != nil {
		return nil, nil, err
	}
	if rr.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("expected credential response to disable caching")
	}
	formURL := "http://example.com/webdav/" + tenantx.PublicID.String() + "/" +
		spacePublicID + "/"
	if !strings.Contains(rr.Body.String(), formURL) {
		t.Fatal("expected complete URL in credential dialog")
	}
	if count := strings.Count(rr.Body.String(), "data-copy-value"); count != 3 {
		t.Fatalf("expected all three credential values to be copyable, got %d", count)
	}
	if count := strings.Count(rr.Body.String(), "overflow-wrap: anywhere"); count != 3 {
		t.Fatalf("expected all three credential values to wrap, got %d", count)
	}

	credentialx := mainTx.WebDAVCredential.Query().OnlyX(mainCtx)
	if credentialx.AccountID != accountx.ID || credentialx.TenantID != tenantx.ID ||
		credentialx.SpacePublicID.String() != spacePublicID {
		t.Fatal("credential was created outside the current account destination")
	}
	return form, credentialx, nil
}

func editAndRejectDuplicateWebDAVCredential(
	t *testing.T,
	harness *actionTestHarness,
	mainTx *entmain.Tx,
	mainCtx *ctxx.MainContext,
	form url.Values,
	credentialx *entmain.WebDAVCredential,
) error {
	t.Helper()
	editForm := url.Values{}
	editForm.Set("CredentialPublicID", credentialx.PublicID.String())
	editForm.Set("DeviceLabel", "  Front desk scanner  ")
	editReq := newWebDAVCredentialRequest("/-/dashboard/edit-webdav-credential-cmd", editForm)
	if err := harness.actions.Dashboard.EditWebDAVCredentialCmd.Handler(
		httpx.NewResponseWriter(httptest.NewRecorder()),
		httpx.NewRequest(editReq),
		mainCtx,
	); err != nil {
		return err
	}
	if label := mainTx.WebDAVCredential.GetX(mainCtx, credentialx.ID).Label; label !=
		"Front desk scanner" {
		t.Fatalf("expected edited device label, got %q", label)
	}

	form.Set("Label", "Front desk scanner")
	duplicateReq := newWebDAVCredentialRequest(
		"/-/dashboard/create-webdav-credential-cmd",
		form,
	)
	duplicateErr := harness.actions.Dashboard.CreateWebDAVCredentialCmd.Handler(
		httpx.NewResponseWriter(httptest.NewRecorder()),
		httpx.NewRequest(duplicateReq),
		mainCtx,
	)
	var httpErr *e.HTTPError
	if !errors.As(duplicateErr, &httpErr) || httpErr.StatusCode() != http.StatusBadRequest {
		t.Fatalf("expected duplicate label in one Space to be rejected, got %v", duplicateErr)
	}
	return nil
}

func createArchiveAndRevokeWebDAVCredential(
	harness *actionTestHarness,
	mainCtx *ctxx.MainContext,
	form url.Values,
	credentialx *entmain.WebDAVCredential,
	tenantx *entmain.Tenant,
	archiveSpacePublicID string,
) error {
	form.Set("Destination", tenantx.PublicID.String()+":"+archiveSpacePublicID)
	archiveReq := newWebDAVCredentialRequest(
		"/-/dashboard/create-webdav-credential-cmd",
		form,
	)
	if err := harness.actions.Dashboard.CreateWebDAVCredentialCmd.Handler(
		httpx.NewResponseWriter(httptest.NewRecorder()),
		httpx.NewRequest(archiveReq),
		mainCtx,
	); err != nil {
		return err
	}

	revokeForm := url.Values{}
	revokeForm.Set("CredentialPublicID", credentialx.PublicID.String())
	revokeReq := newWebDAVCredentialRequest(
		"/-/dashboard/revoke-webdav-credential-cmd",
		revokeForm,
	)
	return harness.actions.Dashboard.RevokeWebDAVCredentialCmd.Handler(
		httpx.NewResponseWriter(httptest.NewRecorder()),
		httpx.NewRequest(revokeReq),
		mainCtx,
	)
}

func rejectInvalidWebDAVCredentialDestinations(
	t *testing.T,
	harness *actionTestHarness,
	mainTx *entmain.Tx,
	mainCtx *ctxx.MainContext,
	form url.Values,
	otherTenant *entmain.Tenant,
	otherSpacePublicID string,
) {
	t.Helper()
	form.Set("Destination", otherTenant.PublicID.String()+":"+otherSpacePublicID)
	blockedReq := newWebDAVCredentialRequest(
		"/-/dashboard/create-webdav-credential-cmd",
		form,
	)
	blockedErr := harness.actions.Dashboard.CreateWebDAVCredentialCmd.Handler(
		httpx.NewResponseWriter(httptest.NewRecorder()),
		httpx.NewRequest(blockedReq),
		mainCtx,
	)
	var httpErr *e.HTTPError
	if !errors.As(blockedErr, &httpErr) || httpErr.StatusCode() != http.StatusForbidden {
		t.Fatalf("expected forbidden destination error, got %v", blockedErr)
	}
	if count := mainTx.WebDAVCredential.Query().CountX(mainCtx); count != 2 {
		t.Fatalf("expected two credentials after forbidden attempt, got %d", count)
	}

	form.Set("Destination", "malformed")
	malformedReq := newWebDAVCredentialRequest(
		"/-/dashboard/create-webdav-credential-cmd",
		form,
	)
	malformedErr := harness.actions.Dashboard.CreateWebDAVCredentialCmd.Handler(
		httpx.NewResponseWriter(httptest.NewRecorder()),
		httpx.NewRequest(malformedReq),
		mainCtx,
	)
	if !errors.As(malformedErr, &httpErr) || httpErr.StatusCode() != http.StatusBadRequest {
		t.Fatalf("expected bad request for malformed destination, got %v", malformedErr)
	}
}

func newWebDAVCredentialRequest(path string, form url.Values) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func assertWebDAVCredentialsPage(
	t *testing.T,
	harness *actionTestHarness,
	email string,
	archiveSpacePublicID string,
) {
	t.Helper()
	_, mainCtx, rollback := newNavigationRailMainContext(t, harness, email)
	defer rollback()
	pageReq := httpx.NewRequest(httptest.NewRequest(
		http.MethodGet,
		"https://app.simpledms.eu/dashboard/webdav-credentials/",
		nil,
	))
	page, err := harness.actions.Dashboard.WebDAVCredentialsPage.Widget(
		mainCtx,
		pageReq,
		harness.actions.Dashboard.WebDAVCredentialListPartial.Data(""),
	)
	if err != nil {
		t.Fatal(err)
	}

	overview := assertWebDAVPageLayout(t, page)
	assertActiveWebDAVCredentialOverview(t, mainCtx, overview, archiveSpacePublicID)
	assertRenderedWebDAVCredentialOverview(t, harness, mainCtx, overview)
	assertWebDAVCredentialFilterDialog(t, harness)
	assertFilteredWebDAVCredentialOverviews(t, harness, mainCtx, pageReq)
}

func assertWebDAVPageLayout(t *testing.T, page widget.IWidget) *widget.Container {
	t.Helper()
	layout := page.(*widget.MainLayout)
	if len(layout.Navigation.FABs) != 1 {
		t.Fatalf("expected one WebDAV credential FAB, got %d", len(layout.Navigation.FABs))
	}
	listDetail := layout.Content.(*widget.ListDetailLayout)
	if len(listDetail.AppBar.Actions) != 1 ||
		listDetail.AppBar.Actions[0].(*widget.IconButton).Icon != "filter_alt" {
		t.Fatalf("expected filter icon in main app bar, got %#v", listDetail.AppBar.Actions)
	}
	return listDetail.List.(*widget.Container)
}

func assertActiveWebDAVCredentialOverview(
	t *testing.T,
	mainCtx *ctxx.MainContext,
	overview *widget.Container,
	archiveSpacePublicID string,
) {
	t.Helper()
	tabs, ok := overview.Child.(*widget.TabBar)
	if !ok || len(tabs.Tabs) != 1 {
		t.Fatalf("expected only the active credential Space by default, got %#v", overview.Child)
	}
	content := tabs.ActiveTabContent.(*widget.ScrollableContent)
	assertWebDAVCredentialToolbar(t, content)
	column := content.Children.(*widget.Column)
	list := column.Children.(*widget.List)
	items := list.Children.([]*widget.ListItem)
	assertWebDAVCredentialItems(t, mainCtx, items, archiveSpacePublicID)
}

func assertWebDAVCredentialToolbar(t *testing.T, content *widget.ScrollableContent) {
	t.Helper()
	if content.Toolbar == nil || len(content.Toolbar.Children()) != 1 {
		t.Fatalf("expected WebDAV URL toolbar, got %#v", content.Toolbar)
	}
	toolbarRow := content.Toolbar.Children()[0].(*widget.Row)
	toolbarItems := toolbarRow.Children.([]widget.IWidget)
	urlLink, ok := toolbarItems[1].(*widget.Link)
	if !ok || urlLink.CopyValue == "" {
		t.Fatalf("expected click-to-copy URL text link, got %#v", toolbarItems[1])
	}
}

func assertWebDAVCredentialItems(
	t *testing.T,
	mainCtx *ctxx.MainContext,
	items []*widget.ListItem,
	archiveSpacePublicID string,
) {
	t.Helper()
	if len(items) != 2 || items[0].Type != widget.ListItemTypeHelper {
		t.Fatalf("expected one add row followed by one credential, got %#v", items)
	}
	if !strings.Contains(string(items[0].HxVals), archiveSpacePublicID) {
		t.Fatalf("expected add row to preselect its Space, got %s", items[0].HxVals)
	}
	menuButton, ok := items[1].Trailing.(*widget.IconButton)
	if !ok || menuButton.Icon != "more_vert" {
		t.Fatalf("expected credential overflow menu, got %#v", items[1].Trailing)
	}
	assertWebDAVCredentialMenu(t, mainCtx, items[1], menuButton)
}

func assertWebDAVCredentialMenu(
	t *testing.T,
	mainCtx *ctxx.MainContext,
	item *widget.ListItem,
	menuButton *widget.IconButton,
) {
	t.Helper()
	menu := menuButton.Children.(*widget.Menu)
	if len(menu.Items) != 3 || menu.Items[0].Label.String(mainCtx) != "Edit" ||
		!menu.Items[1].IsDivider || menu.Items[2].Label.String(mainCtx) != "Revoke" {
		t.Fatalf("expected edit and revoke submenu, got %#v", menu.Items)
	}
	supportingText := item.SupportingText.String(mainCtx)
	if !strings.HasPrefix(supportingText, "Username: ") ||
		strings.Contains(supportingText, "Archive") {
		t.Fatalf("expected supporting text without Space, got %q", supportingText)
	}
}

func assertRenderedWebDAVCredentialOverview(
	t *testing.T,
	harness *actionTestHarness,
	mainCtx *ctxx.MainContext,
	overview *widget.Container,
) {
	t.Helper()
	pageRR := httptest.NewRecorder()
	if err := harness.infra.Renderer().Render(
		httpx.NewResponseWriter(pageRR),
		mainCtx,
		overview,
	); err != nil {
		t.Fatal(err)
	}
	body := pageRR.Body.String()
	if !strings.Contains(body, "https://app.simpledms.eu/webdav/") {
		t.Fatal("expected active Space URL")
	}
	if !strings.Contains(body, "data-copy-value") {
		t.Fatal("expected URL card to expose copy-on-click behavior")
	}
	if !strings.Contains(body, "Copy WebDAV URL") ||
		!strings.Contains(body, "WebDAV URL copied to clipboard.") {
		t.Fatal("expected copy tooltip and snackbar feedback")
	}
}

func assertWebDAVCredentialFilterDialog(t *testing.T, harness *actionTestHarness) {
	t.Helper()
	filterDialog := harness.actions.Dashboard.WebDAVCredentialFilterDialog.Widget(
		harness.actions.Dashboard.WebDAVCredentialListPartial.Data(""),
	)
	filterChips := filterDialog.Child.(*widget.Container).Child.([]*widget.FilterChip)
	if filterDialog.Layout != widget.DialogLayoutSideSheet ||
		!filterChips[0].IsChecked || filterChips[1].IsChecked {
		t.Fatalf("expected active-only side-sheet filter by default, got %#v", filterChips)
	}
	if !strings.Contains(
		string(filterChips[1].HxOn.Handler),
		"credential_status', 'active",
	) {
		t.Fatal("expected selecting revoked to preserve the implicit active filter")
	}
}

func assertFilteredWebDAVCredentialOverviews(
	t *testing.T,
	harness *actionTestHarness,
	mainCtx *ctxx.MainContext,
	pageReq *httpx.Request,
) {
	t.Helper()
	revokedOverview, err := harness.actions.Dashboard.WebDAVCredentialListPartial.Widget(
		mainCtx,
		pageReq,
		harness.actions.Dashboard.WebDAVCredentialListPartial.Data("", "revoked"),
	)
	if err != nil {
		t.Fatal(err)
	}
	revokedTabs := revokedOverview.Child.(*widget.TabBar)
	if len(revokedTabs.Tabs) != 1 {
		t.Fatalf("expected one revoked credential Space, got %d", len(revokedTabs.Tabs))
	}
	revokedContent := revokedTabs.ActiveTabContent.(*widget.ScrollableContent)
	revokedList := revokedContent.Children.(*widget.Column).Children.(*widget.List)
	revokedItems := revokedList.Children.([]*widget.ListItem)
	if len(revokedItems) != 2 || revokedItems[1].Trailing != nil {
		t.Fatalf("expected one revoked credential without revoke menu, got %#v", revokedItems)
	}

	allOverview, err := harness.actions.Dashboard.WebDAVCredentialListPartial.Widget(
		mainCtx,
		pageReq,
		harness.actions.Dashboard.WebDAVCredentialListPartial.Data("", "active", "revoked"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if tabs := allOverview.Child.(*widget.TabBar); len(tabs.Tabs) != 2 {
		t.Fatalf("expected active and revoked credentials across two Spaces, got %d", len(tabs.Tabs))
	}

	_, err = harness.actions.Dashboard.WebDAVCredentialListPartial.Widget(
		mainCtx,
		pageReq,
		harness.actions.Dashboard.WebDAVCredentialListPartial.Data("", "invalid"),
	)
	var filterErr *e.HTTPError
	if !errors.As(err, &filterErr) || filterErr.StatusCode() != http.StatusBadRequest {
		t.Fatalf("expected invalid credential status to be rejected, got %v", err)
	}
}
