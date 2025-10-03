package ft_supabase

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
)

const (
	testProjectID  = ""
	testProjectURL = "https//" + testProjectID + ".supabase.co"
	testAnonKey    = ""
	testServiceKey = ""
)

// ANSI color codes for terminal output
const (
	colorReset = "\033[0m"
	colorRed   = "\033[31m"
	colorGreen = "\033[32m"
)

// TestResult tracks the outcome of a single test
type TestResult struct {
	Name   string
	Passed bool
	Output string
	Error  string
}

// Global test results tracker
var (
	testResults   []TestResult
	testResultsMu sync.Mutex
)

// recordTestResult records the result of a test
func recordTestResult(name string, passed bool, output string, err string) {
	testResultsMu.Lock()
	defer testResultsMu.Unlock()

	testResults = append(testResults, TestResult{
		Name:   name,
		Passed: passed,
		Output: output,
		Error:  err,
	})
}

// printTestSummary prints a colored summary of all test results
func printTestSummary() {
	var (
		totalTests  int
		passedTests int
		failedTests int
	)

	testResultsMu.Lock()
	defer testResultsMu.Unlock()

	totalTests = len(testResults)
	for _, result := range testResults {
		if result.Passed {
			passedTests++
		} else {
			failedTests++
		}
	}

	fmt.Println("\n\n" + string(bytes.Repeat([]byte("="), 60)))
	fmt.Println("                     TEST SUMMARY")
	fmt.Println(string(bytes.Repeat([]byte("="), 60)))

	// print individual test results
	for _, result := range testResults {
		if result.Passed {
			fmt.Printf("%s[SUCCESS]%s %s\n", colorGreen, colorReset, result.Name)
		} else {
			fmt.Printf("%s[FAIL]%s %s\n", colorRed, colorReset, result.Name)
			if result.Error != "" {
				fmt.Printf("  Error: %s\n", result.Error)
			}
			if result.Output != "" {
				fmt.Printf("  Output:\n%s\n", result.Output)
			}
		}
	}

	// print summary statistics
	fmt.Println(string(bytes.Repeat([]byte("="), 60)))
	fmt.Printf("Total:  %d tests\n", totalTests)
	fmt.Printf("%sPassed: %d tests%s\n", colorGreen, passedTests, colorReset)
	if failedTests > 0 {
		fmt.Printf("%sFailed: %d tests%s\n", colorRed, failedTests, colorReset)
	} else {
		fmt.Printf("Failed: %d tests\n", failedTests)
	}
	fmt.Println(string(bytes.Repeat([]byte("="), 60)))
}

// TestMain runs before all tests and prints summary after all tests complete
func TestMain(m *testing.M) {
	// run all tests
	exitCode := m.Run()

	// print summary
	printTestSummary()

	// exit with appropriate code
	os.Exit(exitCode)
}

// setupTestUser is a helper function to register and login a user for testing.
// Returns the service instance, login response, and any error encountered.
func setupTestUser(email, password string, metadata UserMetadata) (*Service, *LoginResponse, error) {
	var (
		service   *Service
		loginResp *LoginResponse
		err       error
	)

	// create service instance
	service = NewService(testProjectID, testProjectURL, testAnonKey, testServiceKey)
	ctx := context.Background()

	// register user (may fail if already exists)
	_, err = service.RegisterUser(ctx, email, password, "", metadata)
	if err != nil {
		// user might already exist, continue with login
		fmt.Printf("⚠ Registration failed (user may already exist): %v\n", err)
	}

	// login user
	loginResp, err = service.LoginUser(ctx, email, password)
	if err != nil {
		return nil, nil, fmt.Errorf("login failed: %w", err)
	}

	return service, loginResp, nil
}

