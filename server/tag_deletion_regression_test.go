package server

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/db/enttenant/tagassignment"
	"github.com/simpledms/simpledms/db/entx"
	documenttypemodel "github.com/simpledms/simpledms/model/tenant/documenttype"
	taggingmodel "github.com/simpledms/simpledms/model/tenant/tagging"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
)

func TestTagDeletionRegressionRemovesAssignmentsAndRefreshesTagList(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixtureWithWrites(t, h, "tag-delete-regression")
	var childID, keepID, foreignID int64
	var activeID, trashedID int64
	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		sp := tc.TTx.Space.Query().Where(space.PublicID(entx.NewCIText(f.spaceID))).OnlyX(tc)
		sc := ctxx.NewSpaceContext(tc, sp)
		activeID = createRegularFileForTest(sc, sc.SpaceRootDir().ID, "active.txt").Data.ID
		trashedID = createRegularFileForTest(sc, sc.SpaceRootDir().ID, "trashed.txt").Data.ID
		svc := taggingmodel.NewTagService()
		group, err := svc.Create(sc, sp.ID, 0, "Group", tagtype.Group)
		if err != nil {
			return err
		}
		child, err := svc.Create(sc, sp.ID, group.ID, "Child", tagtype.Simple)
		if err != nil {
			return err
		}
		keep, err := svc.Create(sc, sp.ID, 0, "Keep", tagtype.Simple)
		if err != nil {
			return err
		}
		if _, err := svc.AssignToFile(sc, activeID, child.ID, sp.ID); err != nil {
			return err
		}
		if _, err := svc.AssignToFile(sc, trashedID, child.ID, sp.ID); err != nil {
			return err
		}
		if _, err := svc.AssignToFile(sc, activeID, keep.ID, sp.ID); err != nil {
			return err
		}
		tc.TTx.File.UpdateOneID(trashedID).SetDeletedAt(time.Now()).SaveX(sc)
		createSpaceViaCmd(t, h.actions, tc, "Tag Foreign Space")
		foreignSpace := tc.TTx.Space.Query().Where(space.Name("Tag Foreign Space")).OnlyX(tc)
		foreignCtx := ctxx.NewSpaceContext(tc, foreignSpace)
		foreign, err := svc.Create(foreignCtx, foreignSpace.ID, 0, "Foreign", tagtype.Simple)
		if err != nil {
			return err
		}
		childID, keepID, foreignID = child.ID, keep.ID, foreign.ID
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	foreignDelete := f.browserAt(route.ManageTags(f.tenant.PublicID.String(), f.spaceID),
		h.actions.Tagging.DeleteTagCmd.Endpoint(), url.Values{
			"TagID": {strconv.FormatInt(foreignID, 10)},
		})
	if foreignDelete.Code < 400 {
		t.Fatalf("cross-space tag deletion status %d", foreignDelete.Code)
	}

	deleted := f.browserAt(route.ManageTags(f.tenant.PublicID.String(), f.spaceID),
		h.actions.Tagging.DeleteTagCmd.Endpoint(), url.Values{
			"TagID": {strconv.FormatInt(childID, 10)},
		})
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete status %d", deleted.Code)
	}
	if deleted.Header().Get("HX-Trigger") != event.TagDeleted.String() ||
		deleted.Header().Get("HX-Reswap") != "none" {
		t.Fatalf("unexpected command headers: trigger=%q reswap=%q",
			deleted.Header().Get("HX-Trigger"), deleted.Header().Get("HX-Reswap"))
	}
	if strings.Contains(deleted.Body.String(), "tagList") {
		t.Fatal("delete command returned replacement tag list")
	}

	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		sp := tc.TTx.Space.Query().Where(space.PublicID(entx.NewCIText(f.spaceID))).OnlyX(tc)
		sc := ctxx.NewSpaceContext(tc, sp)
		if tc.TTx.TagAssignment.Query().Where(tagassignment.TagID(childID)).CountX(sc) != 0 {
			t.Fatal("deleted tag assignments remain")
		}
		if tc.TTx.TagAssignment.Query().Where(tagassignment.TagID(keepID)).CountX(sc) != 1 {
			t.Fatal("unrelated assignment was removed")
		}
		for _, id := range []int64{activeID, trashedID} {
			if !tc.TTx.File.Query().Where(file.ID(id)).ExistX(schema.SkipSoftDelete(sc)) {
				t.Fatalf("file %d was removed", id)
			}
		}
		foreignSpace := tc.TTx.Space.Query().Where(space.Name("Tag Foreign Space")).OnlyX(tc)
		foreignCtx := ctxx.NewSpaceContext(tc, foreignSpace)
		if !foreignSpace.QueryTags().Where(tag.ID(foreignID)).ExistX(foreignCtx) {
			t.Fatal("cross-space tag was deleted")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	list := f.browserAt(route.ManageTags(f.tenant.PublicID.String(), f.spaceID),
		h.actions.ManageTags.TagListPartial.Endpoint(), url.Values{"ParentTagID": {"0"}})
	if list.Code != http.StatusOK {
		t.Fatalf("tag list status %d", list.Code)
	}
	body := list.Body.String()
	if strings.Contains(body, "Child") || !strings.Contains(body, "Keep") ||
		!strings.Contains(body, `id="tagList"`) {
		t.Fatalf("unexpected refreshed tag-list content: child=%t keep=%t root=%t",
			strings.Contains(body, "Child"),
			strings.Contains(body, "Keep"),
			strings.Contains(body, `id="tagList"`),
		)
	}
}

