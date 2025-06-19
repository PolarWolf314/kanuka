package remove

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/PolarWolf314/kanuka/cmd"
	"github.com/PolarWolf314/kanuka/internal/configs"
)

func TestRemoveCommand_LargeNumberOfUsers(t *testing.T) {
	// Skip this test in CI environments or when running quick tests
	if os.Getenv("SKIP_PERFORMANCE_TESTS") == "true" {
		t.Skip("Skipping performance test for large number of users")
	}

	// Save original state
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	t.Run("RemoveWithLargeNumberOfUsers", func(t *testing.T) {
		testRemoveWithLargeNumberOfUsers(t, originalWd, originalUserSettings)
	})
}

func testRemoveWithLargeNumberOfUsers(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Setup test environment
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Setup user settings
	configs.UserKanukaSettings = &configs.UserSettings{
		UserKeysPath:    filepath.Join(tempUserDir, "keys"),
		UserConfigsPath: filepath.Join(tempUserDir, "config"),
		Username:        "testuser",
	}
	defer func() {
		configs.UserKanukaSettings = originalUserSettings
	}()

	// Create kanuka directories
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
		t.Fatalf("Failed to create public keys directory: %v", err)
	}
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		t.Fatalf("Failed to create secrets directory: %v", err)
	}

	// Create a large number of dummy user files
	numUsers := 100
	userToRemove := "user50" // We'll remove this user

	t.Logf("Creating %d users...", numUsers)
	for i := 1; i <= numUsers; i++ {
		username := fmt.Sprintf("user%d", i)
		
		// Create dummy public key file
		publicKeyPath := filepath.Join(publicKeysDir, username+".pub")
		if err := os.WriteFile(publicKeyPath, []byte(fmt.Sprintf("dummy public key for %s", username)), 0644); err != nil {
			t.Fatalf("Failed to create public key file for user %s: %v", username, err)
		}
		
		// Create dummy kanuka key file
		kanukaKeyPath := filepath.Join(secretsDir, username+".kanuka")
		if err := os.WriteFile(kanukaKeyPath, []byte(fmt.Sprintf("dummy kanuka key for %s", username)), 0600); err != nil {
			t.Fatalf("Failed to create kanuka key file for user %s: %v", username, err)
		}
	}

	// Measure time to remove a user
	start := time.Now()
	
	// Remove the user
	cmd.ResetGlobalState()
	secretsCmd := cmd.GetSecretsCmd()
	secretsCmd.SetArgs([]string{"remove", "--user", userToRemove})
	if err := secretsCmd.Execute(); err != nil {
		t.Errorf("Remove command should succeed: %v", err)
	}
	
	duration := time.Since(start)
	t.Logf("Time to remove user from %d users: %v", numUsers, duration)

	// Verify the user's files are removed
	publicKeyPath := filepath.Join(publicKeysDir, userToRemove+".pub")
	kanukaKeyPath := filepath.Join(secretsDir, userToRemove+".kanuka")
	
	if _, err := os.Stat(publicKeyPath); !os.IsNotExist(err) {
		t.Error("Public key file should be removed")
	}
	
	if _, err := os.Stat(kanukaKeyPath); !os.IsNotExist(err) {
		t.Error("Kanuka key file should be removed")
	}

	// Verify other users' files still exist (check a few random ones)
	samplesToCheck := []int{1, 25, 75, 100}
	for _, i := range samplesToCheck {
		username := fmt.Sprintf("user%d", i)
		if username == userToRemove {
			continue
		}
		
		publicKeyPath := filepath.Join(publicKeysDir, username+".pub")
		kanukaKeyPath := filepath.Join(secretsDir, username+".kanuka")
		
		if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
			t.Errorf("Public key file for user %s should still exist", username)
		}
		
		if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
			t.Errorf("Kanuka key file for user %s should still exist", username)
		}
	}

	// Performance assertion - removal should be reasonably fast
	if duration > 2*time.Second {
		t.Logf("Warning: User removal took longer than expected (%v) with %d users", duration, numUsers)
	}
}