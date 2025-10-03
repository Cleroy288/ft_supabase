package ft_supabase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"ft_supabase/internal/ft_supabase_utils"
	"time"

	"github.com/google/uuid"
)

// Sentinel errors
var (
	ErrUnmarshalResponse = errors.New("failed to unmarshal response")
	ErrUserNotFound      = errors.New("user not found in cache")
)

// Service represents a Supabase HTTP client.
// ProjectID is the Supabase project identifier.
// ProjectURL is the base URL for Supabase project API.
// AnonKey is the anonymous/public API key for client-side operations.
// ServiceKey is the service role key for server-side operations.
// HTTPClient is the HTTP client for making requests.
// Cache is the user session cache for storing authenticated users.
type Service struct {
	ProjectID  string
	ProjectURL string
	AnonKey    string
	ServiceKey string
	HTTPClient ft_supabase_utils.HTTPClient
	Cache      *UserCache
}

// ServiceInterface defines the interface for Supabase authentication operations.
type ServiceInterface interface {
	// RegisterUser registers a new user with email, password, phone, and metadata.
	RegisterUser(ctx context.Context, email, password, phone string, metadata UserMetadata) (*RegisterResponse, error)

	// LoginUser authenticates a user and returns a JWT token.
	LoginUser(ctx context.Context, email, password string) (*LoginResponse, error)

	// GetUserByID retrieves a user by their ID from cache.
	GetUserByID(ctx context.Context, userID string) (*User, error)

	// UpdateUser updates a user's information in Supabase and cache.
	UpdateUser(ctx context.Context, userID string, updates map[string]any) (*User, error)

	// DeleteUser deletes a user from Supabase and removes from cache.
	DeleteUser(ctx context.Context, userID string) error
}

// NewService creates a new Supabase service instance.
// projectID is the Supabase project identifier.
// projectURL is the base URL for the Supabase project API.
// anonKey is the anonymous/public API key.
// serviceKey is the service role key for privileged operations.
func NewService(projectID, projectURL, anonKey, serviceKey string) *Service {
	// create service instance
	return &Service{
		ProjectID:  projectID,
		ProjectURL: projectURL,
		AnonKey:    anonKey,
		ServiceKey: serviceKey,
		HTTPClient: ft_supabase_utils.NewFt_SupabaseHTTPClient(),
		Cache:      NewUserCache(),
	}
}

// RegisterUser registers a new user with Supabase Auth API.
// ctx is the context for request cancellation and timeout.
// email is the user's email address.
// password is the user's password.
// phone is the user's phone number (optional, can be empty string).
// metadata contains user metadata (all stored in user_metadata in Supabase).
// Returns a RegisterResponse with user details or an error if registration fails.
func (s *Service) RegisterUser(ctx context.Context, email, password, phone string, metadata UserMetadata) (*RegisterResponse, error) {
	var (
		url          string
		reqBody      SupabaseRegisterRequest
		bodyBytes    []byte
		supabaseResp SupabaseAuthResponse
		metadataMap  map[string]any
		usernameVal  string
		roleVal      string
		err          error
	)

	// build signup endpoint URL
	url = fmt.Sprintf("%s%s", s.ProjectURL, SignupPath)

	// build metadata map from struct (all fields go into user_metadata)
	metadataMap = make(map[string]any)

	// add common metadata fields if provided
	if metadata.FullName != "" {
		metadataMap["full_name"] = metadata.FullName
	}
	if metadata.DisplayName != "" {
		metadataMap["display_name"] = metadata.DisplayName
	}
	if metadata.AvatarURL != "" {
		metadataMap["avatar_url"] = metadata.AvatarURL
	}

	// add custom metadata fields if provided
	if metadata.Username != "" {
		metadataMap["username"] = metadata.Username
	}
	if metadata.Role != "" {
		metadataMap["role"] = metadata.Role
	}
	if metadata.DateOfBirth != "" {
		metadataMap["date_of_birth"] = metadata.DateOfBirth
	}

	// prepare request body with user metadata
	reqBody = SupabaseRegisterRequest{
		Email:    email,
		Password: password,
		Phone:    phone,
		Data:     metadataMap,
	}

	// send request to Supabase
	bodyBytes, err = s.HTTPClient.Ft_SupabaseSendRequest(ctx, "POST", url, reqBody, s.getDefaultHeaders())
	if err != nil {
		return nil, err
	}

	// parse JSON response
	if err = json.Unmarshal(bodyBytes, &supabaseResp); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnmarshalResponse, err)
	}

	// extract custom metadata
	usernameVal, _ = supabaseResp.User.UserMetadata["username"].(string)
	roleVal, _ = supabaseResp.User.UserMetadata["role"].(string)

	// parse user ID to UUID
	userUUID, err := uuid.Parse(supabaseResp.User.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	// cache user session
	s.Cache.Set(supabaseResp.AccessToken, &CachedUser{
		UserID:       userUUID,
		Email:        supabaseResp.User.Email,
		Username:     usernameVal,
		DisplayName:  metadata.DisplayName,
		Role:         roleVal,
		Phone:        supabaseResp.User.Phone,
		DateOfBirth:  metadata.DateOfBirth,
		AccessToken:  supabaseResp.AccessToken,
		RefreshToken: supabaseResp.RefreshToken,
		ExpiresAt:    time.Unix(supabaseResp.ExpiresAt, 0),
		CachedAt:     time.Now(),
	})

	// return formatted response
	return &RegisterResponse{
		ID:       supabaseResp.User.ID,
		UserName: usernameVal,
		Email:    supabaseResp.User.Email,
		Role:     roleVal,
	}, nil
}

