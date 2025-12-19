package webdav

/*
import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/net/webdav"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ent"
	"github.com/simpledms/simpledms/ent/fileinfo"
	"github.com/simpledms/simpledms/ctxx"
)

type Dir struct {
	infra       *common.Infra
	dir         webdav.Dir
	storagePath string
}

func NewDir(infra *common.Infra, dir webdav.Dir, storagePath string) *Dir {
	return &Dir{
		infra:       infra,
		dir:         dir,
		storagePath: storagePath,
	}
}

func (qq *Dir) parentDir(ctx context.Context, tx *ent.TTx, relPath string) (*ent.File, string, error) {
	relPath = filepath.Clean(relPath)
	parentPath := filepath.Dir(relPath)

	// handle root
	if parentPath == "." {
		parentPath = ""
	}
	// TODO is directory? or implicit? or doesn't matter?
	parentPathX, err := tx.FileInfo.Query().Where(fileinfo.FullPath(parentPath)).Only(ctx)
	if err != nil { // && !ent.IsNotFound(err) {
		if ent.IsNotFound(err) {
			return nil, parentPath, os.ErrNotExist
		}
		log.Println(err)
		// TODO what is a good error code?
		return nil, parentPath, os.ErrInvalid
	}

	// found
	parentDir := tx.File.GetX(ctx, parentPathX.FileID)
	if !parentDir.IsDirectory {
		log.Println("parent path is a file, was", parentPath)
		// TODO correct error code
		return nil, parentPath, os.ErrInvalid
	}

	return parentDir, parentPath, nil
}

func (qq *Dir) wrapTx(ctx context.Context, fn func(tx *ent.TTx) error) error {
	tx, err := qq.infra.UnsafeDB().TTx(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	err = fn(tx)
	if err != nil {
		log.Println(err)
		if err := tx.Rollback(); err != nil {
			log.Println(err)
		}
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (qq *Dir) Mkdir(ctx context.Context, relPath string, perm os.FileMode) error {
	return qq.wrapTx(ctx, func(tx *ent.TTx) error {
		return qq.mkdir(
			ctx,
			relPath,
			perm,
		)
	})
}

func (qq *Dir) OpenFile(ctx context.Context, relPath string, flags int, perm os.FileMode) (webdav.File, error) {
	var file webdav.File
	err := qq.wrapTx(ctx, func(tx *ent.TTx) error {
		var err error
		file, err = qq.openFile(ctx, relPath, flags, perm)
		return err
	})
	return file, err
}

func (qq *Dir) RemoveAll(ctx context.Context, relPath string) error {
	return qq.wrapTx(ctx, func(tx *ent.TTx) error {
		return qq.removeAll(ctx, relPath)
	})
}

func (qq *Dir) Rename(ctx context.Context, oldRelPath, newRelPath string) error {
	return qq.wrapTx(ctx, func(tx *ent.TTx) error {
		return qq.rename(ctx, oldRelPath, newRelPath)
	})
}

func (qq *Dir) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	var info os.FileInfo
	err := qq.wrapTx(ctx, func(tx *ent.TTx) error {
		var err error
		info, err = qq.stat(ctx, name)
		return err
	})
	return info, err
}

// FIXME handle panics
//
// `mkdir -p x/y/z` goes one by one and calls `mkdir x` first, then `mkdir x/y`, etc.
// old impl which can handle `mkdir x/y/z is found below
//
// 1. check if exists
// 2. find longest existing path
// 3. create non existing ones one by one (must be conc save with indexer)
func (qq *Dir) mkdir(ctx *ctxx.Context, relPath string, perm os.FileMode) error {
	log.Println("mkdir", relPath)
	relPath = filepath.Clean(relPath)

	// check if already exists
	exists := ctx.TenantCtx().TTx.FileInfo.Query().Where(fileinfo.FullPath(relPath)).ExistX(ctx)
	if exists {
		// TODO correct
		return os.ErrExist
	}

	parentDir, parentPath, err := qq.parentDir(ctx, relPath)
	if err != nil {
		log.Println(err)
		return err
	}

	// TODO separator `/` is os independent in WebDAV? or use filepath.SplitList instead?
	toCreate := strings.TrimPrefix(relPath, parentPath+"/")

	// TODO set modified_at?
	_ = ctx.TenantCtx().TTx.File.Create().
		SetName(toCreate).
		SetIsDirectory(true).
		SetIndexedAt(time.Now()). // TODO correct?
		SetParentID(parentDir.ID).
		SaveX(ctx)

	// TODO or use os.Mkdir? qq.dir.Mkdir has addtional safety check which makes securejoin
	//		unncessary?
	err = qq.dir.Mkdir(ctx, relPath, perm)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// can be dir or file
func (qq *Dir) openFile(ctx *ctxx.Context, relPath string, flags int, perm os.FileMode) (webdav.File, error) {
	log.Println("openfile", relPath)
	relPath = filepath.Clean(relPath)

	if relPath == "." {
		return qq.dir.OpenFile(ctx, relPath, flags, perm)
	}

	// check if already exists
	// TODO panics some times
	exists := ctx.TenantCtx().TTx.FileInfo.Query().Where(fileinfo.FullPath(relPath)).ExistX(ctx)
	if exists {
		// FIXME update mod time and size if opened for writing
		return qq.dir.OpenFile(ctx, relPath, flags, perm)
	}

	openedForCreation := flags&os.O_CREATE != 0
	if !openedForCreation {
		return nil, os.ErrNotExist
	}

	filename := filepath.Base(relPath)
	file := ctx.TenantCtx().TTx.File.Create().
		SetName(filename).
		SetIsDirectory(false).   // TODO can this also be used for directory creation?
		SetIndexedAt(time.Now()) // TODO correct?

	parentDir, _, err := qq.parentDir(ctx, relPath)
	if err != nil {
		log.Println(err, relPath)
		return nil, err
	}
	file.SetParentID(parentDir.ID)

	file.SaveX(ctx)

	// FIXME set/update modified_at and size?
	return qq.dir.OpenFile(ctx, relPath, flags, perm)
}

func (qq *Dir) removeAll(ctx *ctxx.Context, relPath string) error {
	relPath = filepath.Clean(relPath)

	fileInfo, err := ctx.TenantCtx().TTx.FileInfo.Query().Where(fileinfo.FullPath(relPath)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return os.ErrNotExist
		}
		log.Println(err)
		return err
	}

	// FIXME move file to trash // impl in model
	ctx.TenantCtx().TTx.File.DeleteOneID(fileInfo.FileID).ExecX(ctx)

	return qq.dir.RemoveAll(ctx, relPath)
}

func (qq *Dir) rename(ctx *ctxx.Context, oldRelPath, newRelPath string) error {
	oldRelPath = filepath.Clean(oldRelPath)
	newRelPath = filepath.Clean(newRelPath)
	log.Println("rename", oldRelPath, newRelPath)

	// Dir and Base and not Split because Split doesn't call Clean on dir
	oldParentDir := filepath.Dir(oldRelPath)
	newParentDir := filepath.Dir(newRelPath)
	newName := filepath.Base(newRelPath)

	oldFileInfo := ctx.TenantCtx().TTx.FileInfo.Query().Where(fileinfo.FullPath(oldRelPath)).OnlyX(ctx)
	oldFile := ctx.TenantCtx().TTx.File.GetX(ctx, oldFileInfo.FileID)

	if newParentDir == oldParentDir { // change name
		// TODO set modified date?
		oldFile.Update().SetName(newName).SaveX(ctx)

		return qq.dir.Rename(ctx, oldRelPath, newRelPath)
	}

	// handle overwrite
	nullableFileDestInfo, err := ctx.TenantCtx().TTx.FileInfo.Query().Where(fileinfo.FullPath(newRelPath)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) { // isNotFound is normal path
		log.Println(err)
		return os.ErrInvalid // TODO okay?
	}
	if nullableFileDestInfo != nil { // found
		// FIXME move file to trash // impl in model
		ctx.TenantCtx().TTx.File.DeleteOneID(nullableFileDestInfo.FileID).ExecX(ctx)

		// fileDest := qq.infra.DB().File.GetX(ctx, nullableFileDestInfo.FileID)
		// fileDest.Update().SetDeletedAt(time.Now()).SaveX(ctx)
	}

	// TODO is it save to assume newParent is a directory? probably because we use filepath.Dir()
	newParent := ctx.TenantCtx().TTx.FileInfo.Query().Where(fileinfo.FullPath(newParentDir)).OnlyX(ctx)
	oldFile.Update().SetName(newName).SetParentID(newParent.FileID).SaveX(ctx)

	return qq.dir.Rename(ctx, oldRelPath, newRelPath)
}

func (qq *Dir) stat(ctx *ctxx.Context, name string) (os.FileInfo, error) {
	log.Println("stat", name)
	return qq.dir.Stat(ctx, name)
}

/*
func (qq *Dir) Mkdir(ctx context.Context, relPath string, perm os.FileMode) error {
	relPath = filepath.Clean(relPath)

	// check if already exists
	exists := qq.infra.DB().FilePath.Query().Where(filepathx.FullPath(relPath)).ExistX(ctx)
	if exists {
		// TODO correct
		return os.ErrExist
	}

	var dirFound *ent.File
	var toCreateSlice []string

	for {
		parentPath := filepath.Dir(relPath)

		// TODO is directory? or implicit? or doesn't matter?
		parentPathX, err := qq.infra.DB().FilePath.Query().Where(filepathx.FullPath(parentPath)).Only(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			continue // try next path
		}
		if err != nil {
			log.Println(err)
			// TODO what is a good error code?
			return os.ErrInvalid
		}

		// found
		dirFound = qq.infra.DB().File.GetX(ctx, parentPathX.FileID)
		if !dirFound.IsDirectory {
			log.Println("parent path is a file, was", parentPath)
			// TODO correct error code
			return os.ErrInvalid
		}

		toCreate := strings.TrimPrefix(relPath, parentPath+"/")
		// TODO os independent in WebDAV? or use filepath.SplitList instead?
		toCreateSlice = strings.Split(toCreate, "/")

		break
	}

	lastParentID := dirFound.ID

	for _, toCreate := range toCreateSlice {
		createdDir := qq.infra.DB().File.Create().
			SetName(toCreate).
			SetIsDirectory(true).
			SetIndexedAt(time.Now()). // TODO correct?
			SetParentID(lastParentID).
			SaveX(ctx)

		lastParentID = createdDir.ID

	}

	pwd, err := os.Getwd()
	if err != nil {
		log.Println(err)
		return err
	}

	// also has a safer alternative to os.MkdirAll, but works just on Linux
	absPath, err := securejoin.SecureJoin(pwd, relPath)
	if err != nil {
		log.Println(err)
		return err
	}

	// TODO use `perm` param instead?
	err = os.MkdirAll(absPath, os.ModePerm|os.ModeDir)
	if err != nil {
		log.Println(err)
		return err
	}

	// 1. check if exists
	// 2. find longest existing path
	// 3. create non existing ones one by one (must be conc save with indexer)

	// log.Println(relPath)
	return nil
}
*/
