package session

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashValue returns the stored form of a Session cookie value. Only the hash is persisted,
// so a copy of the database cannot be used to take over Sessions. SHA-256 without a salt
// is sufficient because cookie values are 64 random characters and cannot be guessed.
func HashValue(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