// LoginUser authenticates a user and returns a JWT access token.
// ctx is the context for request cancellation and timeout.
// email is the user's email address.
// password is the user's password.
// Returns a LoginResponse with JWT token and user details or an error if authentication fails.
func (s *Service) LoginUser(ctx context.Context, email, password string) (*LoginResponse, error) {
	var (
		url          string
		reqBody      SupabaseLoginRequest
		bodyBytes    []byte
		supabaseResp SupabaseAuthResponse
		usernameVal  string
		roleVal      string
		err          error
	)

	// build token endpoint URL
	url = fmt.Sprintf("%s%s", s.ProjectURL, LoginPath)

	// prepare login request body
	reqBody = SupabaseLoginRequest{
		Email:    email,
		Password: password,
	}

	// send request to Supabase
	bodyBytes, err = s.HTTPClient.Ft_SupabaseSendRequest(ctx, "POST", url, reqBody, s.getDefaultHeaders())
	if err != nil {
		return nil, err
	}

	// parse JSON response
	if err = json.Unmarshal(bodyBytes, &supabaseResp); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnmarshalResponse, err)
	}

	// extract custom metadata
	usernameVal, _ = supabaseResp.User.UserMetadata["username"].(string)
	roleVal, _ = supabaseResp.User.UserMetadata["role"].(string)

	// parse user ID to UUID
	userUUID, err := uuid.Parse(supabaseResp.User.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	// cache user session
	s.Cache.Set(supabaseResp.AccessToken, &CachedUser{
		UserID:       userUUID,
		Email:        supabaseResp.User.Email,
		Username:     usernameVal,
		DisplayName:  supabaseResp.User.UserMetadata["display_name"].(string),
		Role:         roleVal,
		Phone:        supabaseResp.User.Phone,
		DateOfBirth:  supabaseResp.User.UserMetadata["date_of_birth"].(string),
		AccessToken:  supabaseResp.AccessToken,
		RefreshToken: supabaseResp.RefreshToken,
		ExpiresAt:    time.Unix(supabaseResp.ExpiresAt, 0),
		CachedAt:     time.Now(),
	})

	// return formatted response
	return &LoginResponse{
		Token:    supabaseResp.AccessToken,
		ID:       supabaseResp.User.ID,
		Email:    supabaseResp.User.Email,
		Username: usernameVal,
		Role:     roleVal,
	}, nil
}

