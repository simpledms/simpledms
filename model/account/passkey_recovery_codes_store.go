package account

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type PasskeyRecoveryCodesStore struct {
	mu        sync.Mutex
	entries   map[string]string
	expiresAt map[string]time.Time
}

func NewPasskeyRecoveryCodesStore() *PasskeyRecoveryCodesStore {
	return &PasskeyRecoveryCodesStore{
		entries:   make(map[string]string),
		expiresAt: make(map[string]time.Time),
	}
}

func (qq *PasskeyRecoveryCodesStore) Store(codes []string) string {
	qq.mu.Lock()
	defer qq.mu.Unlock()

	qq.deleteExpiredLocked()

	token := "recovery-codes-" + uuid.NewString()
	qq.entries[token] = strings.Join(codes, "\n")
	qq.expiresAt[token] = time.Now().Add(5 * time.Minute)

	return token
}

func (qq *PasskeyRecoveryCodesStore) Consume(token string) (string, bool) {
	qq.mu.Lock()
	defer qq.mu.Unlock()

	qq.deleteExpiredLocked()

	normalizedToken := strings.TrimSpace(token)
	entry, ok := qq.entries[normalizedToken]
	if !ok {
		return "", false
	}

	delete(qq.entries, normalizedToken)
	delete(qq.expiresAt, normalizedToken)
	return entry, true
}

func (qq *PasskeyRecoveryCodesStore) deleteExpiredLocked() {
	now := time.Now()
	for token, expiresAt := range qq.expiresAt {
		if expiresAt.Before(now) {
			delete(qq.expiresAt, token)
			delete(qq.entries, token)
		}
	}
}
