package revoke

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/PolarWolf314/kanuka/cmd"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

func TestRevokeCommand_LargeNumberOfUsers(t *testing.T) {
	// Skip this test in CI environments or when running quick tests.
	if os.Getenv("SKIP_PERFORMANCE_TESTS") == "true" {
		t.Skip("Skipping performance test for large number of users")
	}

	// Save original state.
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
	// Setup test environment.
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

	// Change to temp directory.
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Fatalf("Failed to restore working directory: %v", err)
		}
	}()

	// Setup user settings.
	configs.UserKanukaSettings = &configs.UserSettings{
		UserKeysPath:    filepath.Join(tempUserDir, "keys"),
		UserConfigsPath: filepath.Join(tempUserDir, "config"),
		Username:        "testuser",
	}
	defer func() {
		configs.UserKanukaSettings = originalUserSettings
	}()

	// Create required directories.
	if err := os.MkdirAll(configs.UserKanukaSettings.UserKeysPath, 0755); err != nil {
		t.Fatalf("Failed to create user keys directory: %v", err)
	}
	if err := os.MkdirAll(configs.UserKanukaSettings.UserConfigsPath, 0755); err != nil {
		t.Fatalf("Failed to create user configs directory: %v", err)
	}

	// Create user config.
	userConfig := &configs.UserConfig{
		User: configs.User{
			UUID:  shared.TestUserUUID,
			Email: shared.TestUserEmail,
		},
		Projects: make(map[string]configs.UserProjectEntry),
	}
	if err := configs.SaveUserConfig(userConfig); err != nil {
		t.Fatalf("Failed to save user config: %v", err)
	}

	// Initialize the project using the CLI command (creates keys properly).
	cmd.ResetGlobalState()
	initCmd := shared.CreateTestCLIWithArgs("init", []string{"--yes"}, nil, nil, false, false)
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}

	// Generate users and register them.
	numUsers := 100
	userToRemoveIndex := 50

	type userInfo struct {
		uuid   string
		email  string
		pubKey string
	}

	users := make([]userInfo, numUsers)
	t.Logf("Creating %d users...", numUsers)

	for i := 0; i < numUsers; i++ {
		userUUID := fmt.Sprintf("user%d-uuid-1234", i+1)
		email := fmt.Sprintf("user%d@example.com", i+1)

		// Generate RSA key pair for this user.
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("Failed to generate RSA key for user %d: %v", i+1, err)
		}

		pubASN1, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
		if err != nil {
			t.Fatalf("Failed to marshal public key for user %d: %v", i+1, err)
		}

		pubPem := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubASN1,
		}
		pubKeyStr := string(pem.EncodeToMemory(pubPem))

		users[i] = userInfo{
			uuid:   userUUID,
			email:  email,
			pubKey: pubKeyStr,
		}

		// Add user to project config.
		projectConfig, err := configs.LoadProjectConfig()
		if err != nil {
			t.Fatalf("Failed to load project config: %v", err)
		}
		projectConfig.Users[userUUID] = email
		projectConfig.Devices[userUUID] = configs.DeviceConfig{
			Email:     email,
			Name:      fmt.Sprintf("device%d", i+1),
			CreatedAt: time.Now().UTC(),
		}
		if err := configs.SaveProjectConfig(projectConfig); err != nil {
			t.Fatalf("Failed to save project config: %v", err)
		}

		// Register user.
		cmd.ResetGlobalState()
		registerCmd := shared.CreateTestCLIWithArgs("register", []string{"--pubkey", pubKeyStr, "--user", email}, nil, nil, false, false)
		if err := registerCmd.Execute(); err != nil {
			t.Fatalf("Failed to register user %s: %v", email, err)
		}
	}

	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	// Measure time to revoke a user.
	start := time.Now()

	// Revoke user using --user flag.
	userToRemove := users[userToRemoveIndex-1] // Index is 0-based, user is 50th (index 49).
	cmd.ResetGlobalState()
	revokeCmd := shared.CreateTestCLIWithArgs("revoke", []string{"--user", userToRemove.email, "--yes"}, nil, nil, false, false)
	if err := revokeCmd.Execute(); err != nil {
		t.Errorf("Revoke command should succeed: %v", err)
	}

	duration := time.Since(start)
	t.Logf("Time to revoke user from %d users: %v", numUsers, duration)

	// Verify the user's files are removed.
	publicKeyPath := filepath.Join(publicKeysDir, userToRemove.uuid+".pub")
	kanukaKeyPath := filepath.Join(secretsDir, userToRemove.uuid+".kanuka")

	if _, err := os.Stat(publicKeyPath); !os.IsNotExist(err) {
		t.Error("Public key file should be revoked")
	}

	if _, err := os.Stat(kanukaKeyPath); !os.IsNotExist(err) {
		t.Error("Kanuka key file should be revoked")
	}

	// Verify other users' files still exist (check a few random ones).
	samplesToCheck := []int{0, 24, 74, 99} // 0-indexed: users 1, 25, 75, 100
	for _, i := range samplesToCheck {
		if i == userToRemoveIndex-1 {
			continue // This is the revoked user
		}

		user := users[i]
		publicKeyPath := filepath.Join(publicKeysDir, user.uuid+".pub")
		kanukaKeyPath := filepath.Join(secretsDir, user.uuid+".kanuka")

		if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
			t.Errorf("Public key file for user%d should still exist", i+1)
		}

		if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
			t.Errorf("Kanuka key file for user%d should still exist", i+1)
		}
	}

	// Performance assertion - removal should be reasonably fast.
	if duration > 5*time.Second {
		t.Logf("Warning: User removal took longer than expected (%v) with %d users", duration, numUsers)
	}
}
