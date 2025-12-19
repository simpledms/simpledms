package common

type Factory struct {
	// TODO on factory or infra?
	// inboxDirInfo   *ent.FileInfo
	// storageDirInfo *ent.FileInfo
}

func NewFactory() *Factory {
	return &Factory{
		// inboxDirInfo:   inboxDirInfo,
		// storageDirInfo: storageDirInfo,
	}
}

/*
func (qq *Factory) InboxDirInfo() *ent.FileInfo {
	return qq.inboxDirInfo
}

func (qq *Factory) StorageDirInfo() *ent.FileInfo {
	return qq.storageDirInfo
}
*/
