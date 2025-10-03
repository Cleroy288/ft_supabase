package ft_supabase

import (
	"time"

	"github.com/google/uuid"
)

// NewUserCache creates a new UserCache instance.
// Returns an initialized UserCache with empty user maps and default max size of 1000.
func NewUserCache() *UserCache {
	Log("NewUserCache", "Creating new user cache with max size: 1000")
	return &UserCache{
		users:     make(map[string]*CachedUser),
		usersByID: make(map[uuid.UUID]*CachedUser),
		MaxSize:   1000,
	}
}

// Set stores a user in the cache using their access token as the key.
// token is the JWT access token used as the cache key.
// user is the CachedUser pointer to store.
// Also stores the user by UserID for lookup by ID.
// If cache size reaches MaxSize, evicts oldest cached users first.
// Thread-safe operation using write lock.
func (c *UserCache) Set(token string, user *CachedUser) {
	var (
		now           time.Time
		oldestToken   string
		oldestUser    *CachedUser
		currentUser   *CachedUser
		needsEviction bool
		expiredCount  int
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	Logf("UserCache.Set", "Caching user - UserID: %s, Email: %s", user.UserID.String(), user.Email)

	// check if cache is full and needs eviction
	if len(c.users) >= c.MaxSize {
		Logf("UserCache.Set", "Cache full (%d/%d), cleaning expired entries", len(c.users), c.MaxSize)

		// first try to remove expired entries
		now = time.Now()
		expiredCount = 0
		for t, u := range c.users {
			if now.After(u.ExpiresAt) {
				delete(c.users, t)
				delete(c.usersByID, u.UserID)
				expiredCount++
			}
		}

		if expiredCount > 0 {
			Logf("UserCache.Set", "Removed %d expired entries", expiredCount)
		}

		// if still at max capacity after cleanup, evict oldest entry
		if len(c.users) >= c.MaxSize {
			Logf("UserCache.Set", "Still at max capacity after cleanup, evicting oldest user")
			needsEviction = true
			for t, u := range c.users {
				if oldestUser == nil || u.CachedAt.Before(oldestUser.CachedAt) {
					oldestToken = t
					oldestUser = u
				}
			}

			// evict oldest user
			if needsEviction && oldestUser != nil {
				Logf("UserCache.Set", "Evicting oldest user - UserID: %s, Email: %s", oldestUser.UserID.String(), oldestUser.Email)
				delete(c.users, oldestToken)
				delete(c.usersByID, oldestUser.UserID)
			}
		}
	}

	// check if user already exists (update case)
	currentUser, _ = c.usersByID[user.UserID]
	if currentUser != nil {
		Log("UserCache.Set", "Updating existing user in cache")
		// remove old token entry if token changed
		if currentUser.AccessToken != token {
			delete(c.users, currentUser.AccessToken)
		}
	}

	// store new user
	c.users[token] = user
	c.usersByID[user.UserID] = user

	Logf("UserCache.Set", "Successfully cached user - Total users: %d", len(c.users))
}

// Get retrieves a user from the cache by their access token.
// token is the JWT access token used as the cache key.
// Returns the CachedUser pointer and true if found and not expired.
// Returns nil and false if not found or expired.
// Thread-safe operation using read lock.
func (c *UserCache) Get(token string) (*CachedUser, bool) {
	var (
		user   *CachedUser
		exists bool
	)

	c.mu.RLock()
	defer c.mu.RUnlock()

	user, exists = c.users[token]
	if !exists {
		return nil, false
	}

	// check if token is expired
	if time.Now().After(user.ExpiresAt) {
		return nil, false
	}

	return user, true
}

// Delete removes a user from the cache by their access token.
// token is the JWT access token used as the cache key.
// Thread-safe operation using write lock.
func (c *UserCache) Delete(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.users, token)
}

// DeleteByUserID removes a user from the cache by their UserID.
// userID is the Supabase user unique identifier (UUID).
// Removes from both token and userID indexes.
// Thread-safe operation using write lock.
func (c *UserCache) DeleteByUserID(userID uuid.UUID) {
	var (
		user   *CachedUser
		exists bool
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	// get user to find their token
	user, exists = c.usersByID[userID]
	if exists {
		// delete from both maps
		delete(c.users, user.AccessToken)
		delete(c.usersByID, userID)
	}
}

// IsValid checks if a token exists in cache and is not expired.
// token is the JWT access token to validate.
// Returns true if token exists and is valid, false otherwise.
// Thread-safe operation using read lock.
func (c *UserCache) IsValid(token string) bool {
	var (
		user   *CachedUser
		exists bool
	)

	c.mu.RLock()
	defer c.mu.RUnlock()

	user, exists = c.users[token]
	if !exists {
		return false
	}

	// check if token is expired
	return time.Now().Before(user.ExpiresAt)
}

// Cleanup removes all expired tokens from the cache.
// Iterates through all cached users and removes expired ones from both indexes.
// Thread-safe operation using write lock.
func (c *UserCache) Cleanup() {
	var (
		now           time.Time
		expiredTokens []string
		expiredUIDs   []uuid.UUID
		beforeCount   int
		afterCount    int
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	beforeCount = len(c.users)
	Logf("UserCache.Cleanup", "Starting cache cleanup - Current users: %d", beforeCount)

	now = time.Now()
	expiredTokens = make([]string, 0)
	expiredUIDs = make([]uuid.UUID, 0)

	// collect expired entries
	for token, user := range c.users {
		if now.After(user.ExpiresAt) {
			expiredTokens = append(expiredTokens, token)
			expiredUIDs = append(expiredUIDs, user.UserID)
		}
	}

	// delete expired users from both maps
	for _, token := range expiredTokens {
		delete(c.users, token)
	}
	for _, userID := range expiredUIDs {
		delete(c.usersByID, userID)
	}

	afterCount = len(c.users)

	if len(expiredTokens) > 0 {
		Logf("UserCache.Cleanup", "Removed %d expired entries - Remaining users: %d", len(expiredTokens), afterCount)
	} else {
		Logf("UserCache.Cleanup", "No expired entries found - Remaining users: %d", afterCount)
	}
}

// Count returns the number of users currently in the cache.
// Thread-safe operation using read lock.
func (c *UserCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.users)
}

// GetByUserID retrieves a user from the cache by their UserID.
// userID is the Supabase user unique identifier (UUID).
// Returns the CachedUser pointer and true if found and not expired.
// Returns nil and false if not found or expired.
// Thread-safe operation using read lock.
func (c *UserCache) GetByUserID(userID uuid.UUID) (*CachedUser, bool) {
	var (
		user   *CachedUser
		exists bool
	)

	c.mu.RLock()
	defer c.mu.RUnlock()

	user, exists = c.usersByID[userID]
	if !exists {
		return nil, false
	}

	// check if token is expired
	if time.Now().After(user.ExpiresAt) {
		return nil, false
	}

	return user, true
}