// TestRegisterUser tests user registration functionality.
func TestRegisterUser(t *testing.T) {
	var (
		testName     = "TestRegisterUser"
		service      *Service
		ctx          context.Context
		email        string
		password     string
		phone        string
		metadata     UserMetadata
		resp         *RegisterResponse
		userID       uuid.UUID
		output       bytes.Buffer
		errorMessage string
		err          error
	)

	// setup
	service = NewService(testProjectID, testProjectURL, testAnonKey, testServiceKey)
	ctx = context.Background()

	// generate random user data to avoid conflicts
	randomID := uuid.New().String()[:8]
	email = fmt.Sprintf("testuser_%s@example.com", randomID)
	password = "SecurePassword123!"
	phone = ""
	metadata = UserMetadata{
		DisplayName: fmt.Sprintf("Test User %s", randomID),
		Username:    fmt.Sprintf("testuser_%s", randomID),
		Role:        "user",
		DateOfBirth: "1990-01-01",
	}

	output.WriteString("\n========================================\n")
	output.WriteString("Testing RegisterUser\n")
	output.WriteString("========================================\n")

	// execute
	resp, err = service.RegisterUser(ctx, email, password, phone, metadata)
	if err != nil {
		errorMessage = fmt.Sprintf("RegisterUser failed: %v", err)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Fatalf("%s", errorMessage)
		return
	}

	// verify
	output.WriteString("\n✓ Registration successful!\n")
	output.WriteString(fmt.Sprintf("  ID: %s\n", resp.ID))
	output.WriteString(fmt.Sprintf("  Email: %s\n", resp.Email))
	output.WriteString(fmt.Sprintf("  Username: %s\n", resp.UserName))
	output.WriteString(fmt.Sprintf("  Role: %s\n", resp.Role))

	if resp.Email != email {
		errorMessage = fmt.Sprintf("Expected email %s, got %s", email, resp.Email)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Errorf("%s", errorMessage)
		return
	}
	if resp.UserName != metadata.Username {
		errorMessage = fmt.Sprintf("Expected username %s, got %s", metadata.Username, resp.UserName)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Errorf("%s", errorMessage)
		return
	}
	if resp.Role != metadata.Role {
		errorMessage = fmt.Sprintf("Expected role %s, got %s", metadata.Role, resp.Role)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Errorf("%s", errorMessage)
		return
	}

	// cleanup: delete the test user
	userID, err = uuid.Parse(resp.ID)
	if err != nil {
		errorMessage = fmt.Sprintf("Failed to parse user ID for cleanup: %v", err)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Errorf("%s", errorMessage)
		return
	}

	err = service.DeleteUser(ctx, userID)
	if err != nil {
		output.WriteString(fmt.Sprintf("\n⚠ Warning: Failed to cleanup test user: %v\n", err))
	} else {
		output.WriteString("\n✓ Test user cleaned up\n")
	}
	output.WriteString("========================================\n")

	recordTestResult(testName, true, output.String(), "")
}

// TestLoginUser tests user authentication functionality.
func TestLoginUser(t *testing.T) {
	var (
		testName     = "TestLoginUser"
		service      *Service
		ctx          context.Context
		email        string
		password     string
		metadata     UserMetadata
		loginResp    *LoginResponse
		output       bytes.Buffer
		errorMessage string
		err          error
	)

	// setup
	service = NewService(testProjectID, testProjectURL, testAnonKey, testServiceKey)
	ctx = context.Background()

	email = "logintest@example22.com"
	password = "SecurePassword123!"
	metadata = UserMetadata{
		DisplayName: "Login User2",
		Username:    "loginuser2",
		Role:        "user",
		DateOfBirth: "1995-05-15",
	}

	output.WriteString("\n========================================\n")
	output.WriteString("Testing LoginUser\n")
	output.WriteString("========================================\n")

	// register user first (may already exist)
	_, err = service.RegisterUser(ctx, email, password, "", metadata)
	if err != nil {
		output.WriteString(fmt.Sprintf("⚠ Registration failed (user may already exist): %v\n", err))
	}

	// execute
	loginResp, err = service.LoginUser(ctx, email, password)
	if err != nil {
		errorMessage = fmt.Sprintf("LoginUser failed: %v", err)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Fatalf("%s", errorMessage)
		return
	}

	// verify
	output.WriteString("\n✓ Login successful!\n")
	output.WriteString(fmt.Sprintf("  Token: %s...\n", loginResp.Token[:50]))
	output.WriteString(fmt.Sprintf("  ID: %s\n", loginResp.ID))
	output.WriteString(fmt.Sprintf("  Email: %s\n", loginResp.Email))
	output.WriteString(fmt.Sprintf("  Username: %s\n", loginResp.Username))
	output.WriteString(fmt.Sprintf("  Role: %s\n", loginResp.Role))
	output.WriteString("========================================\n")

	if loginResp.Email != email {
		errorMessage = fmt.Sprintf("Expected email %s, got %s", email, loginResp.Email)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Errorf("%s", errorMessage)
		return
	}
	if loginResp.Token == "" {
		errorMessage = "Expected non-empty token"
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Error(errorMessage)
		return
	}

	// verify user is cached
	cachedUser, found := service.Cache.Get(loginResp.Token)
	if !found {
		errorMessage = "User should be in cache after login"
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Error(errorMessage)
		return
	} else {
		output.WriteString("\n✓ User found in cache\n")
		output.WriteString(fmt.Sprintf("  Cached UserID: %s\n", cachedUser.UserID))
		output.WriteString(fmt.Sprintf("  Cached Email: %s\n", cachedUser.Email))
		output.WriteString(fmt.Sprintf("  Cached Username: %s\n", cachedUser.Username))
		output.WriteString(fmt.Sprintf("  Token valid: %v\n", service.Cache.IsValid(loginResp.Token)))
	}

	recordTestResult(testName, true, output.String(), "")
}

