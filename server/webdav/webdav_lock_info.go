package webdav

import "time"

type webDAVLockInfo struct {
	credentialID string
	expiresAt    time.Time
}
