package file

import (
	"context"
	"log"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/privacy"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	dbfile "github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/db/enttenant/storedfile"
	"github.com/simpledms/simpledms/db/entx"
)

type DuplicateDetectionResult struct {
	HasContentHash bool
	Matches        []*DuplicateMatch
}

type DuplicateMatch struct {
	TenantPublicID    string
	SpacePublicID     string
	SpaceName         string
	ParentDirPublicID string
	ParentDirName     string
	ParentDirIsRoot   bool
	FilePublicID      string
	FileName          string
	VersionNumber     int
	IsCurrentVersion  bool
	UploadedAt        time.Time
	Size              int64
	ContentSHA256     string
}

type DuplicateDetectionService struct{}

func NewDuplicateDetectionService() *DuplicateDetectionService {
	return &DuplicateDetectionService{}
}

func (qq *DuplicateDetectionService) FindDuplicates(
	ctx ctxx.Context,
	sourceFilePublicID string,
) (*DuplicateDetectionResult, error) {
	sourceFile, err := ctx.SpaceCtx().Space.QueryFiles().
		Where(dbfile.PublicIDEQ(entx.NewCIText(sourceFilePublicID))).
		Only(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	contentSHA256s, err := qq.sourceContentSHA256s(ctx, sourceFile)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if len(contentSHA256s) == 0 {
		return &DuplicateDetectionResult{}, nil
	}

	accessibleSpaceIDs, err := ctx.TenantCtx().TTx.Space.Query().IDs(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	queryCtx := privacy.DecisionContext(ctx, privacy.Allow)
	versions, err := ctx.TenantCtx().TTx.FileVersion.Query().
		Where(
			fileversion.HasStoredFileWith(storedfile.ContentSha256In(contentSHA256s...)),
			fileversion.HasFileWith(
				dbfile.SpaceIDIn(accessibleSpaceIDs...),
				dbfile.DeletedAtIsNil(),
				dbfile.IsDirectory(false),
				dbfile.IDNEQ(sourceFile.ID),
			),
		).
		WithStoredFile().
		WithFile(func(query *enttenant.FileQuery) {
			query.WithSpace()
			query.WithParent()
		}).
		Order(fileversion.ByFileID(sql.OrderAsc()), fileversion.ByVersionNumber(sql.OrderDesc())).
		All(queryCtx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	latestVersionByFileID, err := qq.latestVersionByFileID(ctx, queryCtx, versions)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	matches := make([]*DuplicateMatch, 0, len(versions))
	for _, version := range versions {
		filex := version.Edges.File
		storedFilex := version.Edges.StoredFile
		if filex == nil || storedFilex == nil || filex.Edges.Space == nil || filex.Edges.Parent == nil {
			log.Println("duplicate match is missing loaded file, stored file, space, or parent edge")
			continue
		}
		spacex := filex.Edges.Space
		parentDir := filex.Edges.Parent

		matches = append(matches, &DuplicateMatch{
			TenantPublicID:    ctx.TenantCtx().TenantID,
			SpacePublicID:     spacex.PublicID.String(),
			SpaceName:         spacex.Name,
			ParentDirPublicID: parentDir.PublicID.String(),
			ParentDirName:     parentDir.Name,
			ParentDirIsRoot:   parentDir.IsRootDir,
			FilePublicID:      filex.PublicID.String(),
			FileName:          filex.Name,
			VersionNumber:     version.VersionNumber,
			IsCurrentVersion:  latestVersionByFileID[filex.ID] == version.VersionNumber,
			UploadedAt:        storedFilex.CreatedAt,
			Size:              storedFilex.Size,
			ContentSHA256:     storedFilex.ContentSha256,
		})
	}

	return &DuplicateDetectionResult{
		HasContentHash: true,
		Matches:        matches,
	}, nil
}

func (qq *DuplicateDetectionService) sourceContentSHA256s(
	ctx ctxx.Context,
	sourceFile *enttenant.File,
) ([]string, error) {
	versions, err := sourceFile.QueryFileVersions().WithStoredFile().All(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	seen := make(map[string]bool)
	contentSHA256s := make([]string, 0, len(versions))
	for _, version := range versions {
		if version.Edges.StoredFile == nil {
			continue
		}
		contentSHA256 := version.Edges.StoredFile.ContentSha256
		if contentSHA256 == "" || seen[contentSHA256] {
			continue
		}
		seen[contentSHA256] = true
		contentSHA256s = append(contentSHA256s, contentSHA256)
	}

	return contentSHA256s, nil
}

func (qq *DuplicateDetectionService) latestVersionByFileID(
	ctx ctxx.Context,
	queryCtx context.Context,
	versions []*enttenant.FileVersion,
) (map[int64]int, error) {
	fileIDs := make([]int64, 0, len(versions))
	seen := make(map[int64]bool)
	for _, version := range versions {
		fileID := version.FileID
		if seen[fileID] {
			continue
		}
		seen[fileID] = true
		fileIDs = append(fileIDs, fileID)
	}

	latestVersionByFileID := make(map[int64]int)
	if len(fileIDs) == 0 {
		return latestVersionByFileID, nil
	}

	latestVersions, err := ctx.TenantCtx().TTx.FileVersion.Query().
		Where(fileversion.FileIDIn(fileIDs...)).
		Order(fileversion.ByFileID(sql.OrderAsc()), fileversion.ByVersionNumber(sql.OrderDesc())).
		All(queryCtx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	for _, version := range latestVersions {
		if _, exists := latestVersionByFileID[version.FileID]; exists {
			continue
		}
		latestVersionByFileID[version.FileID] = version.VersionNumber
	}

	return latestVersionByFileID, nil
}