// TestGetUserByID tests retrieving a user by their ID from cache.
func TestGetUserByID(t *testing.T) {
	var (
		testName     = "TestGetUserByID"
		service      *Service
		loginResp    *LoginResponse
		ctx          context.Context
		userID       uuid.UUID
		user         *User
		output       bytes.Buffer
		errorMessage string
		err          error
	)

	// setup
	service, loginResp, err = setupTestUser(
		"getuserbyid@example22.com",
		"SecurePassword123!",
		UserMetadata{
			DisplayName: "GetUser Test",
			Username:    "getusertest",
			Role:        "user",
			DateOfBirth: "1992-03-10",
		},
	)
	if err != nil {
		errorMessage = fmt.Sprintf("Setup failed: %v", err)
		recordTestResult(testName, false, "", errorMessage)
		t.Fatalf("%s", errorMessage)
		return
	}

	ctx = context.Background()

	output.WriteString("\n========================================\n")
	output.WriteString("Testing GetUserByID\n")
	output.WriteString("========================================\n")

	// parse user ID
	userID, err = uuid.Parse(loginResp.ID)
	if err != nil {
		errorMessage = fmt.Sprintf("Failed to parse user ID: %v", err)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Fatalf("%s", errorMessage)
		return
	}

	// execute
	user, err = service.GetUserByID(ctx, userID)
	if err != nil {
		errorMessage = fmt.Sprintf("GetUserByID failed: %v", err)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Fatalf("%s", errorMessage)
		return
	}

	// verify
	output.WriteString("✓ User retrieved by ID\n")
	output.WriteString(fmt.Sprintf("  UserID: %s\n", user.UserID))
	output.WriteString(fmt.Sprintf("  Email: %s\n", user.Email))
	output.WriteString(fmt.Sprintf("  Username: %s\n", user.Username))
	output.WriteString(fmt.Sprintf("  DisplayName: %s\n", user.DisplayName))
	output.WriteString(fmt.Sprintf("  Role: %s\n", user.Role))
	output.WriteString(fmt.Sprintf("  DateOfBirth: %s\n", user.DateOfBirth))
	output.WriteString("========================================\n")

	if user.UserID != userID {
		errorMessage = fmt.Sprintf("Expected UserID %s, got %s", userID, user.UserID)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Errorf("%s", errorMessage)
		return
	}
	if user.Email != loginResp.Email {
		errorMessage = fmt.Sprintf("Expected Email %s, got %s", loginResp.Email, user.Email)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Errorf("%s", errorMessage)
		return
	}

	recordTestResult(testName, true, output.String(), "")
}

