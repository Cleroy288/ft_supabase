package ft_supabase

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// UserCache manages cached user sessions with thread-safe operations.
// users is a map where JWT tokens are keys and CachedUser pointers are values.
// usersByID is a map where UserIDs (UUID) are keys and CachedUser pointers are values.
// mu is a read-write mutex for thread-safe access to the cache.
// MaxSize is the maximum number of users allowed in cache (default 1000).
//
// Used in:
// - Service struct - holds the cache instance
// - NewService() - initializes new cache
// - RegisterUser() - stores user after registration
// - LoginUser() - stores user after login
// - GetUserByID() - retrieves user from cache
// - UpdateUser() - updates cached user data
// - DeleteUser() - removes user from cache
type UserCache struct {
	users     map[string]*CachedUser
	usersByID map[uuid.UUID]*CachedUser
	mu        sync.RWMutex
	MaxSize   int
}

// CachedUser represents a cached user session with authentication details.
// UserID is the Supabase user unique identifier (UUID).
// Email is the user's email address.
// Username is the user's unique username.
// DisplayName is the user's display name.
// Role is the user's application role.
// Phone is the user's phone number.
// DateOfBirth is the user's date of birth.
// AccessToken is the JWT authentication token.
// RefreshToken is the token used to refresh the access token.
// ExpiresAt is the timestamp when the access token expires.
// CachedAt is the timestamp when the user was cached.
//
// Used in:
// - Cache.Set() - stores user in cache
// - Cache.Get() - retrieves user from cache by token
// - Cache.GetByUserID() - retrieves user from cache by user ID
// - Cache.DeleteByUserID() - deletes user from cache
// - RegisterUser() - caches user after registration
// - LoginUser() - caches user after login
// - GetUserByID() - retrieves cached user data
// - UpdateUser() - updates cached user data
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

// User represents a user object returned from cache or API.
// UserID is the Supabase user unique identifier (UUID).
// Email is the user's email address.
// Username is the user's unique username.
// DisplayName is the user's display name.
// Role is the user's application role.
// Phone is the user's phone number.
// DateOfBirth is the user's date of birth.
//
// Used in:
// - GetUserByID() - returns User object from cache
// - UpdateUser() - returns updated User object
type User struct {
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	Phone       string    `json:"phone"`
	DateOfBirth string    `json:"date_of_birth,omitempty"`
}

// UserMetadata represents custom user metadata stored in Supabase.
// All fields are stored in user_metadata object in Supabase.
// Common fields like FullName, DisplayName, AvatarURL are optional.
// App-specific fields like Username, Role, DateOfBirth can be customized.
//
// Used in:
// - RegisterUser() - accepts metadata parameter for user registration
type UserMetadata struct {
	// Commonly used fields (optional)
	FullName    string `json:"full_name,omitempty"`
	DisplayName string `json:"display_name,omitempty"` // What shows to others (e.g., "John Smith")
	AvatarURL   string `json:"avatar_url,omitempty"`

	// Custom app fields
	Username    string `json:"username,omitempty"`      // Unique identifier (e.g., "@johnsmith")
	Role        string `json:"role,omitempty"`          // User role (e.g., "admin", "user")
	DateOfBirth string `json:"date_of_birth,omitempty"` // Format: YYYY-MM-DD
}

// SupabaseRegisterRequest represents a registration request payload.
// Email is the user's email address.
// Password is the user's password.
// Phone is the user's phone number (optional).
// Data contains custom user metadata (stored in user_metadata in Supabase).
//
// Used in:
// - RegisterUser() - builds request body for Supabase signup API
type SupabaseRegisterRequest struct {
	Email    string         `json:"email"`
	Password string         `json:"password"`
	Phone    string         `json:"phone,omitempty"`
	Data     map[string]any `json:"data"`
}

// SupabaseLoginRequest represents a login request payload.
// Email is the user's email address.
// Password is the user's password.
//
// Used in:
// - LoginUser() - builds request body for Supabase login API
type SupabaseLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SupabaseIdentity represents a user's identity provider information.
// Contains details about the authentication provider (email, OAuth, etc.).
//
// Used in:
// - SupabaseUser struct - part of user's identities array
type SupabaseIdentity struct {
	IdentityID   string         `json:"identity_id"`
	ID           string         `json:"id"`
	UserID       string         `json:"user_id"`
	IdentityData map[string]any `json:"identity_data"`
	Provider     string         `json:"provider"`
	LastSignInAt string         `json:"last_sign_in_at"`
	CreatedAt    string         `json:"created_at"`
	UpdatedAt    string         `json:"updated_at"`
	Email        string         `json:"email"`
}

// SupabaseUser represents the complete user object from Supabase Auth API.
// Contains all user fields returned by Supabase including metadata and identities.
//
// Used in:
// - SupabaseAuthResponse - nested in auth response
// - UpdateUser() - parses response from update endpoint
type SupabaseUser struct {
	ID               string             `json:"id"`
	Aud              string             `json:"aud"`
	Role             string             `json:"role"`
	Email            string             `json:"email"`
	EmailConfirmedAt string             `json:"email_confirmed_at"`
	Phone            string             `json:"phone"`
	ConfirmedAt      string             `json:"confirmed_at,omitempty"`
	LastSignInAt     string             `json:"last_sign_in_at"`
	AppMetadata      map[string]any     `json:"app_metadata"`
	UserMetadata     map[string]any     `json:"user_metadata"`
	Identities       []SupabaseIdentity `json:"identities"`
	CreatedAt        string             `json:"created_at"`
	UpdatedAt        string             `json:"updated_at"`
	IsAnonymous      bool               `json:"is_anonymous"`
}

// SupabaseAuthResponse represents the authentication response from Supabase.
// Contains access token, refresh token, expiration info, and user object.
//
// Used in:
// - RegisterUser() - parses response from signup endpoint
// - LoginUser() - parses response from login endpoint
type SupabaseAuthResponse struct {
	AccessToken  string       `json:"access_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int          `json:"expires_in"`
	ExpiresAt    int64        `json:"expires_at"`
	RefreshToken string       `json:"refresh_token"`
	User         SupabaseUser `json:"user"`
	WeakPassword *string      `json:"weak_password,omitempty"`
}

// RegisterResponse represents the response returned after user registration.
// Contains basic user information returned to the client.
//
// Used in:
// - RegisterUser() - returns this response to caller
type RegisterResponse struct {
	ID       string `json:"id"`
	UserName string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// LoginResponse represents the response returned after user login.
// Contains JWT token and basic user information.
//
// Used in:
// - LoginUser() - returns this response to caller
type LoginResponse struct {
	Token    string `json:"token"`
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// UpdateUserRequest represents the request payload for updating user metadata.
// Data contains the metadata fields to update (e.g., {"display_name": "New Name", "role": "admin"}).
//
// Used in:
// - UpdateUser() - builds request body for Supabase update endpoint
type UpdateUserRequest struct {
	Data map[string]any `json:"data"`
}

// RefreshTokenRequest represents the request payload for refreshing an access token.
// RefreshToken is the refresh token obtained during login or registration.
//
// Used in:
// - RefreshToken() - builds request body for Supabase token refresh endpoint
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshTokenResponse represents the response returned after token refresh.
// Contains new access token, refresh token, and basic user information.
//
// Used in:
// - RefreshToken() - returns this response to caller
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
