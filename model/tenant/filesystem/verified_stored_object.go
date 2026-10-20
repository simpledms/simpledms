package filesystem

import (
	"errors"
	"os"
)

// VerifiedStoredObject holds a storage snapshot and the metadata calculated while reading its
// entire decrypted and decompressed content. The caller must close it to remove the local copy.
type VerifiedStoredObject struct {
	file   *os.File
	result savedFileResult
}

func newVerifiedStoredObject(file *os.File) *VerifiedStoredObject {
	return &VerifiedStoredObject{
		file: file,
	}
}

// Close releases the local storage snapshot.
func (qq *VerifiedStoredObject) Close() error {
	return errors.Join(qq.file.Close(), os.Remove(qq.file.Name()))
}

// StorageSize returns the size of the compressed and encrypted bytes.
func (qq *VerifiedStoredObject) StorageSize() int64 {
	return qq.result.storageSize
}

// StorageSHA256 returns the SHA-256 of the compressed and encrypted bytes.
func (qq *VerifiedStoredObject) StorageSHA256() string {
	return qq.result.storageSHA256
}

// StorageCRC32C returns the base64-encoded CRC32C of the storage bytes.
func (qq *VerifiedStoredObject) StorageCRC32C() string {
	return qq.result.storageCRC32C
}

// ContentSize returns the size of the decoded content.
func (qq *VerifiedStoredObject) ContentSize() int64 {
	return qq.result.fileSize
}

// ContentSHA256 returns the SHA-256 of the decoded content.
func (qq *VerifiedStoredObject) ContentSHA256() string {
	return qq.result.contentSHA256
}
