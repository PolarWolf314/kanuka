package register

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsRegisterProjectState contains project state tests for the `kanuka secrets register` command.
func TestSecretsRegisterProjectState(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("RegisterInUninitializedProject", func(t *testing.T) {
		testRegisterInUninitializedProject(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWhenCurrentUserHasNoAccess", func(t *testing.T) {
		testRegisterWhenCurrentUserHasNoAccess(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWhenCurrentUserPrivateKeyMissing", func(t *testing.T) {
		testRegisterWhenCurrentUserPrivateKeyMissing(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWhenTargetUserAlreadyRegistered", func(t *testing.T) {
		testRegisterWhenTargetUserAlreadyRegistered(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterInCorruptedProjectStructure", func(t *testing.T) {
		testRegisterInCorruptedProjectStructure(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithCorruptedKanukaFile", func(t *testing.T) {
		testRegisterWithCorruptedKanukaFile(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithCorruptedPrivateKey", func(t *testing.T) {
		testRegisterWithCorruptedPrivateKey(t, originalWd, originalUserSettings)
	})
}

// testRegisterInUninitializedProject tests error when project is not initialized.
func testRegisterInUninitializedProject(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-uninit-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	// Note: NOT calling shared.InitializeProject here - that's the point
	// Also NOT creating any .kanuka directory - truly uninitialized project

	// Create a temporary public key file outside of any .kanuka directory
	targetUserKeyFile := filepath.Join(tempDir, "targetuser.pub")

	// Create a valid RSA key pair
	privateKeyPath := filepath.Join(tempUserDir, "targetuser.key")
	if err := shared.GenerateRSAKeyPair(privateKeyPath, targetUserKeyFile); err != nil {
		t.Fatalf("Failed to generate RSA key pair: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--file", targetUserKeyFile})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol not found in output: %s", output)
	}

	if !strings.Contains(output, "Kānuka has not been initialized") {
		t.Errorf("Expected 'not initialized' message not found in output: %s", output)
	}

	if !strings.Contains(output, "kanuka secrets init") {
		t.Errorf("Expected init instruction not found in output: %s", output)
	}
}

// testRegisterWhenCurrentUserHasNoAccess tests error when current user's .kanuka file is missing.
func testRegisterWhenCurrentUserHasNoAccess(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-no-access-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project structure but don't create user access
	shared.InitializeProjectStructureOnly(t, tempDir, tempUserDir)

	// Create target user's public key
	targetUser := "targetuser"
	createTestUserKeyPair(t, tempDir, targetUser)
	targetUserKeyFile := filepath.Join(tempDir, ".kanuka", "public_keys", targetUser+".pub")

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--file", targetUserKeyFile})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol not found in output: %s", output)
	}

	if !strings.Contains(output, "Couldn't get your Kānuka key") {
		t.Errorf("Expected 'no access' message not found in output: %s", output)
	}

	if !strings.Contains(output, "Are you sure you have access") {
		t.Errorf("Expected access question not found in output: %s", output)
	}
}

// testRegisterWhenCurrentUserPrivateKeyMissing tests error when current user's private key is missing.
func testRegisterWhenCurrentUserPrivateKeyMissing(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-no-privkey-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Remove the current user's private key
	projectUUID := shared.GetProjectUUID(t)
	privateKeyPath := filepath.Join(tempUserDir, "keys", projectUUID)
	if err := os.Remove(privateKeyPath); err != nil {
		t.Fatalf("Failed to remove private key: %v", err)
	}

	// Create target user's public key
	targetUser := "targetuser"
	createTestUserKeyPair(t, tempDir, targetUser)
	targetUserKeyFile := filepath.Join(tempDir, ".kanuka", "public_keys", targetUser+".pub")

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--file", targetUserKeyFile})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol not found in output: %s", output)
	}

	if !strings.Contains(output, "Couldn't get your private key") {
		t.Errorf("Expected 'no private key' message not found in output: %s", output)
	}

	if !strings.Contains(output, "Are you sure you have access") {
		t.Errorf("Expected access question not found in output: %s", output)
	}
}

// testRegisterWhenTargetUserAlreadyRegistered tests handling re-registration of existing user.
func testRegisterWhenTargetUserAlreadyRegistered(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-already-registered-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create target user's public key
	targetUser := "targetuser"
	targetUserKeyPair := createTestUserKeyPair(t, tempDir, targetUser)
	targetUserKeyFile := filepath.Join(tempDir, ".kanuka", "public_keys", targetUser+".pub")

	// Register the user first time
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--file", targetUserKeyFile})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	// Read the original .kanuka file content
	targetKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", targetUser+".kanuka")
	originalContent, err := os.ReadFile(targetKanukaFile)
	if err != nil {
		t.Fatalf("Failed to read original .kanuka file: %v", err)
	}

	// Register the user again
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--file", targetUserKeyFile})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Second registration failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	// Verify the .kanuka file was updated (should be different due to new encryption)
	newContent, err := os.ReadFile(targetKanukaFile)
	if err != nil {
		t.Errorf("Failed to read new .kanuka file: %v", err)
	}

	// The content should be different because RSA encryption with random padding produces different results
	if string(originalContent) == string(newContent) {
		t.Errorf("Re-registration should produce different encrypted content due to random padding")
	}

	// Verify the target user can still decrypt the symmetric key
	verifyUserCanDecrypt(t, targetUser, targetUserKeyPair.privateKey)
}

// testRegisterInCorruptedProjectStructure tests handling missing directories gracefully.
func testRegisterInCorruptedProjectStructure(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-corrupted-structure-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Remove the secrets directory to corrupt the project structure
	secretsDir := filepath.Join(tempDir, ".kanuka", "secrets")
	if err := os.RemoveAll(secretsDir); err != nil {
		t.Fatalf("Failed to remove secrets directory: %v", err)
	}

	// Create target user's public key
	targetUser := "targetuser"
	createTestUserKeyPair(t, tempDir, targetUser)
	targetUserKeyFile := filepath.Join(tempDir, ".kanuka", "public_keys", targetUser+".pub")

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--file", targetUserKeyFile})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol not found in output: %s", output)
	}

	if !strings.Contains(output, "Couldn't get your Kānuka key") {
		t.Errorf("Expected 'no kanuka key' message not found in output: %s", output)
	}
}

