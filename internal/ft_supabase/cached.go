package ft_supabase

import (
	"time"

	"github.com/google/uuid"
)

// NewUserCache creates a new UserCache instance.
// Returns an initialized UserCache with empty user maps.
func NewUserCache() *UserCache {
	return &UserCache{
		users:     make(map[string]*CachedUser),
		usersByID: make(map[uuid.UUID]*CachedUser),
	}
}

// Set stores a user in the cache using their access token as the key.
// token is the JWT access token used as the cache key.
// user is the CachedUser pointer to store.
// Also stores the user by UserID for lookup by ID.
// Thread-safe operation using write lock.
func (c *UserCache) Set(token string, user *CachedUser) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.users[token] = user
	c.usersByID[user.UserID] = user
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
		user *CachedUser
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
// Iterates through all cached users and removes expired ones.
// Thread-safe operation using write lock.
func (c *UserCache) Cleanup() {
	var (
		now         time.Time
		expiredKeys []string
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	now = time.Now()
	expiredKeys = make([]string, 0)

	// collect expired keys
	for token, user := range c.users {
		if now.After(user.ExpiresAt) {
			expiredKeys = append(expiredKeys, token)
		}
	}

	// delete expired users
	for _, token := range expiredKeys {
		delete(c.users, token)
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