// TestGetCurrentUser tests retrieving the current user by JWT token from cache.
func TestGetCurrentUser(t *testing.T) {
	var (
		testName     = "TestGetCurrentUser"
		service      *Service
		loginResp    *LoginResponse
		ctx          context.Context
		currentUser  *User
		output       bytes.Buffer
		errorMessage string
		err          error
	)

	// setup
	service, loginResp, err = setupTestUser(
		"getcurrentuser@example22.com",
		"SecurePassword123!",
		UserMetadata{
			DisplayName: "Current User Test",
			Username:    "currentusertest",
			Role:        "user",
			DateOfBirth: "1993-07-20",
		},
	)
	if err != nil {
		errorMessage = fmt.Sprintf("Setup failed: %v", err)
		recordTestResult(testName, false, "", errorMessage)
		t.Fatalf("%s", errorMessage)
		return
	}

	ctx = context.Background()

	output.WriteString("\n========================================\n")
	output.WriteString("Testing GetCurrentUser\n")
	output.WriteString("========================================\n")

	// execute
	currentUser, err = service.GetCurrentUser(ctx, loginResp.Token)
	if err != nil {
		errorMessage = fmt.Sprintf("GetCurrentUser failed: %v", err)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Fatalf("%s", errorMessage)
		return
	}

	// verify
	output.WriteString("✓ Current user retrieved by token\n")
	output.WriteString(fmt.Sprintf("  UserID: %s\n", currentUser.UserID))
	output.WriteString(fmt.Sprintf("  Email: %s\n", currentUser.Email))
	output.WriteString(fmt.Sprintf("  Username: %s\n", currentUser.Username))
	output.WriteString(fmt.Sprintf("  DisplayName: %s\n", currentUser.DisplayName))
	output.WriteString(fmt.Sprintf("  Role: %s\n", currentUser.Role))
	output.WriteString(fmt.Sprintf("  DateOfBirth: %s\n", currentUser.DateOfBirth))
	output.WriteString("========================================\n")

	if currentUser.Email != loginResp.Email {
		errorMessage = fmt.Sprintf("Expected Email %s, got %s", loginResp.Email, currentUser.Email)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Errorf("%s", errorMessage)
		return
	}
	if currentUser.Username != loginResp.Username {
		errorMessage = fmt.Sprintf("Expected Username %s, got %s", loginResp.Username, currentUser.Username)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Errorf("%s", errorMessage)
		return
	}

	recordTestResult(testName, true, output.String(), "")
}

// TestUpdateUser tests updating user metadata.
func TestUpdateUser(t *testing.T) {
	var (
		testName     = "TestUpdateUser"
		service      *Service
		loginResp    *LoginResponse
		ctx          context.Context
		userID       uuid.UUID
		updates      map[string]any
		updatedUser  *User
		cachedUser   *CachedUser
		found        bool
		output       bytes.Buffer
		errorMessage string
		err          error
	)

	// setup
	service, loginResp, err = setupTestUser(
		"updateuser@example22.com",
		"SecurePassword123!",
		UserMetadata{
			DisplayName: "Update Test",
			Username:    "updatetest",
			Role:        "user",
			DateOfBirth: "1991-12-05",
		},
	)
	if err != nil {
		errorMessage = fmt.Sprintf("Setup failed: %v", err)
		recordTestResult(testName, false, "", errorMessage)
		t.Fatalf("%s", errorMessage)
		return
	}

	ctx = context.Background()

	output.WriteString("\n========================================\n")
	output.WriteString("Testing UpdateUser\n")
	output.WriteString("========================================\n")

	// parse user ID
	userID, err = uuid.Parse(loginResp.ID)
	if err != nil {
		errorMessage = fmt.Sprintf("Failed to parse user ID: %v", err)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Fatalf("%s", errorMessage)
		return
	}

	// prepare updates
	updates = map[string]any{
		"display_name": "Updated Display Name",
		"role":         "admin",
	}

	// execute
	updatedUser, err = service.UpdateUser(ctx, userID, updates)
	if err != nil {
		errorMessage = fmt.Sprintf("UpdateUser failed: %v", err)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Fatalf("%s", errorMessage)
		return
	}

	// verify
	output.WriteString("✓ User updated successfully\n")
	output.WriteString(fmt.Sprintf("  UserID: %s\n", updatedUser.UserID))
	output.WriteString(fmt.Sprintf("  Email: %s\n", updatedUser.Email))
	output.WriteString(fmt.Sprintf("  Username: %s\n", updatedUser.Username))
	output.WriteString(fmt.Sprintf("  DisplayName: %s (updated)\n", updatedUser.DisplayName))
	output.WriteString(fmt.Sprintf("  Role: %s (updated)\n", updatedUser.Role))
	output.WriteString(fmt.Sprintf("  DateOfBirth: %s\n", updatedUser.DateOfBirth))

	if updatedUser.DisplayName != "Updated Display Name" {
		errorMessage = fmt.Sprintf("Expected DisplayName 'Updated Display Name', got %s", updatedUser.DisplayName)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Errorf("%s", errorMessage)
		return
	}
	if updatedUser.Role != "admin" {
		errorMessage = fmt.Sprintf("Expected Role 'admin', got %s", updatedUser.Role)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Errorf("%s", errorMessage)
		return
	}

	// verify cache was updated
	cachedUser, found = service.Cache.Get(loginResp.Token)
	if !found {
		errorMessage = "User should still be in cache after update"
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Error(errorMessage)
		return
	} else {
		output.WriteString("\n✓ Cache updated:\n")
		output.WriteString(fmt.Sprintf("  Cached DisplayName: %s\n", cachedUser.DisplayName))
		output.WriteString(fmt.Sprintf("  Cached Role: %s\n", cachedUser.Role))

		if cachedUser.DisplayName != "Updated Display Name" {
			errorMessage = fmt.Sprintf("Cache not updated: expected DisplayName 'Updated Display Name', got %s", cachedUser.DisplayName)
			recordTestResult(testName, false, output.String(), errorMessage)
			t.Errorf("%s", errorMessage)
			return
		}
		if cachedUser.Role != "admin" {
			errorMessage = fmt.Sprintf("Cache not updated: expected Role 'admin', got %s", cachedUser.Role)
			recordTestResult(testName, false, output.String(), errorMessage)
			t.Errorf("%s", errorMessage)
			return
		}
	}
	output.WriteString("========================================\n")

	recordTestResult(testName, true, output.String(), "")
}

