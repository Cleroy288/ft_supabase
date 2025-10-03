package ft_supabase

import (
	"context"
	"fmt"
	"testing"
)

const (
	testProjectID  = ""
	testProjectURL = ""
	testAnonKey    = ""
	testServiceKey = ""
)

func TestRegisterUser(t *testing.T) {
	service := NewService(testProjectID, testProjectURL, testAnonKey, testServiceKey)
	ctx := context.Background()

	email := "testuser@example22.com"
	password := "SecurePassword123!"
	phone := ""
	metadata := UserMetadata{
		DisplayName: "Test User2",
		Username:    "testuser2",
		Role:        "user",
		DateOfBirth: "1990-01-01",
	}

	fmt.Println("\n========================================")
	fmt.Println("Testing RegisterUser")
	fmt.Println("========================================")

	resp, err := service.RegisterUser(ctx, email, password, phone, metadata)
	if err != nil {
		t.Fatalf("RegisterUser failed: %v", err)
	}

	fmt.Printf("\n✓ Registration successful!\n")
	fmt.Printf("  ID: %s\n", resp.ID)
	fmt.Printf("  Email: %s\n", resp.Email)
	fmt.Printf("  Username: %s\n", resp.UserName)
	fmt.Printf("  Role: %s\n", resp.Role)
	fmt.Println("========================================")

	if resp.Email != email {
		t.Errorf("Expected email %s, got %s", email, resp.Email)
	}
	if resp.UserName != metadata.Username {
		t.Errorf("Expected username %s, got %s", metadata.Username, resp.UserName)
	}
	if resp.Role != metadata.Role {
		t.Errorf("Expected role %s, got %s", metadata.Role, resp.Role)
	}
}

func TestLoginUser(t *testing.T) {
	service := NewService(testProjectID, testProjectURL, testAnonKey, testServiceKey)
	ctx := context.Background()

	// First, register a user
	email := "logintest@example22.com"
	password := "SecurePassword123!"
	phone := ""
	metadata := UserMetadata{
		DisplayName: "Login User2",
		Username:    "loginuser2",
		Role:        "user",
		DateOfBirth: "1995-05-15",
	}

	fmt.Println("\n========================================")
	fmt.Println("Testing LoginUser (registering first)")
	fmt.Println("========================================")

	// Register the user first
	_, err := service.RegisterUser(ctx, email, password, phone, metadata)
	if err != nil {
		// User might already exist, continue with login test
		fmt.Printf("⚠ Registration failed (user may already exist): %v\n", err)
	}

	// Now test login
	fmt.Println("\n========================================")
	fmt.Println("Logging in...")
	fmt.Println("========================================")

	loginResp, err := service.LoginUser(ctx, email, password)
	if err != nil {
		t.Fatalf("LoginUser failed: %v", err)
	}

	fmt.Printf("\n✓ Login successful!\n")
	fmt.Printf("  Token: %s...\n", loginResp.Token[:50])
	fmt.Printf("  ID: %s\n", loginResp.ID)
	fmt.Printf("  Email: %s\n", loginResp.Email)
	fmt.Printf("  Username: %s\n", loginResp.Username)
	fmt.Printf("  Role: %s\n", loginResp.Role)
	fmt.Println("========================================")

	if loginResp.Email != email {
		t.Errorf("Expected email %s, got %s", email, loginResp.Email)
	}
	if loginResp.Token == "" {
		t.Error("Expected non-empty token")
	}

	// Test cache functionality
	fmt.Println("\n========================================")
	fmt.Println("Testing Cache")
	fmt.Println("========================================")

	// Check if user is in cache
	cachedUser, found := service.Cache.Get(loginResp.Token)
	if !found {
		t.Error("User should be in cache after login")
	} else {
		fmt.Printf("✓ User found in cache\n")
		fmt.Printf("  Cached UserID: %s\n", cachedUser.UserID)
		fmt.Printf("  Cached Email: %s\n", cachedUser.Email)
		fmt.Printf("  Cached Username: %s\n", cachedUser.Username)
		fmt.Printf("  Token valid: %v\n", service.Cache.IsValid(loginResp.Token))
		fmt.Printf("  Cache count: %d\n", service.Cache.Count())
	}

	// Test GetUserByID
	fmt.Println("\n========================================")
	fmt.Println("Testing GetUserByID")
	fmt.Println("========================================")

	user, err := service.GetUserByID(ctx, loginResp.ID)
	if err != nil {
		t.Errorf("GetUserByID failed: %v", err)
	} else {
		fmt.Printf("✓ User retrieved by ID\n")
		fmt.Printf("  UserID: %s\n", user.UserID)
		fmt.Printf("  Email: %s\n", user.Email)
		fmt.Printf("  Username: %s\n", user.Username)
		fmt.Printf("  DisplayName: %s\n", user.DisplayName)
		fmt.Printf("  Role: %s\n", user.Role)
		fmt.Printf("  DateOfBirth: %s\n", user.DateOfBirth)
	}
	fmt.Println("========================================")

	// Test UpdateUser
	fmt.Println("\n========================================")
	fmt.Println("Testing UpdateUser")
	fmt.Println("========================================")

	updates := map[string]any{
		"display_name": "Updated User Name",
		"role":         "admin",
	}

	updatedUser, err := service.UpdateUser(ctx, loginResp.ID, updates)
	if err != nil {
		t.Errorf("UpdateUser failed: %v", err)
	} else {
		fmt.Printf("✓ User updated successfully\n")
		fmt.Printf("  UserID: %s\n", updatedUser.UserID)
		fmt.Printf("  Email: %s\n", updatedUser.Email)
		fmt.Printf("  Username: %s\n", updatedUser.Username)
		fmt.Printf("  DisplayName: %s (updated)\n", updatedUser.DisplayName)
		fmt.Printf("  Role: %s (updated)\n", updatedUser.Role)
		fmt.Printf("  DateOfBirth: %s\n", updatedUser.DateOfBirth)

		// Verify cache was updated
		cachedAfterUpdate, foundAfterUpdate := service.Cache.Get(loginResp.Token)
		if foundAfterUpdate {
			fmt.Printf("\n✓ Cache updated:\n")
			fmt.Printf("  Cached DisplayName: %s\n", cachedAfterUpdate.DisplayName)
			fmt.Printf("  Cached Role: %s\n", cachedAfterUpdate.Role)
		}
	}
	fmt.Println("========================================")

	// Test DeleteUser
	fmt.Println("\n========================================")
	fmt.Println("Testing DeleteUser")
	fmt.Println("========================================")

	// Verify user exists in cache before deletion
	_, foundBeforeDelete := service.Cache.GetByUserID(updatedUser.UserID)
	fmt.Printf("User in cache before delete: %v\n", foundBeforeDelete)

	// Delete user from Supabase and cache
	err = service.DeleteUser(ctx, loginResp.ID)
	if err != nil {
		t.Errorf("DeleteUser failed: %v", err)
	} else {
		fmt.Printf("✓ User deleted from Supabase\n")

		// Verify user removed from cache
		_, foundAfterDelete := service.Cache.GetByUserID(updatedUser.UserID)
		if foundAfterDelete {
			t.Error("User should not be in cache after deletion")
		} else {
			fmt.Printf("✓ User removed from cache\n")
		}
	}
	fmt.Println("========================================")
}
