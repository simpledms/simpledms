package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestTenantMaintenanceWarningIsScopedToCurrentTenant(t *testing.T) {
	harness := newActionTestHarness(t)
	for _, maintenance := range []bool{true, false} {
		email := "healthy-warning@example.com"
		if maintenance {
			email = "maintenance-warning@example.com"
		}
		accountx, tenantx := signUpAccount(t, harness, email)
		tenantDB := initTenantDB(t, harness, tenantx)
		if maintenance {
			tenantx = tenantx.Update().SetMaintenanceModeEnabledAt(time.Now()).SaveX(context.Background())
		}
		err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(
			mainTx *entmain.Tx,
			_ *enttenant.Tx,
			tenantCtx *ctxx.TenantContext,
		) error {
			// Navigation follows account edges, so bind the account to the request transaction.
			tenantCtx.MainCtx().Account = mainTx.Account.GetX(tenantCtx, accountx.ID)
			createSpaceViaCmd(t, harness.actions, tenantCtx, "Warning Test Space")
			for _, htmx := range []bool{false, true} {
				req := httptest.NewRequest(http.MethodGet, route.SpacesRoot(tenantx.PublicID.String()), nil)
				if htmx {
					req.Header.Set("HX-Request", "true")
					req.Header.Set("HX-Boosted", "true")
				}
				rr := httptest.NewRecorder()
				if err := harness.actions.Spaces.SpacesPage.Handler(
					httpx.NewResponseWriter(rr), httpx.NewRequest(req), tenantCtx,
				); err != nil {
					return err
				}
				body := rr.Body.String()
				if got := strings.Contains(body, `id="tenantMaintenanceWarning"`); got != maintenance {
					t.Fatalf("maintenance=%t htmx=%t: warning present=%t", maintenance, htmx, got)
				}
				if !strings.Contains(body, "Warning Test Space") {
					t.Fatal("normal organization content is missing")
				}
				if maintenance && !strings.Contains(body, "Please contact your administrator.") {
					t.Fatal("warning must explain what to do")
				}
			}

			spacex := tenantCtx.TTx.Space.Query().Where(space.Name("Warning Test Space")).OnlyX(tenantCtx)
			for _, ctx := range []ctxx.Context{
				tenantCtx.MainCtx().VisitorContext,
				tenantCtx.MainCtx(),
				ctxx.NewSpaceContext(tenantCtx, spacex),
			} {
				rr := httptest.NewRecorder()
				if err := harness.infra.Renderer().Render(httpx.NewResponseWriter(rr), ctx,
					&widget.MainLayout{Content: widget.Tu("Page content")},
				); err != nil {
					return err
				}
				want := maintenance && ctx.IsSpaceCtx()
				if got := strings.Contains(rr.Body.String(), `id="tenantMaintenanceWarning"`); got != want {
					t.Fatalf("space=%t maintenance=%t: warning present=%t", ctx.IsSpaceCtx(), maintenance, got)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
