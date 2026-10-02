package file

import (
	"log"
	"net/http"
	"strings"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/documentnote"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/db/enttenant/privacy"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/enttenant/tagassignment"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/db/enttenant/webdavresource"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/spacerole"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	"github.com/simpledms/simpledms/util/e"
)

// InboxTransferService coordinates transfers in the caller's transaction.
// Callers must roll back the transaction if any step fails.
type InboxTransferService struct{}

func NewInboxTransferService() *InboxTransferService {
	return &InboxTransferService{}
}

func (qq *InboxTransferService) Destinations(
	ctx ctxx.Context, fileID string,
) ([]*enttenant.Space, error) {
	if _, err := qq.source(ctx, fileID); err != nil {
		return nil, err
	}
	query, err := qq.destinationQuery(ctx)
	if err != nil {
		return nil, err
	}
	// Opting in publishes only the destination's name and public ID to tenant members.
	// Keep the privacy exception local to this query, never on the request context.
	destinations, err := query.Select(space.FieldPublicID, space.FieldName).Order(space.ByName()).
		All(privacy.DecisionContext(ctx, privacy.Allow))
	if err != nil {
		log.Printf("list Inbox transfer destinations: %v", err)
	}
	return destinations, err
}

func (qq *InboxTransferService) Transfer(
	ctx ctxx.Context, fileID, destinationID, message string,
) (*enttenant.Space, error) {
	doc, err := qq.source(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if _, err := NewDocumentNotes().editable(ctx, doc); err != nil {
		return nil, err
	}
	// Re-read destination access and opt-in; knowing a Space's public ID is insufficient.
	query, err := qq.destinationQuery(ctx)
	if err != nil {
		return nil, err
	}
	destination, err := query.Where(space.PublicID(entx.NewCIText(destinationID))).
		Select(space.FieldID, space.FieldPublicID, space.FieldName).
		Only(privacy.DecisionContext(ctx, privacy.Allow))
	if err != nil {
		log.Printf("resolve Inbox transfer destination: %v", err)
		if enttenant.IsNotFound(err) {
			return nil, qq.failure(http.StatusNotFound, "Destination Inbox is unavailable.")
		}
		return nil, err
	}
	// The destination context stays inside this operation and grants no user membership.
	destinationCtx := ctxx.NewSpaceContext(ctx.TenantCtx(), destination)
	root, err := ctx.TenantCtx().TTx.File.Query().Where(
		file.SpaceID(destination.ID), file.IsRootDir(true), file.IsDirectory(true),
	).Only(destinationCtx)
	if err != nil {
		log.Printf("resolve Inbox transfer root: %v", err)
		return nil, err
	}

	if strings.TrimSpace(message) != "" {
		title := ctx.VisitorCtx().Printer.Sprintf("Inbox transfer")
		if _, err := NewDocumentNotes().Create(ctx, fileID, title, message); err != nil {
			return nil, err
		}
	}
	if _, err := ctx.TenantCtx().TTx.DocumentNote.Update().Where(
		documentnote.FileID(doc.ID), documentnote.SpaceID(doc.SpaceID),
	).SetSpaceID(destination.ID).Save(ctx); err != nil {
		log.Printf("move Inbox document notes: %v", err)
		return nil, err
	}
	if _, err := ctx.TenantCtx().TTx.TagAssignment.Delete().
		Where(tagassignment.FileID(doc.ID)).Exec(ctx); err != nil {
		log.Printf("clear transferred document tags: %v", err)
		return nil, err
	}
	if _, err := ctx.TenantCtx().TTx.FilePropertyAssignment.Delete().
		Where(filepropertyassignment.FileID(doc.ID)).Exec(ctx); err != nil {
		log.Printf("clear transferred document fields: %v", err)
		return nil, err
	}
	// Upload aliases belong to the old Inbox and must not outlive the transfer.
	if _, err := ctx.TenantCtx().TTx.WebDAVResource.Delete().
		Where(enttenantwebdavresource.FileID(doc.ID)).Exec(ctx); err != nil {
		log.Printf("remove transferred document upload aliases: %v", err)
		return nil, err
	}
	count, err := ctx.TenantCtx().TTx.File.Update().Where(
		file.ID(doc.ID), file.SpaceID(doc.SpaceID), file.IsInInbox(true),
		file.IsDirectory(false), file.DeletedAtIsNil(),
	).SetSpaceID(destination.ID).SetParentID(root.ID).ClearDocumentTypeID().Save(ctx)
	if err != nil {
		log.Printf("transfer Inbox document: %v", err)
		return nil, err
	}
	if count != 1 {
		return nil, qq.failure(http.StatusConflict, "File has changed. Please reload.")
	}
	return destination, nil
}

func (qq *InboxTransferService) destinationQuery(ctx ctxx.Context) (*enttenant.SpaceQuery, error) {
	actor, err := ctx.TenantCtx().TTx.User.Query().Where(
		user.ID(ctx.TenantCtx().User.ID), user.AccountID(ctx.MainCtx().Account.ID),
	).Only(ctx)
	if err != nil {
		log.Printf("verify Inbox transfer actor: %v", err)
		return nil, err
	}
	query := ctx.TenantCtx().TTx.Space.Query().Where(
		space.IDNEQ(ctx.SpaceCtx().Space.ID), space.DeletedAtIsNil(),
	)
	if actor.Role != tenantrole.Owner {
		query.Where(space.Or(
			space.AcceptsInboxTransfers(true),
			space.HasUserAssignmentWith(
				spaceuserassignment.UserID(actor.ID),
				spaceuserassignment.RoleIn(spacerole.User, spacerole.Owner),
			),
		))
	}
	return query, nil
}

func (qq *InboxTransferService) source(
	ctx ctxx.Context, fileID string,
) (*enttenant.File, error) {
	// Reuse document access checks, including current tenant and Space membership.
	doc, err := NewDocumentNotes().document(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if !doc.DeletedAt.IsZero() || !doc.IsInInbox || doc.IsRootDir {
		return nil, qq.failure(http.StatusBadRequest, "Only files in the Inbox can be transferred.")
	}
	return doc, nil
}

func (qq *InboxTransferService) failure(status int, message string) error {
	log.Printf("Inbox transfer: %s", message)
	return e.NewHTTPErrorf(status, message)
}
