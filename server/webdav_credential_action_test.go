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

	err = withMainContext(t, harness, accountx, func(
		mainTx *entmain.Tx,
		mainCtx *ctxx.MainContext,
	) error {
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
		formURL := "http://example.com/webdav/" + tenantx.PublicID.String() + "/" +
			spacePublicID + "/"
		emptyOverview, err := harness.actions.Dashboard.WebDAVCredentialListPartial.Widget(
			mainCtx,
			httpx.NewRequest(formReq),
			harness.actions.Dashboard.WebDAVCredentialListPartial.Data(""),
		)
		if err != nil {
			return err
		}
		emptyState, ok := emptyOverview.Child.(*widget.EmptyState)
		if !ok || len(emptyState.Actions) != 0 {
			t.Fatalf("expected empty state without an inline create action, got %#v", emptyOverview.Child)
		}

		form := url.Values{}
		form.Set("Destination", tenantx.PublicID.String()+":"+spacePublicID)
		form.Set("Label", "Office scanner")
		req := httptest.NewRequest(
			http.MethodPost,
			"/-/dashboard/create-webdav-credential-cmd",
			strings.NewReader(form.Encode()),
		)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()

		if err := harness.actions.Dashboard.CreateWebDAVCredentialCmd.Handler(
			httpx.NewResponseWriter(rr),
			httpx.NewRequest(req),
			mainCtx,
		); err != nil {
			return err
		}
		if rr.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("expected credential response to disable caching")
		}
		if !strings.Contains(rr.Body.String(), formURL) {
			t.Fatalf("expected complete URL in credential dialog")
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
		editForm := url.Values{}
		editForm.Set("CredentialPublicID", credentialx.PublicID.String())
		editForm.Set("DeviceLabel", "  Front desk scanner  ")
		editReq := httptest.NewRequest(
			http.MethodPost,
			"/-/dashboard/edit-webdav-credential-cmd",
			strings.NewReader(editForm.Encode()),
		)
		editReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if err := harness.actions.Dashboard.EditWebDAVCredentialCmd.Handler(
			httpx.NewResponseWriter(httptest.NewRecorder()),
			httpx.NewRequest(editReq),
			mainCtx,
		); err != nil {
			return err
		}
		if label := mainTx.WebDAVCredential.GetX(mainCtx, credentialx.ID).Label; label != "Front desk scanner" {
			t.Fatalf("expected edited device label, got %q", label)
		}
		form.Set("Label", "Front desk scanner")
		duplicateReq := httptest.NewRequest(
			http.MethodPost,
			"/-/dashboard/create-webdav-credential-cmd",
			strings.NewReader(form.Encode()),
		)
		duplicateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		duplicateErr := harness.actions.Dashboard.CreateWebDAVCredentialCmd.Handler(
			httpx.NewResponseWriter(httptest.NewRecorder()),
			httpx.NewRequest(duplicateReq),
			mainCtx,
		)
		var duplicateHTTPErr *e.HTTPError
		if !errors.As(duplicateErr, &duplicateHTTPErr) ||
			duplicateHTTPErr.StatusCode() != http.StatusBadRequest {
			t.Fatalf("expected duplicate label in one Space to be rejected, got %v", duplicateErr)
		}

		form.Set("Destination", tenantx.PublicID.String()+":"+archiveSpacePublicID)
		archiveReq := httptest.NewRequest(
			http.MethodPost,
			"/-/dashboard/create-webdav-credential-cmd",
			strings.NewReader(form.Encode()),
		)
		archiveReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if err := harness.actions.Dashboard.CreateWebDAVCredentialCmd.Handler(
			httpx.NewResponseWriter(httptest.NewRecorder()),
			httpx.NewRequest(archiveReq),
			mainCtx,
		); err != nil {
			return err
		}
		revokeForm := url.Values{}
		revokeForm.Set("CredentialPublicID", credentialx.PublicID.String())
		revokeReq := httptest.NewRequest(
			http.MethodPost,
			"/-/dashboard/revoke-webdav-credential-cmd",
			strings.NewReader(revokeForm.Encode()),
		)
		revokeReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if err := harness.actions.Dashboard.RevokeWebDAVCredentialCmd.Handler(
			httpx.NewResponseWriter(httptest.NewRecorder()),
			httpx.NewRequest(revokeReq),
			mainCtx,
		); err != nil {
			return err
		}

		form.Set("Destination", otherTenant.PublicID.String()+":"+otherSpacePublicID)
		blockedReq := httptest.NewRequest(
			http.MethodPost,
			"/-/dashboard/create-webdav-credential-cmd",
			strings.NewReader(form.Encode()),
		)
		blockedReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
		malformedReq := httptest.NewRequest(
			http.MethodPost,
			"/-/dashboard/create-webdav-credential-cmd",
			strings.NewReader(form.Encode()),
		)
		malformedReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		malformedErr := harness.actions.Dashboard.CreateWebDAVCredentialCmd.Handler(
			httpx.NewResponseWriter(httptest.NewRecorder()),
			httpx.NewRequest(malformedReq),
			mainCtx,
		)
		if !errors.As(malformedErr, &httpErr) || httpErr.StatusCode() != http.StatusBadRequest {
			t.Fatalf("expected bad request for malformed destination, got %v", malformedErr)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("create WebDAV credential: %v", err)
	}

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
	layout := page.(*widget.MainLayout)
	if len(layout.Navigation.FABs) != 1 {
		t.Fatalf("expected one WebDAV credential FAB, got %d", len(layout.Navigation.FABs))
	}
	listDetail := layout.Content.(*widget.ListDetailLayout)
	if len(listDetail.AppBar.Actions) != 1 ||
		listDetail.AppBar.Actions[0].(*widget.IconButton).Icon != "filter_alt" {
		t.Fatalf("expected filter icon in main app bar, got %#v", listDetail.AppBar.Actions)
	}
	overview := listDetail.List.(*widget.Container)
	tabs, ok := overview.Child.(*widget.TabBar)
	if !ok || len(tabs.Tabs) != 1 {
		t.Fatalf("expected only the active credential Space by default, got %#v", overview.Child)
	}
	content := tabs.ActiveTabContent.(*widget.ScrollableContent)
	if content.Toolbar == nil || len(content.Toolbar.Children()) != 1 {
		t.Fatalf("expected WebDAV URL toolbar, got %#v", content.Toolbar)
	}
	toolbarRow := content.Toolbar.Children()[0].(*widget.Row)
	toolbarItems := toolbarRow.Children.([]widget.IWidget)
	urlLink, ok := toolbarItems[1].(*widget.Link)
	if !ok || urlLink.CopyValue == "" {
		t.Fatalf("expected click-to-copy URL text link, got %#v", toolbarItems[1])
	}
	column := content.Children.(*widget.Column)
	list := column.Children.(*widget.List)
	items := list.Children.([]*widget.ListItem)
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
	menu := menuButton.Children.(*widget.Menu)
	if len(menu.Items) != 3 || menu.Items[0].Label.String(mainCtx) != "Edit" ||
		!menu.Items[1].IsDivider || menu.Items[2].Label.String(mainCtx) != "Revoke" {
		t.Fatalf("expected edit and revoke submenu, got %#v", menu.Items)
	}
	if supportingText := items[1].SupportingText.String(mainCtx); !strings.HasPrefix(
		supportingText,
		"Username: ",
	) || strings.Contains(supportingText, "Archive") {
		t.Fatalf("expected supporting text without Space, got %q", supportingText)
	}
	pageRR := httptest.NewRecorder()
	if err := harness.infra.Renderer().Render(
		httpx.NewResponseWriter(pageRR),
		mainCtx,
		overview,
	); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pageRR.Body.String(), "https://app.simpledms.eu/webdav/") {
		t.Fatal("expected active Space URL")
	}
	if !strings.Contains(pageRR.Body.String(), "data-copy-value") {
		t.Fatal("expected URL card to expose copy-on-click behavior")
	}
	if !strings.Contains(pageRR.Body.String(), "Copy WebDAV URL") ||
		!strings.Contains(pageRR.Body.String(), "WebDAV URL copied to clipboard.") {
		t.Fatal("expected copy tooltip and snackbar feedback")
	}

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

	revokedOverview, err := harness.actions.Dashboard.WebDAVCredentialListPartial.Widget(
		mainCtx,
		pageReq,
		harness.actions.Dashboard.WebDAVCredentialListPartial.Data(
			"",
			"revoked",
		),
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
		harness.actions.Dashboard.WebDAVCredentialListPartial.Data(
			"",
			"active",
			"revoked",
		),
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
