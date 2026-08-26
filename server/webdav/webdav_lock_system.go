package webdav

import (
	"sync"
	"time"

	"golang.org/x/net/webdav"
)

const webDAVMaxLocksPerCredential = 32

type webDAVLockSystem struct {
	inner webdav.LockSystem
	mu    sync.Mutex
	locks map[string]webDAVLockInfo
}

func newWebDAVLockSystem() *webDAVLockSystem {
	return &webDAVLockSystem{
		inner: webdav.NewMemLS(),
		locks: make(map[string]webDAVLockInfo),
	}
}

func (qq *webDAVLockSystem) forCredential(credentialID string) webdav.LockSystem {
	return &webDAVLockView{
		owner:        qq,
		credentialID: credentialID,
	}
}

func (qq *webDAVLockSystem) purgeExpired(now time.Time) {
	qq.mu.Lock()
	defer qq.mu.Unlock()
	for token, info := range qq.locks {
		if !info.expiresAt.After(now) {
			delete(qq.locks, token)
		}
	}
}

func (qq *webDAVLockSystem) lockCountLocked(credentialID string) int {
	count := 0
	for _, info := range qq.locks {
		if info.credentialID == credentialID {
			count++
		}
	}
	return count
}

func (qq *webDAVLockSystem) tokenBelongsToCredential(token string, credentialID string) bool {
	qq.mu.Lock()
	defer qq.mu.Unlock()
	info, ok := qq.locks[token]
	return ok && info.credentialID == credentialID
}
