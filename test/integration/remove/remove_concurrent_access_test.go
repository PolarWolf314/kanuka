package remove

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/PolarWolf314/kanuka/cmd"
	"github.com/PolarWolf314/kanuka/internal/configs"
)

func TestRemoveCommand_ConcurrentAccess(t *testing.T) {
	// Skip on Windows as file locking works differently
	if os.Getenv("SKIP_CONCURRENT_TESTS") == "true" {
		t.Skip("Skipping concurrent access tests")
	}

	// Save original state
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	t.Run("RemoveWithFileBeingAccessed", func(t *testing.T) {
		testRemoveWithFileBeingAccessed(t, originalWd, originalUserSettings)
	})
}

func testRemoveWithFileBeingAccessed(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
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

	// Create test user files
	testUser := "testuser2"
	publicKeyPath := filepath.Join(publicKeysDir, testUser+".pub")
	kanukaKeyPath := filepath.Join(secretsDir, testUser+".kanuka")

	// Create dummy files
	err = os.WriteFile(publicKeyPath, []byte("dummy public key"), 0600)
	if err != nil {
		t.Fatalf("Failed to create public key file: %v", err)
	}

	err = os.WriteFile(kanukaKeyPath, []byte("dummy kanuka key"), 0600)
	if err != nil {
		t.Fatalf("Failed to create kanuka key file: %v", err)
	}

	// Open the file to simulate it being accessed by another process
	file, err := os.Open(publicKeyPath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	// Create a channel to signal when the remove command is done
	done := make(chan bool)

	// Run the remove command in a goroutine
	go func() {
		cmd.ResetGlobalState()
		secretsCmd := cmd.GetSecretsCmd()
		secretsCmd.SetArgs([]string{"remove", "--user", testUser})
		err := secretsCmd.Execute()
		if err != nil {
			t.Errorf("Remove command should not return error even with concurrent access: %v", err)
		}
		done <- true
	}()

	// Wait for the command to complete or timeout
	select {
	case <-done:
		// Command completed
	case <-time.After(5 * time.Second):
		t.Fatal("Remove command timed out")
	}

	// Close the file to allow cleanup
	file.Close()

	// Check if the kanuka key file was removed (it should be since we only locked the public key)
	if _, err := os.Stat(kanukaKeyPath); !os.IsNotExist(err) {
		t.Error("Kanuka key file should be removed even if public key file is locked")
	}
}