// GetUserByID retrieves a user by their ID from the cache.
// ctx is the context for request cancellation and timeout.
// userID is the Supabase user unique identifier (string format).
// Returns a User object with user details or an error if not found in cache.
func (s *Service) GetUserByID(ctx context.Context, userID string) (*User, error) {
	var (
		cachedUser *CachedUser
		found      bool
		parsedUUID uuid.UUID
		err        error
	)

	// parse string to UUID
	parsedUUID, err = uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID format: %w", err)
	}

	// lookup user in cache by ID
	cachedUser, found = s.Cache.GetByUserID(parsedUUID)
	if !found {
		return nil, ErrUserNotFound
	}

	// return user object from cache
	return &User{
		UserID:      cachedUser.UserID,
		Email:       cachedUser.Email,
		Username:    cachedUser.Username,
		DisplayName: cachedUser.DisplayName,
		Role:        cachedUser.Role,
		Phone:       cachedUser.Phone,
		DateOfBirth: cachedUser.DateOfBirth,
	}, nil
}

// UpdateUser updates a user's information in Supabase and refreshes the cache.
// ctx is the context for request cancellation and timeout.
// userID is the Supabase user unique identifier (string format).
// updates is a map of fields to update (e.g., {"display_name": "New Name", "role": "admin"}).
// Returns updated User object or an error if update fails.
func (s *Service) UpdateUser(ctx context.Context, userID string, updates map[string]any) (*User, error) {
	var (
		cachedUser *CachedUser
		found      bool
		parsedUUID uuid.UUID
		url        string
		reqBody    UpdateUserRequest
		bodyBytes  []byte
		updateResp SupabaseUser
		err        error
	)

	// parse string to UUID
	parsedUUID, err = uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID format: %w", err)
	}

	// lookup user in cache to get access token
	cachedUser, found = s.Cache.GetByUserID(parsedUUID)
	if !found {
		return nil, ErrUserNotFound
	}

	// build update endpoint URL
	url = fmt.Sprintf("%s%s", s.ProjectURL, UpdateUserPath)

	// prepare request body with data field for metadata updates
	reqBody = UpdateUserRequest{
		Data: updates,
	}

	// send PUT request to Supabase with user's auth token
	bodyBytes, err = s.HTTPClient.Ft_SupabaseSendRequest(ctx, "PUT", url, reqBody, s.getAuthHeaders(cachedUser.AccessToken))
	if err != nil {
		return nil, err
	}

	// parse JSON response (update returns user object directly)
	if err = json.Unmarshal(bodyBytes, &updateResp); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnmarshalResponse, err)
	}

	// extract updated metadata
	usernameVal, _ := updateResp.UserMetadata["username"].(string)
	roleVal, _ := updateResp.UserMetadata["role"].(string)
	displayNameVal, _ := updateResp.UserMetadata["display_name"].(string)
	dobVal, _ := updateResp.UserMetadata["date_of_birth"].(string)

	// update cache with new values
	cachedUser.Username = usernameVal
	cachedUser.Role = roleVal
	cachedUser.DisplayName = displayNameVal
	cachedUser.DateOfBirth = dobVal
	cachedUser.Email = updateResp.Email
	cachedUser.Phone = updateResp.Phone

	// return updated user object
	return &User{
		UserID:      cachedUser.UserID,
		Email:       cachedUser.Email,
		Username:    cachedUser.Username,
		DisplayName: cachedUser.DisplayName,
		Role:        cachedUser.Role,
		Phone:       cachedUser.Phone,
		DateOfBirth: cachedUser.DateOfBirth,
	}, nil
}

// DeleteUser deletes a user from Supabase and removes from cache.
// ctx is the context for request cancellation and timeout.
// userID is the Supabase user unique identifier (string format).
// Returns an error if deletion fails.
// Note: Requires service role key for admin operations.
func (s *Service) DeleteUser(ctx context.Context, userID string) error {
	var (
		parsedUUID uuid.UUID
		url        string
		err        error
	)

	// parse string to UUID
	parsedUUID, err = uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid UUID format: %w", err)
	}

	// build delete endpoint URL with user ID
	url = fmt.Sprintf("%s%s/%s", s.ProjectURL, DeleteUserPath, userID)

	// send DELETE request to Supabase with service role key
	_, err = s.HTTPClient.Ft_SupabaseSendRequest(ctx, "DELETE", url, nil, s.getServiceHeaders())
	if err != nil {
		return err
	}

	// delete user from cache
	s.Cache.DeleteByUserID(parsedUUID)

	return nil
}
