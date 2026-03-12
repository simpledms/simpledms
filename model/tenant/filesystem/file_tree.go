package filesystem

import (
	"log"
	"net/http"
	"path/filepath"
	"slices"
	"strings"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/util/e"
)

type FileTree struct{}

func NewFileTree() *FileTree {
	return &FileTree{}
}

func (qq *FileTree) PathFilesByFileID(ctx ctxx.Context, fileID int64) ([]*enttenant.File, error) {
	if fileID == 0 {
		return []*enttenant.File{}, nil
	}

	pathFiles := []*enttenant.File{}
	seenFileIDs := map[int64]struct{}{}
	currentFileID := fileID

	for currentFileID != 0 {
		if _, found := seenFileIDs[currentFileID]; found {
			return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Detected cycle in directory tree.")
		}
		seenFileIDs[currentFileID] = struct{}{}

		currentFile, err := ctx.TenantCtx().TTx.File.Query().
			Select(
				file.FieldID,
				file.FieldParentID,
				file.FieldName,
				file.FieldPublicID,
			).
			Where(
				file.ID(currentFileID),
				file.SpaceID(ctx.SpaceCtx().Space.ID),
			).
			Only(ctx)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		pathFiles = append(pathFiles, currentFile)
		currentFileID = currentFile.ParentID
	}

	slices.Reverse(pathFiles)
	return pathFiles, nil
}

func (qq *FileTree) PathFilesByFileIDX(ctx ctxx.Context, fileID int64) []*enttenant.File {
	pathFiles, err := qq.PathFilesByFileID(ctx, fileID)
	if err != nil {
		log.Println(err)
		panic(err)
	}
	return pathFiles
}

func (qq *FileTree) FullPathByFileID(ctx ctxx.Context, fileID int64) (string, error) {
	if fileID == 0 {
		return "", nil
	}

	pathFiles, err := qq.PathFilesByFileID(ctx, fileID)
	if err != nil {
		return "", err
	}

	return qq.fullPathFromPathFiles(pathFiles), nil
}

func (qq *FileTree) FullPathByFileIDX(ctx ctxx.Context, fileID int64) string {
	fullPath, err := qq.FullPathByFileID(ctx, fileID)
	if err != nil {
		log.Println(err)
		panic(err)
	}
	return fullPath
}

func (qq *FileTree) FullPathByPublicID(ctx ctxx.Context, filePublicID string) (string, error) {
	filex, err := ctx.TenantCtx().TTx.File.Query().
		Select(file.FieldID).
		Where(
			file.PublicID(entx.NewCIText(filePublicID)),
			file.SpaceID(ctx.SpaceCtx().Space.ID),
		).
		Only(ctx)
	if err != nil {
		log.Println(err)
		return "", err
	}

	return qq.FullPathByFileID(ctx, filex.ID)
}

func (qq *FileTree) FullPathByPublicIDX(ctx ctxx.Context, filePublicID string) string {
	fullPath, err := qq.FullPathByPublicID(ctx, filePublicID)
	if err != nil {
		log.Println(err)
		panic(err)
	}
	return fullPath
}

func (qq *FileTree) FullPathsByFileID(ctx ctxx.Context, fileIDs []int64) (map[int64]string, error) {
	pathByFileID := map[int64]string{}

	if len(fileIDs) == 0 {
		return pathByFileID, nil
	}

	uniqueFileIDs := slices.Clone(fileIDs)
	slices.Sort(uniqueFileIDs)
	uniqueFileIDs = slices.Compact(uniqueFileIDs)

	pendingFileIDs := []int64{}
	for _, fileID := range uniqueFileIDs {
		if fileID != 0 {
			pendingFileIDs = append(pendingFileIDs, fileID)
		}
	}

	if len(pendingFileIDs) == 0 {
		return pathByFileID, nil
	}

	fileByID := map[int64]*enttenant.File{}
	for len(pendingFileIDs) > 0 {
		batch := slices.Clone(pendingFileIDs)
		pendingFileIDs = []int64{}

		batchFiles, err := ctx.TenantCtx().TTx.File.Query().
			Select(
				file.FieldID,
				file.FieldParentID,
				file.FieldName,
			).
			Where(
				file.IDIn(batch...),
				file.SpaceID(ctx.SpaceCtx().Space.ID),
			).
			All(ctx)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		for _, batchFile := range batchFiles {
			fileByID[batchFile.ID] = batchFile
		}

		nextParentIDsMap := map[int64]struct{}{}
		for _, batchFile := range batchFiles {
			if batchFile.ParentID == 0 {
				continue
			}
			if _, found := fileByID[batchFile.ParentID]; found {
				continue
			}
			nextParentIDsMap[batchFile.ParentID] = struct{}{}
		}

		for nextParentID := range nextParentIDsMap {
			pendingFileIDs = append(pendingFileIDs, nextParentID)
		}
	}

	for _, fileID := range uniqueFileIDs {
		if fileID == 0 {
			pathByFileID[fileID] = ""
			continue
		}

		pathFiles := []*enttenant.File{}
		seenFileIDs := map[int64]struct{}{}
		currentFileID := fileID

		for currentFileID != 0 {
			if _, found := seenFileIDs[currentFileID]; found {
				return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Detected cycle in directory tree.")
			}
			seenFileIDs[currentFileID] = struct{}{}

			currentFile, found := fileByID[currentFileID]
			if !found {
				return nil, e.NewHTTPErrorf(http.StatusNotFound, "Could not load path of file.")
			}

			pathFiles = append(pathFiles, currentFile)
			currentFileID = currentFile.ParentID
		}

		slices.Reverse(pathFiles)
		pathByFileID[fileID] = qq.fullPathFromPathFiles(pathFiles)
	}

	return pathByFileID, nil
}

