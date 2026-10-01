package server

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/fieldtype"
	credentialmodel "github.com/simpledms/simpledms/model/main/mcpcredential"
	documenttypemodel "github.com/simpledms/simpledms/model/tenant/documenttype"
	propertymodel "github.com/simpledms/simpledms/model/tenant/property"
	taggingmodel "github.com/simpledms/simpledms/model/tenant/tagging"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
)

func TestMCPClassificationCommitErrorLeavesTagTypeAndPropertyUnchanged(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixtureWithWrites(t, h, "classification-commit")
	var tagID, propertyID, documentTypeID string
	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		spacex := tc.TTx.Space.Query().Where(space.PublicID(entx.NewCIText(f.spaceID))).OnlyX(tc)
		sc := ctxx.NewSpaceContext(tc, spacex)
		tagx, err := taggingmodel.NewTagService().Create(sc, spacex.ID, 0, "Committed", tagtype.Simple)
		if err != nil {
			return err
		}
		propertyx, err := propertymodel.NewPropertyService().Create(sc, spacex.ID, "Reference", fieldtype.Text, "")
		if err != nil {
			return err
		}
		typex, err := documenttypemodel.Create(sc, spacex.ID, "Invoice")
		if err != nil {
			return err
		}
		tagID, propertyID, documentTypeID = tagx.PublicID.String(), propertyx.PublicID.String(), typex.Data.PublicID.String()
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := f.connect(t, server.URL+"/mcp")
	before := callMCP(t, client, "get_file", map[string]any{"file_id": f.fileID})

	_, err := credentialmodel.NewCredentialService().ExecuteWrite(
		context.Background(), h.mainDB, h.tenantDBs, h.i18n, false, f.token,
		func(sc *ctxx.SpaceContext, _ *entmain.MCPCredential) error {
			sc.TTx.OnCommit(func(enttenant.Committer) enttenant.Committer {
				return enttenant.CommitFunc(func(context.Context, *enttenant.Tx) error {
					return errors.New("injected classification commit failure")
				})
			})
			tag, err := sc.TTx.Tag.Query().Where(tag.PublicID(entx.NewCIText(tagID))).Only(sc)
			if err != nil {
				return err
			}
			filex, err := sc.TTx.File.Query().Where(file.PublicID(entx.NewCIText(f.fileID))).Only(sc)
			if err != nil {
				return err
			}
			if _, err := taggingmodel.NewTagService().AssignToFile(sc, filex.ID, tag.ID, sc.Space.ID); err != nil {
				return err
			}
			propertyx, err := sc.TTx.Property.Query().Where(property.PublicID(entx.NewCIText(propertyID))).Only(sc)
			if err != nil {
				return err
			}
			if _, _, err := propertymodel.NewFilePropertyAssignmentService().Set(
				sc, filex.ID, propertyx.ID, propertymodel.NewTextFilePropertyValue("changed"),
			); err != nil {
				return err
			}
			documentType, err := sc.TTx.DocumentType.Query().Where(documenttype.PublicID(entx.NewCIText(documentTypeID))).Only(sc)
			if err != nil {
				return err
			}
			_, err = documenttypemodel.NewAssignmentService().Set(sc, filex.ID, documentType.ID)
			return err
		},
	)
	if err == nil {
		t.Fatal("classification operation succeeded despite commit failure")
	}
	after := callMCP(t, client, "get_file", map[string]any{"file_id": f.fileID})
	if len(after["direct_tags"].([]any)) != len(before["direct_tags"].([]any)) ||
		len(after["properties"].([]any)) != len(before["properties"].([]any)) {
		t.Fatalf("classification commit leaked metadata: before=%v after=%v", before, after)
	}
	if _, exists := after["document_type"]; exists {
		t.Fatalf("classification commit leaked document type: %v", after)
	}
}