// TestLogout tests user logout functionality.
func TestLogout(t *testing.T) {
	var (
		testName          = "TestLogout"
		service           *Service
		loginResp         *LoginResponse
		ctx               context.Context
		foundBeforeLogout bool
		foundAfterLogout  bool
		isValid           bool
		output            bytes.Buffer
		errorMessage      string
		err               error
	)

	// setup
	service, loginResp, err = setupTestUser(
		"logout@example22.com",
		"SecurePassword123!",
		UserMetadata{
			DisplayName: "Logout Test",
			Username:    "logouttest",
			Role:        "user",
			DateOfBirth: "1994-08-15",
		},
	)
	if err != nil {
		errorMessage = fmt.Sprintf("Setup failed: %v", err)
		recordTestResult(testName, false, "", errorMessage)
		t.Fatalf("%s", errorMessage)
		return
	}

	ctx = context.Background()

	output.WriteString("\n========================================\n")
	output.WriteString("Testing Logout\n")
	output.WriteString("========================================\n")

	// verify user exists in cache before logout
	_, foundBeforeLogout = service.Cache.Get(loginResp.Token)
	output.WriteString(fmt.Sprintf("User in cache before logout: %v\n", foundBeforeLogout))

	if !foundBeforeLogout {
		errorMessage = "User should be in cache before logout"
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Error(errorMessage)
		return
	}

	// execute
	err = service.Logout(ctx, loginResp.Token)
	if err != nil {
		errorMessage = fmt.Sprintf("Logout failed: %v", err)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Fatalf("%s", errorMessage)
		return
	}

	// verify
	output.WriteString("✓ User logged out from Supabase\n")

	// verify user removed from cache
	_, foundAfterLogout = service.Cache.Get(loginResp.Token)
	if foundAfterLogout {
		errorMessage = "User should not be in cache after logout"
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Error(errorMessage)
		return
	} else {
		output.WriteString("✓ User removed from cache\n")
	}

	// verify token is no longer valid
	isValid = service.Cache.IsValid(loginResp.Token)
	if isValid {
		errorMessage = "Token should not be valid after logout"
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Error(errorMessage)
		return
	} else {
		output.WriteString("✓ Token is no longer valid\n")
	}
	output.WriteString("========================================\n")

	recordTestResult(testName, true, output.String(), "")
}

