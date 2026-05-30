package server

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/country"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	"github.com/simpledms/simpledms/model/main/common/plan"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	wx "github.com/simpledms/simpledms/ui/widget"
)

func TestNavigationRailShowsMainDestinations(t *testing.T) {
	harness := newActionTestHarness(t)

	createAccountWithRole(t, harness.mainDB, "rail-user@example.com", "supersecret", mainrole.User)
	_, userCtx, userRollback := newNavigationRailMainContext(t, harness, "rail-user@example.com")
	defer userRollback()

	userRail := partial2.NewNavigationRail(userCtx, harness.infra, "dashboard", nil)
	assertNavigationRailLabelsContain(t, userRail.GetItems(), "Dashboard", "Account")
	assertNavigationRailLabelsExclude(t, userRail.GetItems(), "System", "Users")
	assertNavigationRailLabelsContain(t, userRail.FooterItems, "Misc", "Sign out", "About SimpleDMS")
	assertNavigationRailItemActive(t, userRail.GetItems(), "Dashboard")

	createAccountWithRole(t, harness.mainDB, "rail-admin@example.com", "supersecret", mainrole.Admin)
	_, adminCtx, adminRollback := newNavigationRailMainContext(t, harness, "rail-admin@example.com")
	defer adminRollback()

	adminRail := partial2.NewNavigationRail(adminCtx, harness.infra, "system", nil)
	assertNavigationRailLabelsContain(t, adminRail.GetItems(), "Dashboard", "Account", "System")
	assertNavigationRailItemActive(t, adminRail.GetItems(), "System")
}

func TestNavigationRailShowsTenantUserDestinationAndSections(t *testing.T) {
	harness := newActionTestHarness(t)

	_, tenantx := signUpAccount(t, harness, "rail-tenant-owner@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)

	_, ownerCtx, ownerRollback := newNavigationRailMainContext(
		t,
		harness,
		"rail-tenant-owner@example.com",
	)
	defer ownerRollback()

	ownerRail := partial2.NewNavigationRail(ownerCtx, harness.infra, "dashboard", nil)
	assertNavigationRailLabelsExclude(t, ownerRail.GetItems(), "Users")
	assertNavigationRailLabelsContain(t, ownerRail.TopItems, "Organizations", tenantx.Name)
	assertNavigationRailLabelsContainRecursive(t, ownerRail.TopItems, "Spaces", "Users", "Settings")
	assertNavigationRailLabelsExclude(t, ownerRail.TopItems, "Spaces")

	wantFooter := []string{"Misc", "Sign out", "About SimpleDMS"}
	if got := navigationRailLabels(ownerRail.FooterItems); !reflect.DeepEqual(got, wantFooter) {
		t.Fatalf("expected footer labels %v, got %v", wantFooter, got)
	}
	activeOwnerRail := partial2.NewNavigationRail(
		ownerCtx,
		harness.infra,
		partial2.TenantUsersNavigationRailValue(tenantx.PublicID.String()),
		nil,
	)
	assertNavigationRailLabelsContain(t, activeOwnerRail.GetItems(), "Dashboard", "Account")
	assertNavigationRailItemActiveRecursive(t, activeOwnerRail.TopItems, "Users")
	activeSpacesRail := partial2.NewNavigationRail(
		ownerCtx,
		harness.infra,
		partial2.SpacesNavigationRailValue(tenantx.PublicID.String()),
		nil,
	)
	assertNavigationRailItemActiveRecursive(t, activeSpacesRail.TopItems, "Spaces")
	activeSettingsRail := partial2.NewNavigationRail(
		ownerCtx,
		harness.infra,
		partial2.TenantSettingsNavigationRailValue(tenantx.PublicID.String()),
		nil,
	)
	assertNavigationRailItemActiveRecursive(t, activeSettingsRail.TopItems, "Settings")

	createAccount(t, harness.mainDB, "rail-tenant-user@example.com", "supersecret")
	nonOwnerAccount := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.Email(entx.NewCIText("rail-tenant-user@example.com"))).
		OnlyX(context.Background())
	harness.mainDB.ReadWriteConn.TenantAccountAssignment.Create().
		SetTenantID(tenantx.ID).
		SetAccountID(nonOwnerAccount.ID).
		SetRole(tenantrole.User).
		SaveX(context.Background())
	tenantDB.ReadWriteConn.User.Create().
		SetAccountID(nonOwnerAccount.ID).
		SetRole(tenantrole.User).
		SetEmail("rail-tenant-user@example.com").
		SetFirstName("Test").
		SetLastName("User").
		SaveX(context.Background())

	_, nonOwnerCtx, nonOwnerRollback := newNavigationRailMainContext(
		t,
		harness,
		"rail-tenant-user@example.com",
	)
	defer nonOwnerRollback()

	nonOwnerRail := partial2.NewNavigationRail(nonOwnerCtx, harness.infra, "dashboard", nil)
	assertNavigationRailLabelsExclude(t, nonOwnerRail.GetItems(), "Users")
	assertNavigationRailLabelsContainRecursive(t, nonOwnerRail.TopItems, "Spaces")
	assertNavigationRailLabelsExcludeRecursive(t, nonOwnerRail.TopItems, "Users", "Settings")
}