func TestTagDeletionRegressionPreservesAttributeReferencedTag(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixtureWithWrites(t, h, "tag-delete-reference")
	var groupID, childID, assignedFileID int64
	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		sp := tc.TTx.Space.Query().Where(space.PublicID(entx.NewCIText(f.spaceID))).OnlyX(tc)
		sc := ctxx.NewSpaceContext(tc, sp)
		service := taggingmodel.NewTagService()
		group, err := service.Create(sc, sp.ID, 0, "Referenced group", tagtype.Group)
		if err != nil {
			return err
		}
		child, err := service.Create(sc, sp.ID, group.ID, "Assigned child", tagtype.Simple)
		if err != nil {
			return err
		}
		fileID := createRegularFileForTest(sc, sc.SpaceRootDir().ID, "reference.txt").Data.ID
		if _, err := service.AssignToFile(sc, fileID, child.ID, sp.ID); err != nil {
			return err
		}
		documentType, err := documenttypemodel.Create(sc, sp.ID, "Referenced type")
		if err != nil {
			return err
		}
		if _, err := documentType.CreateTagAttribute(sc, "Group", group.ID, false); err != nil {
			return err
		}
		groupID, childID, assignedFileID = group.ID, child.ID, fileID
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	response := f.browserAt(route.ManageTags(f.tenant.PublicID.String(), f.spaceID),
		h.actions.Tagging.DeleteTagCmd.Endpoint(), url.Values{
			"TagID": {strconv.FormatInt(groupID, 10)},
		})
	if response.Code < http.StatusBadRequest {
		t.Fatalf("expected referenced tag deletion to fail, got %d", response.Code)
	}
	if strings.Contains(response.Header().Get("HX-Trigger"), event.TagDeleted.String()) {
		t.Fatal("failed deletion emitted the tag-deleted event")
	}
	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		sp := tc.TTx.Space.Query().Where(space.PublicID(entx.NewCIText(f.spaceID))).OnlyX(tc)
		sc := ctxx.NewSpaceContext(tc, sp)
		if !tc.TTx.Tag.Query().Where(tag.ID(groupID)).ExistX(sc) {
			t.Fatal("attribute-referenced group was deleted")
		}
		if !tc.TTx.TagAssignment.Query().Where(
			tagassignment.TagID(childID), tagassignment.FileID(assignedFileID),
		).ExistX(sc) {
			t.Fatal("failed deletion removed the child's assignment")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
