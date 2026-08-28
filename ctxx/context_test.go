package ctxx

import (
	"context"
	"testing"

	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/i18n"
	"github.com/simpledms/simpledms/model/main/common/language"
)

var (
	_ Context = (*MainContext)(nil)
	_ Context = (*TenantContext)(nil)
	_ Context = (*SpaceContext)(nil)
)

func TestDerivedContextsKeepParentContextsUnchanged(t *testing.T) {
	baseKey := struct{}{}
	base := context.WithValue(context.Background(), baseKey, "base value")
	i18nx := i18n.NewI18n()
	visitorCtx := NewVisitorContext(base, nil, i18nx, "", "", false, false, false)
	visitorParent := visitorCtx.Context

	mainCtx := NewMainContext(
		visitorCtx,
		&entmain.Account{Language: language.English},
		i18nx,
		nil,
		nil,
		true,
	)
	if visitorCtx.Context != visitorParent {
		t.Fatal("creating main context changed visitor context")
	}
	if _, ok := MainCtx(visitorCtx); ok {
		t.Fatal("visitor context unexpectedly contains main context")
	}

	mainParent := mainCtx.Context
	tenantCtx := NewTenantContextWithUser(
		mainCtx,
		nil,
		&entmain.Tenant{PublicID: entx.NewCIText("tenant")},
		&enttenant.User{},
		true,
	)
	if mainCtx.Context != mainParent {
		t.Fatal("creating tenant context changed main context")
	}
	if _, ok := TenantCtx(mainCtx); ok {
		t.Fatal("main context unexpectedly contains tenant context")
	}

	tenantParent := tenantCtx.Context
	spaceCtx := NewSpaceContext(
		tenantCtx,
		&enttenant.Space{PublicID: entx.NewCIText("space")},
	)
	if tenantCtx.Context != tenantParent {
		t.Fatal("creating space context changed tenant context")
	}
	if _, ok := SpaceCtx(tenantCtx); ok {
		t.Fatal("tenant context unexpectedly contains space context")
	}

	if got := spaceCtx.Value(baseKey); got != "base value" {
		t.Fatalf("expected base context value, got %v", got)
	}
	if got, ok := VisitorCtx(spaceCtx); !ok || got != visitorCtx {
		t.Fatal("derived context lost visitor context value")
	}
	if got, ok := MainCtx(spaceCtx); !ok || got != mainCtx {
		t.Fatal("derived context lost main context value")
	}
	if got, ok := TenantCtx(spaceCtx); !ok || got != tenantCtx {
		t.Fatal("derived context lost tenant context value")
	}
	if got, ok := SpaceCtx(spaceCtx); !ok || got != spaceCtx {
		t.Fatal("derived context lost space context value")
	}
}
