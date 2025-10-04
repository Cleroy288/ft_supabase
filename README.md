# ft_supabase

A Go client library for Supabase Authentication with built-in caching, automatic cleanup, and session management.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [API Reference](#api-reference)
  - [Service](#service)
  - [Cache](#cache)
  - [Models](#models)
- [Cache Management](#cache-management)
- [Error Handling](#error-handling)
- [Thread Safety](#thread-safety)
- [Examples](#examples)

## Overview

`ft_supabase` is a comprehensive Go package that provides authentication and user management functionality for Supabase applications. It includes a thread-safe caching system with automatic cleanup, support for custom user metadata, and a clean interface for all common authentication operations.

## Features

- **User Registration** - Create new users with email, password, and custom metadata
- **User Authentication** - Login/logout users and receive JWT access tokens
- **Token Refresh** - Refresh access tokens using refresh tokens
- **User Management** - Retrieve, update, and delete users
- **Session Caching** - Thread-safe in-memory cache with intelligent eviction
- **Automatic Cache Cleanup** - Background goroutine removes expired tokens every 24 hours
- **Cache Size Limits** - Configurable max cache size (default 1000 users) with LRU eviction
- **Safe Type Assertions** - Panic-free metadata extraction
- **Custom Metadata** - Support for custom user fields (username, role, display name, etc.)
- **Multiple Authentication Levels** - Anon key for public operations, service role key for admin operations
- **Context Support** - All operations support context for cancellation and timeout
- **Type Safety** - Strongly typed models with UUID support

## Installation

```bash
go get github.com/Cleroy288/ft_supabase
```

### Dependencies

- Go 1.25.1 or higher
- `github.com/google/uuid` v1.6.0

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    ft_supabase "github.com/Cleroy288/ft_supabase"
)

func main() {
    // Initialize the service
    service := ft_supabase.NewService(
        "project-id",
        "https://project.supabase.co",
        "anon-key",
        "service-role-key",
    )
    // Prints: [ft_supabase] [NewService] Creating new Supabase service - ProjectID: project-id, ProjectURL: https://project.supabase.co
    // Prints: [ft_supabase] [NewService] Successfully created Supabase service instance

    // Start automatic cache cleanup (runs every 24 hours)
    service.StartCacheCleanup()
    defer service.StopCacheCleanup()

    // Register a new user
    metadata := ft_supabase.UserMetadata{
        Username:    "johndoe",
        DisplayName: "John Doe",
        Role:        "user",
    }

    response, err := service.RegisterUser(
        context.Background(),
        "user@example.com",
        "password123",
        "+1234567890",
        metadata,
    )
    if err != nil {
        fmt.Printf("Registration failed: %v\n", err)
        return
    }

    fmt.Printf("User registered: %s\n", response.ID)
}
```

## Architecture

The library is organized into the following files:

### ft_supabase Package

- **service.go** - Main service implementation with authentication functions and cache management
- **models.go** - Data structures and type definitions
- **cached.go** - Thread-safe cache implementation with eviction and cleanup
- **logger.go** - Simple context-based logging system
- **utils.go** - HTTP client utilities for making API requests
- **headers.go** - HTTP header constants and helper functions
- **endpoints.go** - Supabase API endpoint constants

## API Reference

### Service

The `Service` struct is the main entry point for all authentication operations.

#### NewService

Creates a new Supabase service instance. Prints initialization status on creation.

```go
func NewService(projectID, projectURL, anonKey, serviceKey string) *Service
```

**Parameters:**
- `projectID` - Supabase project identifier
- `projectURL` - Base URL for the Supabase project API (e.g., `https://project.supabase.co`)
- `anonKey` - Anonymous/public API key for client-side operations
- `serviceKey` - Service role key for privileged server-side operations

**Returns:** Initialized `*Service` with HTTP client and cache (max size: 1000)

**Initialization Output:**
```go
service := ft_supabase.NewService("project-id", "https://project.supabase.co", "anon-key", "service-key")
// Prints: [ft_supabase] [NewService] Creating new Supabase service - ProjectID: project-id, ProjectURL: https://project.supabase.co
// Prints: [ft_supabase] [NewService] Successfully created Supabase service instance
```

This helps verify the service was initialized correctly with the expected configuration.

---

#### StartCacheCleanup

Starts a background goroutine that cleans expired cache entries every 24 hours.

```go
func (s *Service) StartCacheCleanup()
```

**Behavior:**
- Runs cleanup immediately on start
- Repeats every 24 hours using `time.Ticker`
- Removes expired sessions from both token and userID indexes
- Thread-safe operation
- Should be called after creating service

**Example:**
```go
service := ft_supabase.NewService(projectID, projectURL, anonKey, serviceKey)
service.StartCacheCleanup()
defer service.StopCacheCleanup() // Always cleanup on shutdown
```

---

#### StopCacheCleanup

Stops the background cache cleanup goroutine.

```go
func (s *Service) StopCacheCleanup()
```

**Behavior:**
- Signals cleanup goroutine to stop
- Prevents goroutine leaks on service shutdown
- Safe to call multiple times
- Should be called when shutting down the service

---

#### RegisterUser

Registers a new user with Supabase Auth API.

```go
func (s *Service) RegisterUser(
    ctx context.Context,
    email, password, phone string,
    metadata UserMetadata,
) (*RegisterResponse, error)
```

**Parameters:**
- `ctx` - Context for request cancellation and timeout
- `email` - User's email address
- `password` - User's password
- `phone` - User's phone number (optional, can be empty string)
- `metadata` - Custom user metadata stored in Supabase `user_metadata` field

**Returns:**
- `*RegisterResponse` - Contains user ID, username, email, and role
- `error` - Error if registration fails

**Behavior:**
- Sends POST request to `/auth/v1/signup`
- Stores user session in cache with JWT token as key
- Parses user ID to UUID format
- Safely extracts custom metadata from response (no panics)
- Triggers cache eviction if max size reached

---

#### LoginUser

Authenticates a user and returns a JWT access token.

```go
func (s *Service) LoginUser(
    ctx context.Context,
    email, password string,
) (*LoginResponse, error)
```

**Parameters:**
- `ctx` - Context for request cancellation and timeout
- `email` - User's email address
- `password` - User's password

**Returns:**
- `*LoginResponse` - Contains JWT token, user ID, email, username, and role
- `error` - Error if authentication fails

**Behavior:**
- Sends POST request to `/auth/v1/token?grant_type=password`
- Stores user session in cache with JWT token as key
- Returns access token for subsequent authenticated requests
- Safely extracts metadata fields

---

#### Logout

Invalidates a user session in Supabase and removes from cache.

```go
func (s *Service) Logout(
    ctx context.Context,
    token string,
) error
```

**Parameters:**
- `ctx` - Context for request cancellation and timeout
- `token` - JWT access token to invalidate

**Returns:**
- `error` - Error if logout fails

**Behavior:**
- Sends POST request to `/auth/v1/logout`
- Removes user session from cache
- Invalidates token in Supabase

---

#### RefreshToken

Refreshes an access token using a refresh token.

```go
func (s *Service) RefreshToken(
    ctx context.Context,
    refreshToken string,
) (*RefreshTokenResponse, error)
```

**Parameters:**
- `ctx` - Context for request cancellation and timeout
- `refreshToken` - Refresh token obtained during login or registration

**Returns:**
- `*RefreshTokenResponse` - Contains new access token, refresh token, and user details
- `error` - Error if refresh fails

**Behavior:**
- Sends POST request to `/auth/v1/token?grant_type=refresh_token`
- Removes old token from cache
- Stores new session with updated tokens
- Handles Supabase's 10-second token reuse window

---

#### GetUserByID

Retrieves a user by their ID from the cache.

```go
func (s *Service) GetUserByID(
    ctx context.Context,
    userID uuid.UUID,
) (*User, error)
```

**Parameters:**
- `ctx` - Context for request cancellation and timeout
- `userID` - Supabase user unique identifier (UUID)

**Returns:**
- `*User` - User object with all user details
- `error` - Returns `ErrUserNotFound` if user not in cache

**Behavior:**
- Looks up user in cache by UUID
- Returns cached user data without making API call
- Validates token expiration

---

#### GetCurrentUser

Retrieves the current user by their JWT token from the cache.

```go
func (s *Service) GetCurrentUser(
    ctx context.Context,
    token string,
) (*User, error)
```

**Parameters:**
- `ctx` - Context for request cancellation and timeout
- `token` - JWT access token

**Returns:**
- `*User` - User object with all user details
- `error` - Returns `ErrUserNotFound` if token not in cache

**Behavior:**
- Looks up user in cache by token
- Returns cached user data without making API call
- Validates token expiration

---

#### UpdateUser

Updates a user's information in Supabase and refreshes the cache.

```go
func (s *Service) UpdateUser(
    ctx context.Context,
    userID uuid.UUID,
    updates map[string]any,
) (*User, error)
```

**Parameters:**
- `ctx` - Context for request cancellation and timeout
- `userID` - Supabase user unique identifier (UUID)
- `updates` - Map of metadata fields to update (e.g., `{"display_name": "New Name", "role": "admin"}`)

**Returns:**
- `*User` - Updated user object with new values
- `error` - Error if update fails or user not found

**Behavior:**
- Retrieves user from cache to get access token
- Sends PUT request to `/auth/v1/user` with user's auth token
- Safely extracts updated metadata
- Updates cache with new values from response
- Returns updated user object

---

#### DeleteUser

Deletes a user from Supabase and removes from cache.

```go
func (s *Service) DeleteUser(
    ctx context.Context,
    userID uuid.UUID,
) error
```

**Parameters:**
- `ctx` - Context for request cancellation and timeout
- `userID` - Supabase user unique identifier (UUID)

**Returns:**
- `error` - Error if deletion fails

**Behavior:**
- Requires service role key for admin operations
- Sends DELETE request to `/auth/v1/admin/users/{id}` with service role key
- Removes user from cache by UUID
- Returns nil on success

**Note:** This is an admin operation requiring elevated privileges.

---

### Cache

The `UserCache` provides thread-safe in-memory storage for user sessions with intelligent eviction and automatic cleanup.

#### Cache Configuration

```go
type UserCache struct {
    MaxSize int // Maximum number of users (default: 1000)
    // ... internal fields
}
```

**Configurable Properties:**
- `MaxSize` - Maximum number of users allowed in cache (default: 1000)

**Example:**
```go
service := ft_supabase.NewService(projectID, projectURL, anonKey, serviceKey)
service.Cache.MaxSize = 500 // Limit to 500 users
```

---

#### NewUserCache

Creates a new UserCache instance with default settings.

```go
func NewUserCache() *UserCache
```

**Returns:** Initialized `*UserCache` with:
- Empty user maps
- MaxSize set to 1000
- Thread-safe mutex initialized

---

#### Set

Stores a user in the cache using their access token as the key.

```go
func (c *UserCache) Set(token string, user *CachedUser)
```

**Parameters:**
- `token` - JWT access token used as cache key
- `user` - Pointer to CachedUser to store

**Behavior:**
- Stores user indexed by both token and UserID
- If cache is full (>= MaxSize):
  1. First removes all expired entries
  2. If still full, evicts oldest user (by CachedAt timestamp)
- Updates existing user if token changes
- Thread-safe using write lock

**Eviction Strategy:**
- Expired entries are removed first
- Oldest cached entry evicted if needed (LRU-style)

---

#### Get

Retrieves a user from the cache by their access token.

```go
func (c *UserCache) Get(token string) (*CachedUser, bool)
```

**Parameters:**
- `token` - JWT access token used as cache key

**Returns:**
- `*CachedUser` - Cached user pointer
- `bool` - True if found and not expired, false otherwise

**Behavior:**
- Validates token expiration
- Thread-safe using read lock

---

#### GetByUserID

Retrieves a user from the cache by their UserID.

```go
func (c *UserCache) GetByUserID(userID uuid.UUID) (*CachedUser, bool)
```

**Parameters:**
- `userID` - Supabase user unique identifier (UUID)

**Returns:**
- `*CachedUser` - Cached user pointer
- `bool` - True if found and not expired, false otherwise

**Behavior:**
- Validates token expiration
- Thread-safe using read lock

---

#### Delete

Removes a user from the cache by their access token.

```go
func (c *UserCache) Delete(token string)
```

**Parameters:**
- `token` - JWT access token used as cache key

**Behavior:**
- Thread-safe using write lock

---

#### DeleteByUserID

Removes a user from the cache by their UserID.

```go
func (c *UserCache) DeleteByUserID(userID uuid.UUID)
```

**Parameters:**
- `userID` - Supabase user unique identifier (UUID)

**Behavior:**
- Removes from both token and userID indexes
- Thread-safe using write lock

---

#### IsValid

Checks if a token exists in cache and is not expired.

```go
func (c *UserCache) IsValid(token string) bool
```

**Parameters:**
- `token` - JWT access token to validate

**Returns:**
- `bool` - True if token exists and is valid, false otherwise

---

#### Cleanup

Removes all expired tokens from the cache.

```go
func (c *UserCache) Cleanup()
```

**Behavior:**
- Iterates through all cached users
- Removes expired sessions from both token and userID indexes
- Thread-safe using write lock
- Called automatically every 24 hours if `StartCacheCleanup()` is used

---

#### Count

Returns the number of users currently in the cache.

```go
func (c *UserCache) Count() int
```

**Returns:** Number of cached users

---

### Models

#### UserMetadata

Custom user metadata stored in Supabase `user_metadata` field.

```go
type UserMetadata struct {
    FullName    string `json:"full_name,omitempty"`
    DisplayName string `json:"display_name,omitempty"`
    AvatarURL   string `json:"avatar_url,omitempty"`
    Username    string `json:"username,omitempty"`
    Role        string `json:"role,omitempty"`
    DateOfBirth string `json:"date_of_birth,omitempty"` // Format: YYYY-MM-DD
}
```

---

#### User

Public user object returned from cache or API.

```go
type User struct {
    UserID      uuid.UUID `json:"user_id"`
    Email       string    `json:"email"`
    Username    string    `json:"username"`
    DisplayName string    `json:"display_name"`
    Role        string    `json:"role"`
    Phone       string    `json:"phone"`
    DateOfBirth string    `json:"date_of_birth,omitempty"`
}
```

---

#### CachedUser

Internal cached user session with authentication details.

```go
type CachedUser struct {
    UserID       uuid.UUID
    Email        string
    Username     string
    DisplayName  string
    Role         string
    Phone        string
    DateOfBirth  string
    AccessToken  string
    RefreshToken string
    ExpiresAt    time.Time
    CachedAt     time.Time
}
```

---

#### RegisterResponse

Response returned after successful user registration.

```go
type RegisterResponse struct {
    ID       string `json:"id"`
    UserName string `json:"username"`
    Email    string `json:"email"`
    Role     string `json:"role"`
}
```

---

#### LoginResponse

Response returned after successful user login.

```go
type LoginResponse struct {
    Token    string `json:"token"`
    ID       string `json:"id"`
    Email    string `json:"email"`
    Username string `json:"username"`
    Role     string `json:"role"`
}
```

---

#### RefreshTokenResponse

Response returned after successful token refresh.

```go
type RefreshTokenResponse struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresIn    int    `json:"expires_in"`
    ExpiresAt    int64  `json:"expires_at"`
    TokenType    string `json:"token_type"`
    ID           string `json:"id"`
    Email        string `json:"email"`
    Username     string `json:"username"`
    Role         string `json:"role"`
}
```

---

## Cache Management

### Automatic Cleanup

The cache includes automatic cleanup functionality to remove expired sessions:

```go
service := ft_supabase.NewService(projectID, projectURL, anonKey, serviceKey)

// Start automatic cleanup (runs every 24 hours)
service.StartCacheCleanup()
defer service.StopCacheCleanup()
```

**Cleanup Behavior:**
- Runs immediately on start
- Repeats every 24 hours
- Removes expired sessions from both indexes
- Thread-safe operation
- Prevents goroutine leaks when stopped

### Cache Size Limits

The cache enforces a maximum size (default: 1000 users) with intelligent eviction:

```go
// Use default max size (1000)
service := ft_supabase.NewService(projectID, projectURL, anonKey, serviceKey)

// Or configure custom max size
service.Cache.MaxSize = 500
```

**Eviction Strategy:**
1. When cache reaches max size during `Set()`:
   - First removes all expired entries
   - If still full, evicts oldest cached user (by `CachedAt` timestamp)
2. Maintains both token and userID index consistency
3. Updates existing users without counting toward limit

### Manual Cache Operations

```go
// Check cache size
count := service.Cache.Count()
fmt.Printf("Cached users: %d/%d\n", count, service.Cache.MaxSize)

// Manual cleanup (removes expired entries)
service.Cache.Cleanup()

// Validate token
if service.Cache.IsValid(token) {
    fmt.Println("Token is valid and not expired")
}

// Remove specific user
service.Cache.Delete(token)
service.Cache.DeleteByUserID(userID)
```

## Error Handling

### Sentinel Errors

The library defines sentinel errors for common error cases:

```go
var (
    ErrUnmarshalResponse = errors.New("failed to unmarshal response")
    ErrUserNotFound      = errors.New("user not found in cache")
    ErrInvalidToken      = errors.New("invalid or malformed JWT token")
    ErrTokenParseUserID  = errors.New("failed to parse user ID from token")
    ErrMissingMetadata   = errors.New("required metadata field is missing or invalid")
)
```

### HTTP Client Errors

```go
var (
    ErrMarshalRequest = errors.New("failed to marshal request")
    ErrCreateRequest  = errors.New("failed to create HTTP request")
    ErrSendRequest    = errors.New("failed to send request")
    ErrReadResponse   = errors.New("failed to read response body")
    ErrInvalidStatus  = errors.New("invalid response status")
)
```

### Safe Type Assertions

All metadata extraction uses safe type assertions that never panic:

```go
// Internal helper - returns empty string if field missing or wrong type
displayName, ok := getStringMetadata(metadata, "display_name")
if !ok {
    // Field not found or not a string - uses empty string
}
```

This ensures robust handling of:
- Missing metadata fields
- Null/nil values
- Incorrect types
- Empty values

## Thread Safety

All cache operations are thread-safe:

- Uses `sync.RWMutex` for concurrent access control
- Read operations use read locks for better performance
- Write operations use exclusive write locks
- Safe for concurrent use from multiple goroutines
- Background cleanup goroutine is thread-safe

## Examples

### Complete Service Setup

```go
package main

import (
    "context"
    "fmt"
    "log"
    ft_supabase "github.com/Cleroy288/ft_supabase"
)

func main() {
    // Create service
    service := ft_supabase.NewService(
        "your-project-id",
        "https://your-project.supabase.co",
        "your-anon-key",
        "your-service-role-key",
    )

    // Start automatic cache cleanup
    service.StartCacheCleanup()
    defer service.StopCacheCleanup()

    // Optional: Configure cache size
    service.Cache.MaxSize = 500

    // Use service...
}
```

### Register a New User

```go
metadata := ft_supabase.UserMetadata{
    Username:    "alice",
    DisplayName: "Alice Smith",
    Role:        "user",
    DateOfBirth: "1990-01-15",
}

response, err := service.RegisterUser(
    context.Background(),
    "alice@example.com",
    "securePassword123",
    "+1234567890",
    metadata,
)
if err != nil {
    log.Fatalf("Failed to register: %v", err)
}

fmt.Printf("Registered user: %s (ID: %s)\n", response.UserName, response.ID)
```

### Login a User

```go
loginResp, err := service.LoginUser(
    context.Background(),
    "alice@example.com",
    "securePassword123",
)
if err != nil {
    log.Fatalf("Login failed: %v", err)
}

fmt.Printf("Access token: %s\n", loginResp.Token)
fmt.Printf("User: %s (%s)\n", loginResp.Username, loginResp.Email)
```

### Logout a User

```go
err := service.Logout(
    context.Background(),
    accessToken,
)
if err != nil {
    log.Fatalf("Logout failed: %v", err)
}

fmt.Println("User logged out successfully")
```

### Refresh Access Token

```go
refreshResp, err := service.RefreshToken(
    context.Background(),
    refreshToken,
)
if err != nil {
    log.Fatalf("Token refresh failed: %v", err)
}

fmt.Printf("New access token: %s\n", refreshResp.AccessToken)
fmt.Printf("Token expires in: %d seconds\n", refreshResp.ExpiresIn)
```

### Get User by Token

```go
user, err := service.GetCurrentUser(
    context.Background(),
    accessToken,
)
if err != nil {
    if errors.Is(err, ft_supabase.ErrUserNotFound) {
        log.Println("User not found in cache - may need to login again")
    } else {
        log.Fatalf("Get user failed: %v", err)
    }
    return
}

fmt.Printf("Current user: %s (%s)\n", user.Username, user.Email)
```

### Get User by ID

```go
userID, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")

user, err := service.GetUserByID(
    context.Background(),
    userID,
)
if err != nil {
    log.Fatalf("Get user failed: %v", err)
}

fmt.Printf("User: %s (%s)\n", user.Username, user.Email)
```

### Update User Information

```go
userID, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")

updates := map[string]any{
    "display_name": "Alice Johnson",
    "role":         "admin",
}

updatedUser, err := service.UpdateUser(
    context.Background(),
    userID,
    updates,
)
if err != nil {
    log.Fatalf("Update failed: %v", err)
}

fmt.Printf("Updated user: %s (role: %s)\n", updatedUser.DisplayName, updatedUser.Role)
```

### Delete a User

```go
userID, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")

err := service.DeleteUser(
    context.Background(),
    userID,
)
if err != nil {
    log.Fatalf("Delete failed: %v", err)
}

fmt.Println("User deleted successfully")
```

### Cache Monitoring

```go
// Monitor cache size
count := service.Cache.Count()
maxSize := service.Cache.MaxSize
fmt.Printf("Cache usage: %d/%d users\n", count, maxSize)

// Check if token is valid
if service.Cache.IsValid(token) {
    fmt.Println("Token is valid and not expired")
} else {
    fmt.Println("Token is invalid or expired")
}

// Manual cleanup
service.Cache.Cleanup()
fmt.Println("Expired entries removed")
```

### Using Context with Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

response, err := service.RegisterUser(
    ctx,
    "user@example.com",
    "password",
    "",
    metadata,
)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Println("Request timed out")
    } else {
        log.Printf("Registration failed: %v\n", err)
    }
}
```

### Complete User Flow Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    ft_supabase "github.com/Cleroy288/ft_supabase"
)

func main() {
    // Setup
    service := ft_supabase.NewService(projectID, projectURL, anonKey, serviceKey)
    service.StartCacheCleanup()
    defer service.StopCacheCleanup()

    ctx := context.Background()

    // 1. Register user
    metadata := ft_supabase.UserMetadata{
        Username:    "testuser",
        DisplayName: "Test User",
        Role:        "user",
    }

    registerResp, err := service.RegisterUser(ctx, "test@example.com", "password123", "", metadata)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("✓ Registered: %s\n", registerResp.UserName)

    // 2. Login user
    loginResp, err := service.LoginUser(ctx, "test@example.com", "password123")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("✓ Logged in: %s\n", loginResp.Token[:30]+"...")

    // 3. Get current user
    user, err := service.GetCurrentUser(ctx, loginResp.Token)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("✓ Retrieved user: %s\n", user.Username)

    // 4. Update user
    updates := map[string]any{"display_name": "Updated Name"}
    updatedUser, err := service.UpdateUser(ctx, user.UserID, updates)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("✓ Updated display name: %s\n", updatedUser.DisplayName)

    // 5. Refresh token
    refreshResp, err := service.RefreshToken(ctx, loginResp.Token)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("✓ Refreshed token (expires in %d seconds)\n", refreshResp.ExpiresIn)

    // 6. Logout
    err = service.Logout(ctx, refreshResp.AccessToken)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("✓ Logged out")

    // 7. Cleanup (admin operation)
    err = service.DeleteUser(ctx, user.UserID)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("✓ User deleted")
}
```

## API Endpoints

The library uses the following Supabase Auth API endpoints:

- **POST** `/auth/v1/signup` - User registration
- **POST** `/auth/v1/token?grant_type=password` - User login
- **POST** `/auth/v1/logout` - User logout
- **POST** `/auth/v1/token?grant_type=refresh_token` - Token refresh
- **PUT** `/auth/v1/user` - Update user metadata
- **DELETE** `/auth/v1/admin/users/{id}` - Delete user (admin)

## Authentication Levels

The library supports three authentication levels:

1. **Anonymous Key** (`anonKey`) - Used for public operations like registration and login
2. **User Token** (Bearer token) - Used for user-specific operations like profile updates and logout
3. **Service Role Key** (`serviceKey`) - Used for admin operations like user deletion

## Best Practices

### Production Setup

```go
service := ft_supabase.NewService(projectID, projectURL, anonKey, serviceKey)

// Always start cache cleanup in production
service.StartCacheCleanup()
defer service.StopCacheCleanup()

// Configure cache size based on expected user load
service.Cache.MaxSize = 1000 // Adjust based on your needs
```

### Error Handling

```go
user, err := service.GetUserByID(ctx, userID)
if err != nil {
    switch {
    case errors.Is(err, ft_supabase.ErrUserNotFound):
        // User not in cache - might need to re-authenticate
        return handleReauth()
    case errors.Is(err, context.DeadlineExceeded):
        // Request timed out
        return handleTimeout()
    default:
        // Other error
        return handleError(err)
    }
}
```

### Context Usage

```go
// Use context with timeout for all operations
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

response, err := service.LoginUser(ctx, email, password)
```

### Graceful Shutdown

```go
// Setup signal handling
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

// Wait for interrupt
<-sigChan

// Cleanup
service.StopCacheCleanup()
fmt.Println("Service shut down gracefully")
```

## License

This is a custom Supabase implementation for Go applications.