// TestDeleteUser tests user deletion functionality.
func TestDeleteUser(t *testing.T) {
	var (
		testName          = "TestDeleteUser"
		service           *Service
		loginResp         *LoginResponse
		ctx               context.Context
		userID            uuid.UUID
		foundBeforeDelete bool
		foundAfterDelete  bool
		output            bytes.Buffer
		errorMessage      string
		err               error
	)

	// setup
	service, loginResp, err = setupTestUser(
		"deleteuser@example22.com",
		"SecurePassword123!",
		UserMetadata{
			DisplayName: "Delete Test",
			Username:    "deletetest",
			Role:        "user",
			DateOfBirth: "1989-11-25",
		},
	)
	if err != nil {
		errorMessage = fmt.Sprintf("Setup failed: %v", err)
		recordTestResult(testName, false, "", errorMessage)
		t.Fatalf("%s", errorMessage)
		return
	}

	ctx = context.Background()

	output.WriteString("\n========================================\n")
	output.WriteString("Testing DeleteUser\n")
	output.WriteString("========================================\n")

	// parse user ID
	userID, err = uuid.Parse(loginResp.ID)
	if err != nil {
		errorMessage = fmt.Sprintf("Failed to parse user ID: %v", err)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Fatalf("%s", errorMessage)
		return
	}

	// verify user exists in cache before deletion
	_, foundBeforeDelete = service.Cache.GetByUserID(userID)
	output.WriteString(fmt.Sprintf("User in cache before delete: %v\n", foundBeforeDelete))

	if !foundBeforeDelete {
		errorMessage = "User should be in cache before deletion"
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Error(errorMessage)
		return
	}

	// execute
	err = service.DeleteUser(ctx, userID)
	if err != nil {
		errorMessage = fmt.Sprintf("DeleteUser failed: %v", err)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Fatalf("%s", errorMessage)
		return
	}

	// verify
	output.WriteString("✓ User deleted from Supabase\n")

	// verify user removed from cache
	_, foundAfterDelete = service.Cache.GetByUserID(userID)
	if foundAfterDelete {
		errorMessage = "User should not be in cache after deletion"
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Error(errorMessage)
		return
	} else {
		output.WriteString("✓ User removed from cache\n")
	}
	output.WriteString("========================================\n")

	recordTestResult(testName, true, output.String(), "")
}