func TestNavigationRailShowsSpaceDestinations(t *testing.T) {
	harness := newActionTestHarness(t)

	ownerAccount, tenantx := signUpAccount(t, harness, "rail-space-owner@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)

	mainTx, tenantTx, tenantCtx := newTenantContext(t, harness, ownerAccount, tenantx, tenantDB)
	createSpaceViaCmd(t, harness.actions, tenantCtx, "Docs")
	createSpaceViaCmd(t, harness.actions, tenantCtx, "Projects")
	if err := mainTx.Commit(); err != nil {
		_ = tenantTx.Rollback()
		t.Fatalf("commit main tx: %v", err)
	}
	if err := tenantTx.Commit(); err != nil {
		t.Fatalf("commit tenant tx: %v", err)
	}

	mainTx, err := harness.mainDB.ReadOnlyConn.Tx(context.Background())
	if err != nil {
		t.Fatalf("start main tx: %v", err)
	}
	defer func() {
		_ = mainTx.Rollback()
	}()
	accountInTx := mainTx.Account.Query().Where(account.ID(ownerAccount.ID)).OnlyX(context.Background())
	tenantInTx := mainTx.Tenant.GetX(context.Background(), tenantx.ID)
	visitorCtx := ctxx.NewVisitorContext(
		context.Background(),
		mainTx,
		harness.i18n,
		"",
		"",
		true,
		false,
		harness.infra.SystemConfig().CommercialLicenseEnabled(),
	)
	mainCtx := ctxx.NewMainContext(
		visitorCtx,
		accountInTx,
		harness.i18n,
		harness.mainDB,
		harness.tenantDBs,
		true,
	)
	tenantTx, err = tenantDB.ReadOnlyConn.Tx(context.Background())
	if err != nil {
		t.Fatalf("start tenant tx: %v", err)
	}
	defer func() {
		_ = tenantTx.Rollback()
	}()
	tenantCtx = ctxx.NewTenantContext(mainCtx, tenantTx, tenantInTx, true)
	tenantRail := partial2.NewNavigationRail(
		tenantCtx,
		harness.infra,
		partial2.SpacesNavigationRailValue(tenantx.PublicID.String()),
		nil,
	)
	tenantRailWant := []string{"Dashboard", "Account"}
	if got := navigationRailLabels(tenantRail.GetItems()); !reflect.DeepEqual(got, tenantRailWant) {
		t.Fatalf("expected tenant rail labels without space %v, got %v", tenantRailWant, got)
	}
	assertNavigationRailLabelsExclude(t, tenantRail.GetItems(), "Users", "System")
	assertNavigationRailLabelsContain(t, tenantRail.TopItems, "Organizations", tenantx.Name)
	assertNavigationRailItemActiveRecursive(t, tenantRail.TopItems, "Spaces")

	spacex := tenantCtx.TTx.Space.Query().Where(space.Name("Docs")).OnlyX(tenantCtx)
	spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
	rail := partial2.NewNavigationRail(spaceCtx, harness.infra, "tags", nil)

	want := []string{
		"Files",
		"Inbox",
		"Trash",
		"Manage space",
		"Document types",
		"Tags",
		"Fields",
		"Users",
	}
	if got := navigationRailLabels(rail.GetItems()); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected space rail labels %v, got %v", want, got)
	}
	assertNavigationRailLabelsExclude(t, rail.TopItems, "Account", "Users", "System")
	assertNavigationRailLabelsContain(t, rail.TopItems, "Home", "Dashboard", "Spaces by Organization", tenantx.Name)
	assertNavigationRailLabelsContainRecursive(t, rail.TopItems, "Docs", "Projects")
	assertNavigationRailLabelsExcludeRecursive(t, rail.TopItems, "Users", "Settings")
	assertNavigationRailItemInactiveRecursive(t, rail.TopItems, "Docs")
	assertNavigationRailItemIconRecursive(t, rail.TopItems, "Docs", "check_box")

	want = []string{"Files", "Inbox", "Trash", "Document types", "Tags", "Fields", "Users"}
	if got := navigationRailLabels(rail.CollapsedItems()); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected collapsed space rail labels %v, got %v", want, got)
	}
	compactWant := []string{"Files", "Document types", "Tags", "Fields"}
	if got := navigationRailLabels(rail.CompactNavigationItems()); !reflect.DeepEqual(got, compactWant) {
		t.Fatalf("expected compact metadata rail labels %v, got %v", compactWant, got)
	}
	assertNavigationRailItemActive(t, rail.GetItems(), "Tags")
	assertNavigationRailItemActive(t, rail.CompactNavigationItems(), "Tags")

	browseRail := partial2.NewNavigationRail(spaceCtx, harness.infra, "browse", nil)
	compactWant = []string{"Files", "Inbox", "Trash", "Metadata"}
	if got := navigationRailLabels(browseRail.CompactNavigationItems()); !reflect.DeepEqual(got, compactWant) {
		t.Fatalf("expected compact default rail labels %v, got %v", compactWant, got)
	}
	assertNavigationRailItemActive(t, browseRail.CompactNavigationItems(), "Files")

	documentTypesRail := partial2.NewNavigationRail(spaceCtx, harness.infra, "document-types", nil)
	assertNavigationRailItemActive(
		t,
		documentTypesRail.CompactNavigationItems(),
		"Document types",
	)
}

