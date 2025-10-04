package ft_supabase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Sentinel errors
var (
	ErrUnmarshalResponse = errors.New("failed to unmarshal response")
	ErrUserNotFound      = errors.New("user not found in cache")
	ErrInvalidToken      = errors.New("invalid or malformed JWT token")
	ErrTokenParseUserID  = errors.New("failed to parse user ID from token")
	ErrMissingMetadata   = errors.New("required metadata field is missing or invalid")
)

// Service represents a Supabase HTTP client.
// ProjectID is the Supabase project identifier.
// ProjectURL is the base URL for Supabase project API.
// AnonKey is the anonymous/public API key for client-side operations.
// ServiceKey is the service role key for server-side operations.
// HTTPClient is the HTTP client for making requests.
// Cache is the user session cache for storing authenticated users.
// cleanupDone is a channel to signal cleanup goroutine shutdown.
type Service struct {
	ProjectID   string
	ProjectURL  string
	AnonKey     string
	ServiceKey  string
	HTTPClient  HTTPClient
	Cache       *UserCache
	cleanupDone chan struct{}
}

// ServiceInterface defines the interface for Supabase authentication operations.
type ServiceInterface interface {
	// RegisterUser registers a new user with email, password, phone, and metadata.
	RegisterUser(ctx context.Context, email, password, phone string, metadata UserMetadata) (*RegisterResponse, error)

	// LoginUser authenticates a user and returns a JWT token.
	LoginUser(ctx context.Context, email, password string) (*LoginResponse, error)

	// GetUserByID retrieves a user by their ID from cache.
	GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error)

	// GetCurrentUser retrieves the current user by their JWT token from cache.
	GetCurrentUser(ctx context.Context, token string) (*User, error)

	// UpdateUser updates a user's information in Supabase and cache.
	UpdateUser(ctx context.Context, userID uuid.UUID, updates map[string]any) (*User, error)

	// DeleteUser deletes a user from Supabase and removes from cache.
	DeleteUser(ctx context.Context, userID uuid.UUID) error

	// Logout logs out a user by invalidating their session in Supabase and removing from cache.
	Logout(ctx context.Context, token string) error

	// RefreshToken refreshes an access token using a refresh token.
	RefreshToken(ctx context.Context, refreshToken string) (*RefreshTokenResponse, error)
}

// NewService creates a new Supabase service instance.
// projectID is the Supabase project identifier.
// projectURL is the base URL for the Supabase project API.
// anonKey is the anonymous/public API key.
// serviceKey is the service role key for privileged operations.
func NewService(projectID, projectURL, anonKey, serviceKey string) *Service {
	Logf("NewService", "Creating new Supabase service - ProjectID: %s, ProjectURL: %s", projectID, projectURL)

	// create service instance
	service := &Service{
		ProjectID:  projectID,
		ProjectURL: projectURL,
		AnonKey:    anonKey,
		ServiceKey: serviceKey,
		HTTPClient: NewFt_SupabaseHTTPClient(),
		Cache:      NewUserCache(),
	}

	Log("NewService", "Successfully created Supabase service instance")
	return service
}