// TestRefreshToken tests token refresh functionality.
func TestRefreshToken(t *testing.T) {
	var (
		testName         = "TestRefreshToken"
		service          *Service
		loginResp        *LoginResponse
		ctx              context.Context
		refreshResp      *RefreshTokenResponse
		oldToken         string
		oldRefreshToken  string
		cachedUserBefore *CachedUser
		cachedUserAfter  *CachedUser
		foundBefore      bool
		foundAfter       bool
		output           bytes.Buffer
		errorMessage     string
		err              error
	)

	// setup
	service, loginResp, err = setupTestUser(
		"refreshtoken@example22.com",
		"SecurePassword123!",
		UserMetadata{
			DisplayName: "Refresh Token Test",
			Username:    "refreshtest",
			Role:        "user",
			DateOfBirth: "1996-02-28",
		},
	)
	if err != nil {
		errorMessage = fmt.Sprintf("Setup failed: %v", err)
		recordTestResult(testName, false, "", errorMessage)
		t.Fatalf("%s", errorMessage)
		return
	}

	ctx = context.Background()

	output.WriteString("\n========================================\n")
	output.WriteString("Testing RefreshToken\n")
	output.WriteString("========================================\n")

	// get old tokens and cached user
	cachedUserBefore, foundBefore = service.Cache.Get(loginResp.Token)
	if !foundBefore {
		errorMessage = "User should be in cache before refresh"
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Error(errorMessage)
		return
	}

	oldToken = loginResp.Token
	oldRefreshToken = cachedUserBefore.RefreshToken

	// safely truncate tokens for display
	accessTokenDisplay := oldToken
	if len(accessTokenDisplay) > 30 {
		accessTokenDisplay = accessTokenDisplay[:30] + "..."
	}
	refreshTokenDisplay := oldRefreshToken
	if len(refreshTokenDisplay) > 30 {
		refreshTokenDisplay = refreshTokenDisplay[:30] + "..."
	}

	output.WriteString(fmt.Sprintf("Old access token: %s\n", accessTokenDisplay))
	output.WriteString(fmt.Sprintf("Old refresh token: %s\n", refreshTokenDisplay))

	// execute
	refreshResp, err = service.RefreshToken(ctx, oldRefreshToken)
	if err != nil {
		errorMessage = fmt.Sprintf("RefreshToken failed: %v", err)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Fatalf("%s", errorMessage)
		return
	}

	// verify
	output.WriteString("\n✓ Token refreshed successfully\n")

	// safely truncate new tokens for display
	newAccessTokenDisplay := refreshResp.AccessToken
	if len(newAccessTokenDisplay) > 30 {
		newAccessTokenDisplay = newAccessTokenDisplay[:30] + "..."
	}
	newRefreshTokenDisplay := refreshResp.RefreshToken
	if len(newRefreshTokenDisplay) > 30 {
		newRefreshTokenDisplay = newRefreshTokenDisplay[:30] + "..."
	}

	output.WriteString(fmt.Sprintf("  New AccessToken: %s\n", newAccessTokenDisplay))
	output.WriteString(fmt.Sprintf("  New RefreshToken: %s\n", newRefreshTokenDisplay))
	output.WriteString(fmt.Sprintf("  ID: %s\n", refreshResp.ID))
	output.WriteString(fmt.Sprintf("  Email: %s\n", refreshResp.Email))
	output.WriteString(fmt.Sprintf("  Username: %s\n", refreshResp.Username))
	output.WriteString(fmt.Sprintf("  Role: %s\n", refreshResp.Role))
	output.WriteString(fmt.Sprintf("  ExpiresIn: %d seconds\n", refreshResp.ExpiresIn))

	// verify tokens (may be same or different depending on Supabase reuse window)
	// Note: Supabase allows refresh token reuse within 10 seconds, which may return the same access token
	if refreshResp.AccessToken == oldToken {
		output.WriteString("\n⚠ Note: Same access token returned (within Supabase 10s reuse window)\n")
	} else {
		output.WriteString("\n✓ New access token generated\n")
	}

	// verify cache behavior (depends on whether token was reused)
	_, foundOldToken := service.Cache.Get(oldToken)
	if refreshResp.AccessToken == oldToken {
		// Same token returned - old token should still be in cache (but refreshed)
		if !foundOldToken {
			errorMessage = "Token should still be in cache when reused"
			recordTestResult(testName, false, output.String(), errorMessage)
			t.Error(errorMessage)
			return
		} else {
			output.WriteString("✓ Token still in cache (reused)\n")
		}
	} else {
		// New token returned - old token should be removed from cache
		if foundOldToken {
			errorMessage = "Old token should be removed from cache when new token generated"
			recordTestResult(testName, false, output.String(), errorMessage)
			t.Error(errorMessage)
			return
		} else {
			output.WriteString("✓ Old token removed from cache\n")
		}
	}

	// verify new token is in cache
	cachedUserAfter, foundAfter = service.Cache.Get(refreshResp.AccessToken)
	if !foundAfter {
		errorMessage = "New token should be in cache"
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Error(errorMessage)
		return
	} else {
		output.WriteString("✓ New token added to cache\n")
		output.WriteString(fmt.Sprintf("  Cached UserID: %s\n", cachedUserAfter.UserID))
		output.WriteString(fmt.Sprintf("  Cached Email: %s\n", cachedUserAfter.Email))
		output.WriteString(fmt.Sprintf("  Cached Username: %s\n", cachedUserAfter.Username))
	}

	// verify user data consistency
	if refreshResp.Email != loginResp.Email {
		errorMessage = fmt.Sprintf("Email mismatch: expected %s, got %s", loginResp.Email, refreshResp.Email)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Errorf("%s", errorMessage)
		return
	}
	if refreshResp.Username != loginResp.Username {
		errorMessage = fmt.Sprintf("Username mismatch: expected %s, got %s", loginResp.Username, refreshResp.Username)
		recordTestResult(testName, false, output.String(), errorMessage)
		t.Errorf("%s", errorMessage)
		return
	}

	output.WriteString("\n✓ User data consistency verified\n")
	output.WriteString("========================================\n")

	recordTestResult(testName, true, output.String(), "")
}