func TestNavigationRailShowsOnlySetupEntriesWhenPasskeyEnrollmentRequired(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "tenant-passkey-navigation-rail@example.com"
	createAccount(t, harness.mainDB, email, "supersecret")
	accountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.Email(entx.NewCIText(email))).
		OnlyX(context.Background())

	now := time.Now()
	tenantRequired := harness.mainDB.ReadWriteConn.Tenant.Create().
		SetName("Required Tenant").
		SetCountry(country.Unknown).
		SetPlan(plan.Unknown).
		SetTermsOfServiceAccepted(now).
		SetPrivacyPolicyAccepted(now).
		SetPasskeyAuthEnforced(true).
		SaveX(context.Background())

	harness.mainDB.ReadWriteConn.TenantAccountAssignment.Create().
		SetTenantID(tenantRequired.ID).
		SetAccountID(accountx.ID).
		SetRole(tenantrole.Owner).
		SetIsDefault(true).
		SaveX(context.Background())

	_, mainCtx, rollback := newNavigationRailMainContext(t, harness, email)
	defer rollback()

	normalRail := partial2.NewNavigationRail(mainCtx, harness.infra, "dashboard", nil)
	assertNavigationRailLabelsContain(t, normalRail.GetItems(), "Dashboard", "Account")

	mainCtx.VisitorCtx().IsTemporarySession = true
	rail := partial2.NewNavigationRail(mainCtx, harness.infra, "dashboard", nil)
	got := navigationRailLabels(rail.GetItems())
	want := []string{"Dashboard"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected setup rail labels %v, got %v", want, got)
	}
	got = navigationRailLabels(rail.FooterItems)
	want = []string{"Sign out"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected setup footer labels %v, got %v", want, got)
	}
}

