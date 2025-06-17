package register

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsRegisterInputValidation contains input validation tests for the `kanuka secrets register` command.
func TestSecretsRegisterInputValidation(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("RegisterWithNoFlags", func(t *testing.T) {
		testRegisterWithNoFlags(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithPubkeyButNoUser", func(t *testing.T) {
		testRegisterWithPubkeyButNoUser(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithInvalidPubkeyFormat", func(t *testing.T) {
		testRegisterWithInvalidPubkeyFormat(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithNonExistentFile", func(t *testing.T) {
		testRegisterWithNonExistentFile(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithInvalidFileExtension", func(t *testing.T) {
		testRegisterWithInvalidFileExtension(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithEmptyPubkeyText", func(t *testing.T) {
		testRegisterWithEmptyPubkeyText(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithEmptyUsername", func(t *testing.T) {
		testRegisterWithEmptyUsername(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithSpecialCharactersInUsername", func(t *testing.T) {
		testRegisterWithSpecialCharactersInUsername(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithVeryLongUsername", func(t *testing.T) {
		testRegisterWithVeryLongUsername(t, originalWd, originalUserSettings)
	})
}

// testRegisterWithNoFlags tests error when no flags are provided.
func testRegisterWithNoFlags(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-noflags-*")
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

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol not found in output: %s", output)
	}

	if !strings.Contains(output, "--user") && !strings.Contains(output, "--file") && !strings.Contains(output, "--pubkey") {
		t.Errorf("Expected flag options not found in output: %s", output)
	}

	if !strings.Contains(output, "must be specified") {
		t.Errorf("Expected 'must be specified' message not found in output: %s", output)
	}

	if !strings.Contains(output, "kanuka secrets register --help") {
		t.Errorf("Expected help instruction not found in output: %s", output)
	}
}

// testRegisterWithPubkeyButNoUser tests error when --pubkey is provided without --user.
func testRegisterWithPubkeyButNoUser(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-pubkey-nouser-*")
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

	// Generate a valid public key text
	keyPair := generateTestKeyPair(t)
	pubkeyText := convertPublicKeyToPEM(t, keyPair.publicKey)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, false, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pubkeyText})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol not found in output: %s", output)
	}

	if !strings.Contains(output, "--pubkey") && !strings.Contains(output, "--user") {
		t.Errorf("Expected flag references not found in output: %s", output)
	}

	if !strings.Contains(output, "required") {
		t.Errorf("Expected 'required' message not found in output: %s", output)
	}
}

// testRegisterWithInvalidPubkeyFormat tests error with malformed public key content.
func testRegisterWithInvalidPubkeyFormat(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-invalid-pubkey-*")
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

	invalidPubkeyText := "this-is-not-a-valid-public-key"
	targetUser := "invaliduser"

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, false, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", invalidPubkeyText, "--user", targetUser})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol not found in output: %s", output)
	}

	if !strings.Contains(output, "Invalid public key format") {
		t.Errorf("Expected invalid format message not found in output: %s", output)
	}
}

// testRegisterWithNonExistentFile tests error when --file points to non-existent file.
func testRegisterWithNonExistentFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-nonexistent-*")
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

	nonExistentFile := filepath.Join(tempUserDir, "nonexistent.pub")

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, false, false)
		cmd.SetArgs([]string{"secrets", "register", "--file", nonExistentFile})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol not found in output: %s", output)
	}

	if !strings.Contains(output, "could not be loaded") || !strings.Contains(output, nonExistentFile) {
		t.Errorf("Expected file not found message not found in output: %s", output)
	}
}

// testRegisterWithInvalidFileExtension tests error when --file doesn't end with .pub.
func testRegisterWithInvalidFileExtension(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-invalid-ext-*")
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

	// Create a file with wrong extension
	invalidFile := filepath.Join(tempUserDir, "key.txt")
	if err := os.WriteFile(invalidFile, []byte("some content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, false, false)
		cmd.SetArgs([]string{"secrets", "register", "--file", invalidFile})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol not found in output: %s", output)
	}

	if !strings.Contains(output, "not a valid path to a public key file") {
		t.Errorf("Expected invalid file extension message not found in output: %s", output)
	}
}

// testRegisterWithEmptyPubkeyText tests error when --pubkey is empty string.
func testRegisterWithEmptyPubkeyText(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-empty-pubkey-*")
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

	targetUser := "emptyuser"

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, false, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", "", "--user", targetUser})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol not found in output: %s", output)
	}

	if !strings.Contains(output, "Invalid public key format") {
		t.Errorf("Expected invalid format message not found in output: %s", output)
	}
}

// testRegisterWithEmptyUsername tests error when --user is empty string.
func testRegisterWithEmptyUsername(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-empty-user-*")
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

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, false, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", ""})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol not found in output: %s", output)
	}

	if !strings.Contains(output, "must be specified") {
		t.Errorf("Expected 'must be specified' message not found in output: %s", output)
	}
}

// testRegisterWithSpecialCharactersInUsername tests handling usernames with valid special characters.
func testRegisterWithSpecialCharactersInUsername(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-special-chars-*")
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

	// Test with underscores and hyphens (commonly valid in usernames)
	targetUser := "user_with-special_chars"
	targetUserKeyPair := createTestUserKeyPair(t, tempDir, targetUser)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, false, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", targetUser})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	// Verify the .kanuka file was created for the target user
	targetKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", targetUser+".kanuka")
	if _, err := os.Stat(targetKanukaFile); os.IsNotExist(err) {
		t.Errorf("Target user's .kanuka file was not created at %s", targetKanukaFile)
	}

	// Verify the target user can actually decrypt the symmetric key
	verifyUserCanDecrypt(t, targetUser, targetUserKeyPair.privateKey)
}

// testRegisterWithVeryLongUsername tests handling very long usernames (within limits).
func testRegisterWithVeryLongUsername(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-long-user-*")
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

	// Create a long but reasonable username (64 characters)
	targetUser := "verylongusernamethatisreasonablebutpushesthelimitsofwhatmightbevalid"
	targetUserKeyPair := createTestUserKeyPair(t, tempDir, targetUser)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, false, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", targetUser})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	// Verify the .kanuka file was created for the target user
	targetKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", targetUser+".kanuka")
	if _, err := os.Stat(targetKanukaFile); os.IsNotExist(err) {
		t.Errorf("Target user's .kanuka file was not created at %s", targetKanukaFile)
	}

	// Verify the target user can actually decrypt the symmetric key
	verifyUserCanDecrypt(t, targetUser, targetUserKeyPair.privateKey)
}
