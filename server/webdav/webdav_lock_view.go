package webdav

import (
	"time"

	"golang.org/x/net/webdav"
)

const webDAVMaxLockDuration = time.Hour

type webDAVLockView struct {
	owner        *webDAVLockSystem
	credentialID string
}

func (qq *webDAVLockView) Confirm(
	now time.Time,
	name0 string,
	name1 string,
	conditions ...webdav.Condition,
) (func(), error) {
	qq.owner.purgeExpired(now)
	return qq.owner.inner.Confirm(
		now,
		qq.namespaced(name0),
		qq.namespaced(name1),
		conditions...,
	)
}

func (qq *webDAVLockView) Create(now time.Time, details webdav.LockDetails) (string, error) {
	qq.owner.purgeExpired(now)
	qq.owner.mu.Lock()
	defer qq.owner.mu.Unlock()
	if qq.owner.lockCountLocked(qq.credentialID) >= webDAVMaxLocksPerCredential {
		return "", webdav.ErrLocked
	}

	details.Root = qq.namespaced(details.Root)
	details.Duration = cappedLockDuration(details.Duration)
	token, err := qq.owner.inner.Create(now, details)
	if err != nil {
		return "", err
	}

	qq.owner.locks[token] = webDAVLockInfo{
		credentialID: qq.credentialID,
		expiresAt:    now.Add(details.Duration),
	}
	return token, nil
}

func (qq *webDAVLockView) Refresh(
	now time.Time,
	token string,
	duration time.Duration,
) (webdav.LockDetails, error) {
	qq.owner.purgeExpired(now)
	if !qq.owner.tokenBelongsToCredential(token, qq.credentialID) {
		return webdav.LockDetails{}, webdav.ErrNoSuchLock
	}
	duration = cappedLockDuration(duration)
	details, err := qq.owner.inner.Refresh(now, token, duration)
	if err != nil {
		return details, err
	}
	qq.owner.mu.Lock()
	info := qq.owner.locks[token]
	info.expiresAt = now.Add(duration)
	qq.owner.locks[token] = info
	qq.owner.mu.Unlock()
	return details, nil
}

func (qq *webDAVLockView) Unlock(now time.Time, token string) error {
	qq.owner.purgeExpired(now)
	if !qq.owner.tokenBelongsToCredential(token, qq.credentialID) {
		return webdav.ErrNoSuchLock
	}
	if err := qq.owner.inner.Unlock(now, token); err != nil {
		return err
	}
	qq.owner.mu.Lock()
	delete(qq.owner.locks, token)
	qq.owner.mu.Unlock()
	return nil
}

func (qq *webDAVLockView) namespaced(name string) string {
	if name == "" {
		return ""
	}
	return "/cred/" + qq.credentialID + name
}

func cappedLockDuration(duration time.Duration) time.Duration {
	if duration <= 0 || duration > webDAVMaxLockDuration {
		return webDAVMaxLockDuration
	}
	return duration
}