func TestNavigationRailPluginHooksAndLegacyDestinations(t *testing.T) {
	harness := newActionTestHarness(t)
	harness.infra.PluginRegistry().SetPlugins(navigationRailTestPlugin{})

	createAccount(t, harness.mainDB, "rail-plugin@example.com", "supersecret")
	_, mainCtx, rollback := newNavigationRailMainContext(t, harness, "rail-plugin@example.com")
	defer rollback()

	rail := partial2.NewNavigationRail(mainCtx, harness.infra, "dashboard", nil)
	assertNavigationRailLabelsContain(t, rail.GetItems(), "Legacy Plugin", "Rail Plugin")
}

type navigationRailTestPlugin struct{}

func (navigationRailTestPlugin) Name() string {
	return "navigation-rail-test"
}

func (navigationRailTestPlugin) ExtendNavigationDestinations(
	ctx ctxx.Context,
	destinations []*wx.NavigationDestination,
) []*wx.NavigationDestination {
	return append(destinations, &wx.NavigationDestination{
		Value: "legacy-plugin",
		Label: "Legacy Plugin",
		Icon:  "extension",
		Href:  "/legacy-plugin/",
	})
}

func (navigationRailTestPlugin) ExtendNavigationRailItems(
	ctx ctxx.Context,
	items []*wx.NavigationRailItem,
) []*wx.NavigationRailItem {
	return append(items, &wx.NavigationRailItem{
		Key:   "rail-plugin",
		Value: "rail-plugin",
		Label: "Rail Plugin",
		Icon:  "extension",
		Href:  "/rail-plugin/",
	})
}

func newNavigationRailMainContext(
	t testing.TB,
	harness *actionTestHarness,
	email string,
) (*entmain.Tx, *ctxx.MainContext, func()) {
	t.Helper()

	mainTx, err := harness.mainDB.ReadOnlyConn.Tx(context.Background())
	if err != nil {
		t.Fatalf("start main tx: %v", err)
	}
	accountx := mainTx.Account.Query().
		Where(account.Email(entx.NewCIText(email))).
		OnlyX(context.Background())

	visitorCtx := ctxx.NewVisitorContext(
		context.Background(),
		mainTx,
		harness.i18n,
		"",
		"",
		true,
		false,
		harness.infra.SystemConfig().CommercialLicenseEnabled(),
	)
	mainCtx := ctxx.NewMainContext(
		visitorCtx,
		accountx,
		harness.i18n,
		harness.mainDB,
		harness.tenantDBs,
		true,
	)

	return mainTx, mainCtx, func() {
		_ = mainTx.Rollback()
	}
}

func navigationRailLabels(items []*wx.NavigationRailItem) []string {
	labels := make([]string, 0, len(items))
	for _, item := range items {
		if item == nil || item.Label == "" {
			continue
		}
		labels = append(labels, item.Label)
	}
	return labels
}

func navigationRailLabelsRecursive(items []*wx.NavigationRailItem) []string {
	labels := make([]string, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.Label != "" {
			labels = append(labels, item.Label)
		}
		labels = append(labels, navigationRailLabelsRecursive(item.Children)...)
	}
	return labels
}

func assertNavigationRailLabelsContain(
	t testing.TB,
	items []*wx.NavigationRailItem,
	wantLabels ...string,
) {
	t.Helper()
	labels := navigationRailLabels(items)
	for _, want := range wantLabels {
		if !navigationRailLabelsContain(labels, want) {
			t.Fatalf("expected rail labels to contain %q, got %v", want, labels)
		}
	}
}

