package entx_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	"entgo.io/ent/privacy"
	_ "github.com/mattn/go-sqlite3"

	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/account"
	mainenttest "github.com/simpledms/simpledms/db/entmain/enttest"
	mainschema "github.com/simpledms/simpledms/db/entmain/schema"
	"github.com/simpledms/simpledms/db/enttenant"
	tenantenttest "github.com/simpledms/simpledms/db/enttenant/enttest"
	tenantschema "github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/attributetype"
	"github.com/simpledms/simpledms/model/main/common/fieldtype"
	"github.com/simpledms/simpledms/model/main/common/language"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
)

func newIntegrityTenant(t *testing.T) (*enttenant.Client, context.Context) {
	t.Helper()
	client := tenantenttest.Open(t, dialect.SQLite,
		"file:"+filepath.Join(t.TempDir(), "tenant.sqlite3")+"?_foreign_keys=1")
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Error(err)
		}
	})
	return client, privacy.DecisionContext(context.Background(), privacy.Allow)
}

func TestTenantSoftDeletePreservesHistoryAndReleasesEmail(t *testing.T) {
	client, ctx := newIntegrityTenant(t)
	create := func(accountID int64, email string) (*enttenant.User, error) {
		return client.User.Create().SetAccountID(accountID).SetRole(tenantrole.User).
			SetFirstName("Member").SetEmail(entx.NewCIText(email)).Save(ctx)
	}
	original, err := create(1, "reuse@example.com")
	if err != nil {
		t.Fatal(err)
	}
	other, err := create(2, "other@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := create(3, "REUSE@example.com"); !enttenant.IsConstraintError(err) {
		t.Fatalf("active duplicate email must be rejected: %v", err)
	}
	if err := client.User.DeleteOneID(original.ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if client.User.Query().Where(user.ID(original.ID)).ExistX(ctx) {
		t.Fatal("soft-deleted user is still visible")
	}
	if !client.User.Query().Where(user.ID(other.ID)).ExistX(ctx) {
		t.Fatal("DeleteOne affected a different user")
	}
	historyCtx := tenantschema.SkipSoftDelete(ctx)
	if client.User.GetX(historyCtx, original.ID).DeletedAt.IsZero() {
		t.Fatal("user history has no deletion timestamp")
	}
	if err := client.User.DeleteOneID(original.ID).Exec(ctx); !enttenant.IsNotFound(err) {
		t.Fatalf("deleting an already deleted user must report not found: %v", err)
	}
	if _, err := create(3, "REUSE@example.com"); err != nil {
		t.Fatalf("deleted email should be reusable: %v", err)
	}
	if _, err := create(1, "different@example.com"); !enttenant.IsConstraintError(err) {
		t.Fatalf("account identity must remain unique across history: %v", err)
	}
	_, err = client.User.UpdateOneID(original.ID).ClearDeletedAt().Save(historyCtx)
	if !enttenant.IsConstraintError(err) {
		t.Fatalf("restoring a duplicate active email must fail: %v", err)
	}
	count, err := client.User.Delete().Where(user.Email(entx.NewCIText("reuse@example.com"))).Exec(ctx)
	if err != nil || count != 1 {
		t.Fatalf("bulk delete affected %d active users: %v", count, err)
	}
	if client.User.Query().CountX(historyCtx) != 3 || client.User.Query().CountX(ctx) != 1 {
		t.Fatal("bulk soft delete lost history or ignored its predicate")
	}
	if err := client.User.DeleteOneID(original.ID).Exec(historyCtx); err != nil {
		t.Fatal(err)
	}
	if client.User.Query().Where(user.ID(original.ID)).ExistX(historyCtx) {
		t.Fatal("hard-delete opt-out retained the user")
	}
}

func TestMainSoftDeleteSingleAndBulkPreserveHistory(t *testing.T) {
	client := mainenttest.Open(t, dialect.SQLite,
		"file:"+filepath.Join(t.TempDir(), "main.sqlite3")+"?_foreign_keys=1")
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	var accounts []*entmain.Account
	for i := range 3 {
		accounts = append(accounts, client.Account.Create().
			SetEmail(entx.NewCIText(fmt.Sprintf("member%d@example.com", i))).
			SetFirstName("Member").SetLastName("").SetLanguage(language.English).
			SetRole(mainrole.User).SaveX(ctx))
	}
	if err := client.Account.DeleteOneID(accounts[0].ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if client.Account.Query().CountX(ctx) != 2 {
		t.Fatal("single soft delete did not target exactly one account")
	}
	historyCtx := mainschema.SkipSoftDelete(ctx)
	if client.Account.GetX(historyCtx, accounts[0].ID).DeletedAt.IsZero() {
		t.Fatal("account history has no deletion timestamp")
	}
	count, err := client.Account.Delete().Where(
		account.IDIn(accounts[0].ID, accounts[1].ID),
	).Exec(ctx)
	if err != nil || count != 1 {
		t.Fatalf("bulk delete should affect only its active match, got %d: %v", count, err)
	}
	if client.Account.Query().CountX(ctx) != 1 || client.Account.Query().CountX(historyCtx) != 3 {
		t.Fatal("bulk delete affected unrelated accounts or lost history")
	}
	if err := client.Account.DeleteOneID(accounts[0].ID).Exec(ctx); !entmain.IsNotFound(err) {
		t.Fatalf("repeated delete should report not found: %v", err)
	}
	if err := client.Account.DeleteOneID(accounts[0].ID).Exec(historyCtx); err != nil {
		t.Fatal(err)
	}
	if client.Account.Query().Where(account.ID(accounts[0].ID)).ExistX(historyCtx) {
		t.Fatal("hard-delete opt-out retained the account")
	}
}

func TestAttributeTypeRequiresExactlyItsMatchingTarget(t *testing.T) {
	client, ctx := newIntegrityTenant(t)
	space := client.Space.Create().SetName("Metadata").SaveX(ctx)
	documentType := client.DocumentType.Create().SetSpaceID(space.ID).SetName("Invoice").SaveX(ctx)
	tag := client.Tag.Create().SetSpaceID(space.ID).SetName("Status").SetType(tagtype.Group).SaveX(ctx)
	property := client.Property.Create().SetSpaceID(space.ID).SetName("Amount").
		SetType(fieldtype.Money).SaveX(ctx)
	for _, test := range []struct {
		name       string
		typex      attributetype.AttributeType
		hasTag     bool
		hasField   bool
		shouldPass bool
	}{
		{"tag", attributetype.Tag, true, false, true},
		{"field", attributetype.Field, false, true, true},
		{"tag without target", attributetype.Tag, false, false, false},
		{"field without target", attributetype.Field, false, false, false},
		{"tag with field", attributetype.Tag, false, true, false},
		{"field with tag", attributetype.Field, true, false, false},
		{"tag with both", attributetype.Tag, true, true, false},
		{"field with both", attributetype.Field, true, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			create := client.Attribute.Create().SetSpaceID(space.ID).
				SetDocumentTypeID(documentType.ID).SetType(test.typex)
			if test.hasTag {
				create.SetTagID(tag.ID)
			}
			if test.hasField {
				create.SetPropertyID(property.ID)
			}
			attribute, err := create.Save(ctx)
			if !test.shouldPass {
				if !enttenant.IsConstraintError(err) {
					t.Fatalf("invalid attribute was not rejected: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			_, err = attribute.Update().ClearTagID().ClearPropertyID().Save(ctx)
			if !enttenant.IsConstraintError(err) {
				t.Fatalf("clearing the sole target must fail: %v", err)
			}
			if err := client.Attribute.DeleteOne(attribute).Exec(ctx); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFilePropertyAssignmentCannotContainConflictingValues(t *testing.T) {
	client, ctx := newIntegrityTenant(t)
	space := client.Space.Create().SetName("Fields").SaveX(ctx)
	file := client.File.Create().SetSpaceID(space.ID).SetName("invoice.pdf").
		SetIsDirectory(false).SetIndexedAt(time.Now()).SaveX(ctx)
	property := client.Property.Create().SetSpaceID(space.ID).SetName("Value").
		SetType(fieldtype.Text).SaveX(ctx)
	for _, column := range []string{"text_value", "number_value", "date_value", "bool_value"} {
		t.Run(column, func(t *testing.T) {
			assignment := client.FilePropertyAssignment.Create().SetSpaceID(space.ID).
				SetFileID(file.ID).SetPropertyID(property.ID).SaveX(ctx)
			// Zero, false, and empty text are populated values, not absent values.
			value := any(0)
			if column == "text_value" {
				value = ""
			} else if column == "date_value" {
				value = "2026-10-23"
			}
			if _, err := client.ExecContext(ctx,
				"UPDATE file_property_assignments SET "+column+" = ? WHERE id = ?", value, assignment.ID,
			); err != nil {
				t.Fatal(err)
			}
			conflicting := "text_value"
			if column == conflicting {
				conflicting = "number_value"
			}
			if _, err := client.ExecContext(ctx,
				"UPDATE file_property_assignments SET "+conflicting+" = 0 WHERE id = ?", assignment.ID,
			); err == nil {
				t.Fatal("database accepted conflicting populated value columns")
			}
			if err := client.FilePropertyAssignment.DeleteOne(assignment).Exec(ctx); err != nil {
				t.Fatal(err)
			}
		})
	}
	_, err := client.FilePropertyAssignment.Create().SetSpaceID(space.ID).
		SetFileID(file.ID).SetPropertyID(property.ID).SetTextValue("").SetBoolValue(false).Save(ctx)
	if !enttenant.IsConstraintError(err) {
		t.Fatalf("creating conflicting values must fail: %v", err)
	}
}