// getStringMetadata safely extracts a string value from a metadata map.
// data is the metadata map to extract from.
// key is the metadata field key to extract.
// Returns the string value and true if found and valid, empty string and false otherwise.
func getStringMetadata(data map[string]any, key string) (string, bool) {
	var (
		val    any
		strVal string
		exists bool
		ok     bool
	)

	Logf("getStringMetadata", "Extracting metadata key: %s", key)

	// check if key exists
	val, exists = data[key]
	if !exists {
		Logf("getStringMetadata", "Key '%s' not found in metadata", key)
		return "", false
	}

	// check if value is nil
	if val == nil {
		Logf("getStringMetadata", "Key '%s' has nil value", key)
		return "", false
	}

	// type assert to string
	strVal, ok = val.(string)
	if !ok {
		Logf("getStringMetadata", "Key '%s' value is not a string", key)
		return "", false
	}

	Logf("getStringMetadata", "Successfully extracted metadata key '%s': %s", key, strVal)
	return strVal, true
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

	Logf("RegisterUser", "Starting user registration for email: %s", email)

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

	Log("RegisterUser", "Sending registration request to Supabase")

	// send request to Supabase
	bodyBytes, err = s.HTTPClient.Ft_SupabaseSendRequest(ctx, "POST", url, reqBody, s.getDefaultHeaders())
	if err != nil {
		Logf("RegisterUser", "Failed to send request: %v", err)
		return nil, err
	}

	Log("RegisterUser", "Parsing Supabase response")

	// parse JSON response
	if err = json.Unmarshal(bodyBytes, &supabaseResp); err != nil {
		Logf("RegisterUser", "Failed to unmarshal response: %v", err)
		return nil, fmt.Errorf("%w: %w", ErrUnmarshalResponse, err)
	}

	// extract custom metadata with safe type assertions
	usernameVal, _ = getStringMetadata(supabaseResp.User.UserMetadata, "username")
	roleVal, _ = getStringMetadata(supabaseResp.User.UserMetadata, "role")

	// parse user ID to UUID
	userUUID, err := uuid.Parse(supabaseResp.User.ID)
	if err != nil {
		Logf("RegisterUser", "Invalid user ID format: %v", err)
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	Log("RegisterUser", "Caching user session")

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

	Logf("RegisterUser", "Successfully registered user - ID: %s, Email: %s, Username: %s, Role: %s", supabaseResp.User.ID, supabaseResp.User.Email, usernameVal, roleVal)

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

	Logf("LoginUser", "Starting user login for email: %s", email)

	// build token endpoint URL
	url = fmt.Sprintf("%s%s", s.ProjectURL, LoginPath)

	// prepare login request body
	reqBody = SupabaseLoginRequest{
		Email:    email,
		Password: password,
	}

	Log("LoginUser", "Sending login request to Supabase")

	// send request to Supabase
	bodyBytes, err = s.HTTPClient.Ft_SupabaseSendRequest(ctx, "POST", url, reqBody, s.getDefaultHeaders())
	if err != nil {
		Logf("LoginUser", "Failed to send login request: %v", err)
		return nil, err
	}

	Log("LoginUser", "Parsing Supabase response")

	// parse JSON response
	if err = json.Unmarshal(bodyBytes, &supabaseResp); err != nil {
		Logf("LoginUser", "Failed to unmarshal response: %v", err)
		return nil, fmt.Errorf("%w: %w", ErrUnmarshalResponse, err)
	}

	// extract custom metadata with safe type assertions
	usernameVal, _ = getStringMetadata(supabaseResp.User.UserMetadata, "username")
	roleVal, _ = getStringMetadata(supabaseResp.User.UserMetadata, "role")
	displayNameVal, _ := getStringMetadata(supabaseResp.User.UserMetadata, "display_name")
	dobVal, _ := getStringMetadata(supabaseResp.User.UserMetadata, "date_of_birth")

	// parse user ID to UUID
	userUUID, err := uuid.Parse(supabaseResp.User.ID)
	if err != nil {
		Logf("LoginUser", "Invalid user ID format: %v", err)
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	Log("LoginUser", "Caching user session")

	// cache user session
	s.Cache.Set(supabaseResp.AccessToken, &CachedUser{
		UserID:       userUUID,
		Email:        supabaseResp.User.Email,
		Username:     usernameVal,
		DisplayName:  displayNameVal,
		Role:         roleVal,
		Phone:        supabaseResp.User.Phone,
		DateOfBirth:  dobVal,
		AccessToken:  supabaseResp.AccessToken,
		RefreshToken: supabaseResp.RefreshToken,
		ExpiresAt:    time.Unix(supabaseResp.ExpiresAt, 0),
		CachedAt:     time.Now(),
	})

	Logf("LoginUser", "Successfully logged in user - ID: %s, Email: %s, Username: %s, Role: %s", supabaseResp.User.ID, supabaseResp.User.Email, usernameVal, roleVal)

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
// userID is the Supabase user unique identifier (UUID).
// Returns a User object with user details or an error if not found in cache.
func (s *Service) GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error) {
	var (
		cachedUser *CachedUser
		found      bool
	)

	Logf("GetUserByID", "Retrieving user from cache - UserID: %s", userID.String())

	// lookup user in cache by ID
	cachedUser, found = s.Cache.GetByUserID(userID)
	if !found {
		Logf("GetUserByID", "User not found in cache - UserID: %s", userID.String())
		return nil, ErrUserNotFound
	}

	Logf("GetUserByID", "Successfully retrieved user - ID: %s, Email: %s, Username: %s", cachedUser.UserID.String(), cachedUser.Email, cachedUser.Username)

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

// GetCurrentUser retrieves the current user by their JWT token from the cache.
// ctx is the context for request cancellation and timeout.
// token is the JWT access token.
// Returns a User object with user details or an error if token is invalid or user not found in cache.
func (s *Service) GetCurrentUser(ctx context.Context, token string) (*User, error) {
	var (
		cachedUser *CachedUser
		found      bool
	)

	Log("GetCurrentUser", "Retrieving user from cache by token")

	// lookup user in cache by token
	cachedUser, found = s.Cache.Get(token)
	if !found {
		Log("GetCurrentUser", "User not found in cache or token expired")
		return nil, ErrUserNotFound
	}

	Logf("GetCurrentUser", "Successfully retrieved user - ID: %s, Email: %s, Username: %s", cachedUser.UserID.String(), cachedUser.Email, cachedUser.Username)

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
// userID is the Supabase user unique identifier (UUID).
// updates is a map of fields to update (e.g., {"display_name": "New Name", "role": "admin"}).
// Returns updated User object or an error if update fails.
func (s *Service) UpdateUser(ctx context.Context, userID uuid.UUID, updates map[string]any) (*User, error) {
	var (
		cachedUser *CachedUser
		found      bool
		url        string
		reqBody    UpdateUserRequest
		bodyBytes  []byte
		updateResp SupabaseUser
		err        error
	)

	Logf("UpdateUser", "Starting user update - UserID: %s, Updates: %v", userID.String(), updates)

	// lookup user in cache to get access token
	cachedUser, found = s.Cache.GetByUserID(userID)
	if !found {
		Logf("UpdateUser", "User not found in cache - UserID: %s", userID.String())
		return nil, ErrUserNotFound
	}

	// build update endpoint URL
	url = fmt.Sprintf("%s%s", s.ProjectURL, UpdateUserPath)

	// prepare request body with data field for metadata updates
	reqBody = UpdateUserRequest{
		Data: updates,
	}

	Log("UpdateUser", "Sending update request to Supabase")

	// send PUT request to Supabase with user's auth token
	bodyBytes, err = s.HTTPClient.Ft_SupabaseSendRequest(ctx, "PUT", url, reqBody, s.getAuthHeaders(cachedUser.AccessToken))
	if err != nil {
		Logf("UpdateUser", "Failed to send update request: %v", err)
		return nil, err
	}

	Log("UpdateUser", "Parsing Supabase response")

	// parse JSON response (update returns user object directly)
	if err = json.Unmarshal(bodyBytes, &updateResp); err != nil {
		Logf("UpdateUser", "Failed to unmarshal response: %v", err)
		return nil, fmt.Errorf("%w: %w", ErrUnmarshalResponse, err)
	}

	// extract updated metadata with safe type assertions
	usernameVal, _ := getStringMetadata(updateResp.UserMetadata, "username")
	roleVal, _ := getStringMetadata(updateResp.UserMetadata, "role")
	displayNameVal, _ := getStringMetadata(updateResp.UserMetadata, "display_name")
	dobVal, _ := getStringMetadata(updateResp.UserMetadata, "date_of_birth")

	Log("UpdateUser", "Updating cached user data")

	// update cache with new values
	cachedUser.Username = usernameVal
	cachedUser.Role = roleVal
	cachedUser.DisplayName = displayNameVal
	cachedUser.DateOfBirth = dobVal
	cachedUser.Email = updateResp.Email
	cachedUser.Phone = updateResp.Phone

	Logf("UpdateUser", "Successfully updated user - ID: %s, Email: %s, Username: %s", cachedUser.UserID.String(), cachedUser.Email, cachedUser.Username)

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
// userID is the Supabase user unique identifier (UUID).
// Returns an error if deletion fails.
// Note: Requires service role key for admin operations.
func (s *Service) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	var (
		url string
		err error
	)

	Logf("DeleteUser", "Starting user deletion - UserID: %s", userID.String())

	// build delete endpoint URL with user ID
	url = fmt.Sprintf("%s%s/%s", s.ProjectURL, DeleteUserPath, userID.String())

	Log("DeleteUser", "Sending delete request to Supabase")

	// send DELETE request to Supabase with service role key
	_, err = s.HTTPClient.Ft_SupabaseSendRequest(ctx, "DELETE", url, nil, s.getServiceHeaders())
	if err != nil {
		Logf("DeleteUser", "Failed to delete user from Supabase: %v", err)
		return err
	}

	Log("DeleteUser", "Removing user from cache")

	// delete user from cache
	s.Cache.DeleteByUserID(userID)

	Logf("DeleteUser", "Successfully deleted user - UserID: %s", userID.String())

	return nil
}

// Logout invalidates a user session in Supabase and removes from cache.
// ctx is the context for request cancellation and timeout.
// token is the JWT access token to invalidate.
// Returns an error if logout fails.
func (s *Service) Logout(ctx context.Context, token string) error {
	var (
		url string
		err error
	)

	Log("Logout", "Starting user logout")

	// build logout endpoint URL
	url = fmt.Sprintf("%s%s", s.ProjectURL, LogoutPath)

	Log("Logout", "Sending logout request to Supabase")

	// send POST request to Supabase with user's auth token
	_, err = s.HTTPClient.Ft_SupabaseSendRequest(ctx, "POST", url, nil, s.getAuthHeaders(token))
	if err != nil {
		Logf("Logout", "Failed to logout from Supabase: %v", err)
		return err
	}

	Log("Logout", "Removing user from cache")

	// remove user from cache
	s.Cache.Delete(token)

	Log("Logout", "Successfully logged out user")

	return nil
}

// RefreshToken refreshes an access token using a refresh token.
// ctx is the context for request cancellation and timeout.
// refreshToken is the refresh token obtained during login or registration.
// Returns a RefreshTokenResponse with new access token and user details or an error if refresh fails.
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*RefreshTokenResponse, error) {
	var (
		url          string
		reqBody      RefreshTokenRequest
		bodyBytes    []byte
		supabaseResp SupabaseAuthResponse
		usernameVal  string
		roleVal      string
		userUUID     uuid.UUID
		oldToken     string
		cachedUser   *CachedUser
		found        bool
		err          error
	)

	Log("RefreshToken", "Starting token refresh")

	// build refresh token endpoint URL
	url = fmt.Sprintf("%s%s", s.ProjectURL, RefreshTokenPath)

	// prepare refresh token request body
	reqBody = RefreshTokenRequest{
		RefreshToken: refreshToken,
	}

	Log("RefreshToken", "Sending refresh request to Supabase")

	// send request to Supabase
	bodyBytes, err = s.HTTPClient.Ft_SupabaseSendRequest(ctx, "POST", url, reqBody, s.getDefaultHeaders())
	if err != nil {
		Logf("RefreshToken", "Failed to send refresh request: %v", err)
		return nil, err
	}

	Log("RefreshToken", "Parsing Supabase response")

	// parse JSON response
	if err = json.Unmarshal(bodyBytes, &supabaseResp); err != nil {
		Logf("RefreshToken", "Failed to unmarshal response: %v", err)
		return nil, fmt.Errorf("%w: %w", ErrUnmarshalResponse, err)
	}

	// extract custom metadata with safe type assertions
	usernameVal, _ = getStringMetadata(supabaseResp.User.UserMetadata, "username")
	roleVal, _ = getStringMetadata(supabaseResp.User.UserMetadata, "role")
	displayNameVal, _ := getStringMetadata(supabaseResp.User.UserMetadata, "display_name")
	dobVal, _ := getStringMetadata(supabaseResp.User.UserMetadata, "date_of_birth")

	// parse user ID to UUID
	userUUID, err = uuid.Parse(supabaseResp.User.ID)
	if err != nil {
		Logf("RefreshToken", "Invalid user ID format: %v", err)
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	Log("RefreshToken", "Removing old token from cache")

	// find and remove old cache entry by user ID
	cachedUser, found = s.Cache.GetByUserID(userUUID)
	if found {
		oldToken = cachedUser.AccessToken
		s.Cache.Delete(oldToken)
		Log("RefreshToken", "Old token removed from cache")
	} else {
		Log("RefreshToken", "No old token found in cache")
	}

	Log("RefreshToken", "Caching new token")

	// cache user session with new tokens
	s.Cache.Set(supabaseResp.AccessToken, &CachedUser{
		UserID:       userUUID,
		Email:        supabaseResp.User.Email,
		Username:     usernameVal,
		DisplayName:  displayNameVal,
		Role:         roleVal,
		Phone:        supabaseResp.User.Phone,
		DateOfBirth:  dobVal,
		AccessToken:  supabaseResp.AccessToken,
		RefreshToken: supabaseResp.RefreshToken,
		ExpiresAt:    time.Unix(supabaseResp.ExpiresAt, 0),
		CachedAt:     time.Now(),
	})

	Logf("RefreshToken", "Successfully refreshed token for user: %s", supabaseResp.User.Email)

	// return formatted response
	return &RefreshTokenResponse{
		AccessToken:  supabaseResp.AccessToken,
		RefreshToken: supabaseResp.RefreshToken,
		ExpiresIn:    supabaseResp.ExpiresIn,
		ExpiresAt:    supabaseResp.ExpiresAt,
		TokenType:    supabaseResp.TokenType,
		ID:           supabaseResp.User.ID,
		Email:        supabaseResp.User.Email,
		Username:     usernameVal,
		Role:         roleVal,
	}, nil
}

// StartCacheCleanup starts a background goroutine that cleans expired cache entries every 24 hours.
// The cleanup runs immediately on start, then repeats every 24 hours.
// Call StopCacheCleanup() to stop the cleanup goroutine.
func (s *Service) StartCacheCleanup() {
	var (
		ticker *time.Ticker
	)

	Log("StartCacheCleanup", "Starting automatic cache cleanup (24-hour interval)")

	// initialize cleanup done channel
	s.cleanupDone = make(chan struct{})

	// run cleanup immediately on start
	go func() {
		Log("CacheCleanup", "Running initial cache cleanup")
		// initial cleanup
		s.Cache.Cleanup()

		// setup ticker for 24-hour interval
		ticker = time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				Log("CacheCleanup", "Running scheduled cache cleanup (24-hour interval)")
				// run cleanup every 24 hours
				s.Cache.Cleanup()
			case <-s.cleanupDone:
				Log("CacheCleanup", "Stopping cache cleanup goroutine")
				// stop cleanup goroutine
				return
			}
		}
	}()
}

// StopCacheCleanup stops the background cache cleanup goroutine.
// Should be called when shutting down the service to prevent goroutine leaks.
func (s *Service) StopCacheCleanup() {
	Log("StopCacheCleanup", "Stopping cache cleanup")

	if s.cleanupDone != nil {
		close(s.cleanupDone)
		s.cleanupDone = nil
		Log("StopCacheCleanup", "Cache cleanup stopped successfully")
	} else {
		Log("StopCacheCleanup", "Cache cleanup was not running")
	}
}