func assertNavigationRailLabelsExclude(
	t testing.TB,
	items []*wx.NavigationRailItem,
	wantExcludedLabels ...string,
) {
	t.Helper()
	labels := navigationRailLabels(items)
	for _, excluded := range wantExcludedLabels {
		if navigationRailLabelsContain(labels, excluded) {
			t.Fatalf("expected rail labels to exclude %q, got %v", excluded, labels)
		}
	}
}

func assertNavigationRailLabelsContainRecursive(
	t testing.TB,
	items []*wx.NavigationRailItem,
	wantLabels ...string,
) {
	t.Helper()
	labels := navigationRailLabelsRecursive(items)
	for _, want := range wantLabels {
		if !navigationRailLabelsContain(labels, want) {
			t.Fatalf("expected rail labels to contain %q, got %v", want, labels)
		}
	}
}

func assertNavigationRailLabelsExcludeRecursive(
	t testing.TB,
	items []*wx.NavigationRailItem,
	wantExcludedLabels ...string,
) {
	t.Helper()
	labels := navigationRailLabelsRecursive(items)
	for _, excluded := range wantExcludedLabels {
		if navigationRailLabelsContain(labels, excluded) {
			t.Fatalf("expected rail labels to exclude %q, got %v", excluded, labels)
		}
	}
}

func assertNavigationRailItemActive(
	t testing.TB,
	items []*wx.NavigationRailItem,
	activeLabel string,
) {
	t.Helper()
	for _, item := range items {
		if item == nil || item.Label != activeLabel {
			continue
		}
		if !item.IsActive {
			t.Fatalf("expected rail item %q to be active", activeLabel)
		}
		return
	}
	t.Fatalf(
		"expected rail labels to contain active item %q, got %v",
		activeLabel,
		navigationRailLabels(items),
	)
}

func assertNavigationRailItemActiveRecursive(
	t testing.TB,
	items []*wx.NavigationRailItem,
	activeLabel string,
) {
	t.Helper()
	if navigationRailItemActiveRecursive(items, activeLabel) {
		return
	}
	t.Fatalf(
		"expected rail labels to contain active item %q, got %v",
		activeLabel,
		navigationRailLabelsRecursive(items),
	)
}

func assertNavigationRailItemInactiveRecursive(
	t testing.TB,
	items []*wx.NavigationRailItem,
	label string,
) {
	t.Helper()
	item := nilableNavigationRailItemRecursive(items, label)
	if item == nil {
		t.Fatalf(
			"expected rail labels to contain item %q, got %v",
			label,
			navigationRailLabelsRecursive(items),
		)
	}
	if item.IsActive {
		t.Fatalf("expected rail item %q not to be active", label)
	}
}

func assertNavigationRailItemIconRecursive(
	t testing.TB,
	items []*wx.NavigationRailItem,
	label string,
	wantIcon string,
) {
	t.Helper()
	item := nilableNavigationRailItemRecursive(items, label)
	if item == nil {
		t.Fatalf(
			"expected rail labels to contain item %q, got %v",
			label,
			navigationRailLabelsRecursive(items),
		)
	}
	if item.Icon != wantIcon {
		t.Fatalf("expected rail item %q icon %q, got %q", label, wantIcon, item.Icon)
	}
}

func navigationRailItemActiveRecursive(items []*wx.NavigationRailItem, activeLabel string) bool {
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.Label == activeLabel && item.IsActive {
			return true
		}
		if navigationRailItemActiveRecursive(item.Children, activeLabel) {
			return true
		}
	}
	return false
}

func nilableNavigationRailItemRecursive(
	items []*wx.NavigationRailItem,
	label string,
) *wx.NavigationRailItem {
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.Label == label {
			return item
		}
		if child := nilableNavigationRailItemRecursive(item.Children, label); child != nil {
			return child
		}
	}
	return nil
}

func navigationRailLabelsContain(labels []string, want string) bool {
	for _, label := range labels {
		if label == want {
			return true
		}
	}
	return false
}
