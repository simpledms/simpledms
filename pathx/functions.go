package pathx

import (
	"fmt"
	"log"
	"path/filepath"

	securejoin "github.com/cyphar/filepath-securejoin"
)

/*
func InboxPath(metaPath string) string {
	// not up to date
	return filepath.Clean(metaPath + "/inbox")
}
*/

// publicID is more robust against programming mistakes
func StoragePath(metaPath string, tenantID string) string {
	return filepath.Clean(fmt.Sprintf("%s/tenants/%s/storage", metaPath, tenantID))
}

// used for selecting encryption key
func S3TenantPrefix() string {
	return filepath.Clean(fmt.Sprintf("tenants"))
}

// publicID is more robust against programming mistakes
func S3StoragePrefix(tenantID string) string {
	return filepath.Clean(fmt.Sprintf("%s/%s/files", S3TenantPrefix(), tenantID))
}

func S3TemporaryStoragePrefix(tenantID string) string {
	return filepath.Clean(fmt.Sprintf("%s/%s/tmp", S3TenantPrefix(), tenantID))
}

func S3SqliteReplicationPrefix(tenantID string) string {
	return filepath.Clean(fmt.Sprintf("%s/%s/sqlite", S3TenantPrefix(), tenantID))
}

// used for selecting encryption key
func S3AccountPrefix() string {
	return filepath.Clean(fmt.Sprintf("accounts"))
}

func S3TemporaryAccountStoragePrefix(accountID string) string {
	return filepath.Clean(fmt.Sprintf("%s/%s/tmp", S3AccountPrefix(), accountID))
}

func TenantDBPath(metaPath string, tenantID string) (string, error) {
	tenantDBPath, err := securejoin.SecureJoin(metaPath, fmt.Sprintf("tenants/%s", tenantID))
	if err != nil {
		log.Println(err)
		return "", err
	}
	return tenantDBPath, nil
}