// testRegisterWithCorruptedKanukaFile tests error when current user's .kanuka file is corrupted.
func testRegisterWithCorruptedKanukaFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-corrupted-kanuka-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Corrupt the current user's .kanuka file
	userUUID := shared.GetUserUUID(t)
	kanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", userUUID+".kanuka")
	if err := os.WriteFile(kanukaFile, []byte("corrupted data"), 0600); err != nil {
		t.Fatalf("Failed to corrupt .kanuka file: %v", err)
	}

	// Create target user's public key
	targetUser := "targetuser"
	createTestUserKeyPair(t, tempDir, targetUser)
	targetUserKeyFile := filepath.Join(tempDir, ".kanuka", "public_keys", targetUser+".pub")

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--file", targetUserKeyFile})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol not found in output: %s", output)
	}

	if !strings.Contains(output, "Failed to decrypt your Kānuka key") {
		t.Errorf("Expected decryption failure message not found in output: %s", output)
	}

	if !strings.Contains(output, "Are you sure you have access") {
		t.Errorf("Expected access question not found in output: %s", output)
	}
}

// testRegisterWithCorruptedPrivateKey tests error when current user's private key is corrupted.
func testRegisterWithCorruptedPrivateKey(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-corrupted-privkey-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Corrupt the current user's private key
	projectUUID := shared.GetProjectUUID(t)
	privateKeyPath := filepath.Join(tempUserDir, "keys", projectUUID)
	if err := os.WriteFile(privateKeyPath, []byte("corrupted private key data"), 0600); err != nil {
		t.Fatalf("Failed to corrupt private key: %v", err)
	}

	// Create target user's public key
	targetUser := "targetuser"
	createTestUserKeyPair(t, tempDir, targetUser)
	targetUserKeyFile := filepath.Join(tempDir, ".kanuka", "public_keys", targetUser+".pub")

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--file", targetUserKeyFile})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol not found in output: %s", output)
	}

	if !strings.Contains(output, "Couldn't get your private key") {
		t.Errorf("Expected private key error message not found in output: %s", output)
	}

	if !strings.Contains(output, "Are you sure you have access") {
		t.Errorf("Expected access question not found in output: %s", output)
	}
}
