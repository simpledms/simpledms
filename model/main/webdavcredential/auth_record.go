package webdavcredential

import "time"

type AuthRecord struct {
	ID             int64
	PublicID       string
	AccountID      int64
	TenantID       int64
	TenantPublicID string
	SpacePublicID  string
	SecretSalt     string
	SecretHash     string
	RevokedAt      *time.Time
	LastUsedAt     *time.Time
}
