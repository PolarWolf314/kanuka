package revoke

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/PolarWolf314/kanuka/cmd"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/google/uuid"
)

func TestRevokeCommand_LargeNumberOfUsers(t *testing.T) {
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
		testRevokeWithLargeNumberOfUsers(t, originalWd, originalUserSettings)
	})
}

func testRevokeWithLargeNumberOfUsers(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
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
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Fatalf("Failed to restore working directory: %v", err)
		}
	}()

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

	// Create project config to prevent legacy migration
	configs.ProjectKanukaSettings.ProjectPath = tempDir
	projectConfig := &configs.ProjectConfig{
		Project: configs.Project{
			UUID: uuid.New().String(),
			Name: "test-project",
		},
		Users:   make(map[string]string),
		Devices: make(map[string]configs.DeviceConfig),
	}

	// Create a large number of dummy user files using UUIDs
	numUsers := 100
	var userToRemoveUUID string
	userUUIDs := make([]string, numUsers+1) // 1-indexed

	t.Logf("Creating %d users...", numUsers)
	for i := 1; i <= numUsers; i++ {
		userUUID := uuid.New().String()
		userUUIDs[i] = userUUID
		email := fmt.Sprintf("user%d@example.com", i)

		if i == 50 {
			userToRemoveUUID = userUUID
		}

		// Add to project config
		projectConfig.Users[userUUID] = email
		projectConfig.Devices[userUUID] = configs.DeviceConfig{
			Email:     email,
			Name:      fmt.Sprintf("device%d", i),
			CreatedAt: time.Now().UTC(),
		}

		// Create dummy public key file
		publicKeyPath := filepath.Join(publicKeysDir, userUUID+".pub")
		if err := os.WriteFile(publicKeyPath, []byte(fmt.Sprintf("dummy public key for user%d", i)), 0600); err != nil {
			t.Fatalf("Failed to create public key file for user %d: %v", i, err)
		}

		// Create dummy kanuka key file
		kanukaKeyPath := filepath.Join(secretsDir, userUUID+".kanuka")
		if err := os.WriteFile(kanukaKeyPath, []byte(fmt.Sprintf("dummy kanuka key for user%d", i)), 0600); err != nil {
			t.Fatalf("Failed to create kanuka key file for user %d: %v", i, err)
		}
	}

	// Save project config
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// Measure time to revoke a user
	start := time.Now()

	// Remove the user using --file flag (use relative path)
	relativeKanukaKeyPath := filepath.Join(".kanuka", "secrets", userToRemoveUUID+".kanuka")
	cmd.ResetGlobalState()
	secretsCmd := cmd.GetSecretsCmd()
	secretsCmd.SetArgs([]string{"revoke", "--file", relativeKanukaKeyPath})
	if err := secretsCmd.Execute(); err != nil {
		t.Errorf("Remove command should succeed: %v", err)
	}

	duration := time.Since(start)
	t.Logf("Time to remove user from %d users: %v", numUsers, duration)

	// Verify the user's files are removed
	publicKeyPath := filepath.Join(publicKeysDir, userToRemoveUUID+".pub")
	kanukaKeyPath := filepath.Join(secretsDir, userToRemoveUUID+".kanuka")

	if _, err := os.Stat(publicKeyPath); !os.IsNotExist(err) {
		t.Error("Public key file should be revoked")
	}

	if _, err := os.Stat(kanukaKeyPath); !os.IsNotExist(err) {
		t.Error("Kanuka key file should be revoked")
	}

	// Verify other users' files still exist (check a few random ones)
	samplesToCheck := []int{1, 25, 75, 100}
	for _, i := range samplesToCheck {
		if i == 50 {
			continue // This is the revoked user
		}

		userUUID := userUUIDs[i]
		publicKeyPath := filepath.Join(publicKeysDir, userUUID+".pub")
		kanukaKeyPath := filepath.Join(secretsDir, userUUID+".kanuka")

		if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
			t.Errorf("Public key file for user%d should still exist", i)
		}

		if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
			t.Errorf("Kanuka key file for user%d should still exist", i)
		}
	}

	// Performance assertion - removal should be reasonably fast
	if duration > 2*time.Second {
		t.Logf("Warning: User removal took longer than expected (%v) with %d users", duration, numUsers)
	}
}
