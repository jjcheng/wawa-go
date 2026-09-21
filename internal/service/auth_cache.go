package service

import (
	"sync"
	"time"

	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
)

const authCacheTTL = 15 * time.Second

type authCacheEntry struct {
	user      dto_account.User
	expiresAt time.Time
}

type AuthCache struct {
	mu      sync.RWMutex
	entries map[string]authCacheEntry
}

func NewAuthCache() *AuthCache {
	return &AuthCache{entries: make(map[string]authCacheEntry)}
}

func (cache *AuthCache) Get(tokenHash string, now time.Time) (*dto_account.User, bool) {
	cache.mu.RLock()
	entry, ok := cache.entries[tokenHash]
	cache.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if !entry.expiresAt.After(now) || entry.user.Session == nil || !entry.user.Session.ExpiresAt.After(now) || entry.user.Session.RevokedAt != nil {
		cache.Delete(tokenHash)
		return nil, false
	}
	user := cloneCachedUser(entry.user)
	return &user, true
}

func (cache *AuthCache) Set(tokenHash string, user *dto_account.User, now time.Time) {
	if user == nil || user.Session == nil {
		return
	}
	expiresAt := now.Add(authCacheTTL)
	if user.Session.ExpiresAt.Before(expiresAt) {
		expiresAt = user.Session.ExpiresAt
	}
	if !expiresAt.After(now) {
		return
	}
	cache.mu.Lock()
	cache.entries[tokenHash] = authCacheEntry{user: cloneCachedUser(*user), expiresAt: expiresAt}
	cache.mu.Unlock()
}

func (cache *AuthCache) Delete(tokenHash string) {
	cache.mu.Lock()
	delete(cache.entries, tokenHash)
	cache.mu.Unlock()
}

func (cache *AuthCache) InvalidateSession(sessionID int32) {
	cache.mu.Lock()
	for tokenHash, entry := range cache.entries {
		if entry.user.Session != nil && entry.user.Session.Id == sessionID {
			delete(cache.entries, tokenHash)
		}
	}
	cache.mu.Unlock()
}

func (cache *AuthCache) InvalidateUser(userID int32) {
	cache.mu.Lock()
	for tokenHash, entry := range cache.entries {
		if entry.user.Id == userID {
			delete(cache.entries, tokenHash)
		}
	}
	cache.mu.Unlock()
}

func cloneCachedUser(user dto_account.User) dto_account.User {
	clone := user
	if user.AccessTokenExpiry != nil {
		expiry := *user.AccessTokenExpiry
		clone.AccessTokenExpiry = &expiry
	}
	if user.Session != nil {
		session := *user.Session
		if user.Session.RevokedAt != nil {
			revokedAt := *user.Session.RevokedAt
			session.RevokedAt = &revokedAt
		}
		clone.Session = &session
	}
	if user.WA != nil {
		wa := *user.WA
		if user.WA.PhoneNumber_ != nil {
			phoneNumber := *user.WA.PhoneNumber_
			wa.PhoneNumber_ = &phoneNumber
		}
		if user.WA.BusinessAccount != nil {
			businessAccount := *user.WA.BusinessAccount
			wa.BusinessAccount = &businessAccount
		}
		if user.WA.BusinessPortfolio != nil {
			businessPortfolio := *user.WA.BusinessPortfolio
			wa.BusinessPortfolio = &businessPortfolio
		}
		clone.WA = &wa
	}
	return clone
}