func (qq *FileTree) FullPathsByFileIDX(ctx ctxx.Context, fileIDs []int64) map[int64]string {
	paths, err := qq.FullPathsByFileID(ctx, fileIDs)
	if err != nil {
		log.Println(err)
		panic(err)
	}
	return paths
}

func (qq *FileTree) FileByFullPath(ctx ctxx.Context, fullPath string) (*enttenant.File, error) {
	cleanPath := filepath.Clean(fullPath)
	if cleanPath == "." || cleanPath == "" || cleanPath == string(filepath.Separator) {
		return ctx.SpaceCtx().SpaceRootDir(), nil
	}

	pathElems := strings.Split(cleanPath, string(filepath.Separator))
	currentFile := ctx.SpaceCtx().SpaceRootDir()

	for _, pathElem := range pathElems {
		if pathElem == "" || pathElem == "." {
			continue
		}

		if !currentFile.IsDirectory {
			return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Path element is file, not a directory.")
		}

		nextFile, err := ctx.TenantCtx().TTx.File.Query().
			Where(
				file.SpaceID(ctx.SpaceCtx().Space.ID),
				file.ParentID(currentFile.ID),
				file.Name(pathElem),
			).
			Only(ctx)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		currentFile = nextFile
	}

	return currentFile, nil
}

func (qq *FileTree) IsDescendantOf(ctx ctxx.Context, fileID, ancestorFileID int64) (bool, error) {
	if fileID == 0 || ancestorFileID == 0 {
		return false, nil
	}

	seenFileIDs := map[int64]struct{}{}
	currentFileID := fileID
	for currentFileID != 0 {
		if currentFileID == ancestorFileID {
			return true, nil
		}

		if _, found := seenFileIDs[currentFileID]; found {
			return false, e.NewHTTPErrorf(http.StatusBadRequest, "Detected cycle in directory tree.")
		}
		seenFileIDs[currentFileID] = struct{}{}

		currentFile, err := ctx.TenantCtx().TTx.File.Query().
			Select(
				file.FieldID,
				file.FieldParentID,
			).
			Where(
				file.ID(currentFileID),
				file.SpaceID(ctx.SpaceCtx().Space.ID),
			).
			Only(ctx)
		if err != nil {
			log.Println(err)
			return false, err
		}

		currentFileID = currentFile.ParentID
	}

	return false, nil
}

func (qq *FileTree) DescendantIDsSubQuery(rootID, spaceID int64) *sql.Selector {
	filesTable := sql.Table(file.Table)
	recursiveFilesTable := sql.Table(file.Table).As("f")
	recursiveDescendantsTable := sql.Table("descendants").As("d")
	descendantsTable := sql.Table("descendants")

	anchor := sql.Select(filesTable.C(file.FieldID)).
		From(filesTable).
		Where(
			sql.And(
				sql.EQ(filesTable.C(file.FieldID), rootID),
				sql.EQ(filesTable.C(file.FieldSpaceID), spaceID),
				sql.IsNull(filesTable.C(file.FieldDeletedAt)),
			),
		)

	recursive := sql.Select(recursiveFilesTable.C(file.FieldID)).
		From(recursiveFilesTable).
		Join(recursiveDescendantsTable).
		On(recursiveFilesTable.C(file.FieldParentID), recursiveDescendantsTable.C("id")).
		Where(
			sql.And(
				sql.EQ(recursiveFilesTable.C(file.FieldSpaceID), spaceID),
				sql.IsNull(recursiveFilesTable.C(file.FieldDeletedAt)),
			),
		)

	withDescendants := sql.WithRecursive("descendants", "id").As(anchor.UnionAll(recursive))

	return sql.Select(descendantsTable.C("id")).
		From(descendantsTable).
		Where(sql.NEQ(descendantsTable.C("id"), rootID)).
		Prefix(withDescendants)
}

func (qq *FileTree) fullPathFromPathFiles(pathFiles []*enttenant.File) string {
	if len(pathFiles) <= 1 {
		return ""
	}

	pathElems := make([]string, 0, len(pathFiles)-1)
	for _, pathFile := range pathFiles[1:] {
		pathElems = append(pathElems, pathFile.Name)
	}

	return strings.Join(pathElems, string(filepath.Separator))
}
