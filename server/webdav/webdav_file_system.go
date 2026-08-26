package webdav

import (
	"context"
	"os"

	"golang.org/x/net/webdav"
)

type webDAVFileSystem struct{}

func (qq *webDAVFileSystem) Mkdir(_ context.Context, _ string, _ os.FileMode) error {
	return os.ErrPermission
}

func (qq *webDAVFileSystem) RemoveAll(_ context.Context, _ string) error {
	return os.ErrPermission
}

func (qq *webDAVFileSystem) Rename(ctx context.Context, oldName string, newName string) error {
	requestCtx, ok := webDAVFromContext(ctx)
	if !ok || requestCtx.op.method != "MOVE" {
		return os.ErrPermission
	}
	oldPath, ok := parseWebDAVSuffix(oldName)
	if !ok || !oldPath.isFile {
		return os.ErrPermission
	}
	newPath, ok := parseWebDAVSuffix(newName)
	if !ok || !newPath.isFile {
		return os.ErrPermission
	}
	if err := requestCtx.renameActiveResource(oldPath, newPath); err != nil {
		requestCtx.op.err = err
		return err
	}
	return nil
}

func (qq *webDAVFileSystem) OpenFile(
	ctx context.Context,
	name string,
	flag int,
	perm os.FileMode,
) (webdav.File, error) {
	requestCtx, ok := webDAVFromContext(ctx)
	if !ok {
		return nil, os.ErrNotExist
	}
	pathx, ok := parseWebDAVSuffix(name)
	if !ok {
		return nil, os.ErrNotExist
	}
	if pathx.isRoot || pathx.isInbox {
		return newWebDAVDirectory(pathx), nil
	}
	if requestCtx.op.method == "LOCK" {
		return &webDAVSyntheticFile{info: webDAVFileInfo{name: pathx.filename}}, nil
	}
	if requestCtx.op.method == "PUT" && flag&os.O_CREATE != 0 && pathx.isFile {
		return requestCtx.openUploadFile(pathx)
	}
	return nil, os.ErrNotExist
}

func (qq *webDAVFileSystem) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	pathx, ok := parseWebDAVSuffix(name)
	if !ok {
		return nil, os.ErrNotExist
	}
	if pathx.isRoot {
		return webDAVFileInfo{name: "/", isDir: true}, nil
	}
	if pathx.isInbox {
		return webDAVFileInfo{name: webDAVInbox, isDir: true}, nil
	}
	return nil, os.ErrNotExist
}
