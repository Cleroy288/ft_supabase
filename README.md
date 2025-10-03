# ft_supabase

A Go client library for Supabase Authentication with built-in caching and session management.

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
- [Error Handling](#error-handling)
- [Thread Safety](#thread-safety)
- [Examples](#examples)

## Overview

`ft_supabase` is a comprehensive Go package that provides authentication and user management functionality for Supabase applications. It includes a thread-safe caching system for user sessions, support for custom user metadata, and a clean interface for all common authentication operations.

## Features

- **User Registration** - Create new users with email, password, and custom metadata
- **User Authentication** - Login users and receive JWT access tokens
- **User Management** - Retrieve, update, and delete users
- **Session Caching** - Thread-safe in-memory cache for user sessions
- **Custom Metadata** - Support for custom user fields (username, role, display name, etc.)
- **Multiple Authentication Levels** - Anon key for public operations, service role key for admin operations
- **Context Support** - All operations support context for cancellation and timeout
- **Type Safety** - Strongly typed models with UUID support

## Installation

```bash
go get ft_supabase
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
    "ft_supabase/internal/ft_supabase"
)

func main() {
    // Initialize the service
    service := ft_supabase.NewService(
        "project-id",
        "https://project.supabase.co",
        "anon-key",
        "service-role-key",
    )

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

The library is organized into the following packages:

### ft_supabase (Core Package)

- **service.go** - Main service implementation with authentication functions
- **models.go** - Data structures and type definitions
- **cached.go** - Thread-safe cache implementation for user sessions
- **headers.go** - HTTP header constants and helper functions
- **endpoints.go** - Supabase API endpoint constants

### ft_supabase_utils (Utilities Package)

- **service.go** - HTTP client utilities for making API requests

## API Reference

### Service

The `Service` struct is the main entry point for all authentication operations.

#### NewService

Creates a new Supabase service instance.

```go
func NewService(projectID, projectURL, anonKey, serviceKey string) *Service
```

**Parameters:**
- `projectID` - Supabase project identifier
- `projectURL` - Base URL for the Supabase project API (e.g., `https://project.supabase.co`)
- `anonKey` - Anonymous/public API key for client-side operations
- `serviceKey` - Service role key for privileged server-side operations

**Returns:** Initialized `*Service` with HTTP client and cache

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
- Extracts custom metadata from response

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

The `UserCache` provides thread-safe in-memory storage for user sessions.

#### NewUserCache

Creates a new UserCache instance.

```go
func NewUserCache() *UserCache
```

**Returns:** Initialized `*UserCache` with empty user maps

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
- Thread-safe using write lock

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
- Removes expired sessions
- Thread-safe using write lock

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

## Error Handling

The library defines sentinel errors for common error cases:

### Service Errors

```go
var (
    ErrUnmarshalResponse = errors.New("failed to unmarshal response")
    ErrUserNotFound      = errors.New("user not found in cache")
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

## Thread Safety

All cache operations are thread-safe:

- Uses `sync.RWMutex` for concurrent access control
- Read operations use read locks for better performance
- Write operations use exclusive write locks
- Safe for concurrent use from multiple goroutines

## Examples

### Register a New User

```go
service := ft_supabase.NewService(projectID, projectURL, anonKey, serviceKey)

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

fmt.Printf("Registered user: %s\n", response.UserName)
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

fmt.Printf("Updated user: %s\n", updatedUser.DisplayName)
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

### Cache Management

```go
// Check if token is valid
if service.Cache.IsValid(token) {
    fmt.Println("Token is valid")
}

// Get cache count
count := service.Cache.Count()
fmt.Printf("Cached users: %d\n", count)

// Cleanup expired tokens
service.Cache.Cleanup()
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
    }
}
```

## API Endpoints

The library uses the following Supabase Auth API endpoints:

- **POST** `/auth/v1/signup` - User registration
- **POST** `/auth/v1/token?grant_type=password` - User login
- **PUT** `/auth/v1/user` - Update user metadata
- **DELETE** `/auth/v1/admin/users/{id}` - Delete user (admin)

## Authentication Levels

The library supports three authentication levels:

1. **Anonymous Key** (`anonKey`) - Used for public operations like registration and login
2. **User Token** (Bearer token) - Used for user-specific operations like profile updates
3. **Service Role Key** (`serviceKey`) - Used for admin operations like user deletion

## License

This is a custom Supabase implementation for Go applications.
