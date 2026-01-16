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

	t.Run("RegisterWithInvalidFileExtension", func(t *testing.T) {
		testRegisterWithInvalidFileExtension(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithEmptyPubkeyText", func(t *testing.T) {
		testRegisterWithEmptyPubkeyText(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithEmptyUsername", func(t *testing.T) {
		testRegisterWithEmptyUsername(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithInvalidEmailFormat", func(t *testing.T) {
		testRegisterWithSpecialCharactersInUsername(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithNonEmailString", func(t *testing.T) {
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
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
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
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
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
	targetUserEmail := "invaliduser@example.com"
	targetUserUUID := "invalid-user-uuid-1234"

	// Add the user to project config so the email lookup succeeds
	addUserToProjectConfig(t, targetUserUUID, targetUserEmail)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", invalidPubkeyText, "--user", targetUserEmail})
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
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
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

	targetUserEmail := "emptyuser@example.com"

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", "", "--user", targetUserEmail})
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
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
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

// testRegisterWithSpecialCharactersInUsername tests that invalid email formats are rejected.
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

	// Test with invalid email format (not a valid email address)
	invalidEmail := "user_with-special_chars"

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", invalidEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol not found in output: %s", output)
	}

	if !strings.Contains(output, "Invalid email format") {
		t.Errorf("Expected 'Invalid email format' message not found in output: %s", output)
	}
}

// testRegisterWithVeryLongUsername tests that invalid email formats (very long strings) are rejected.
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

	// Test with a long string that is not a valid email format
	invalidEmail := "verylongusernamethatisreasonablebutpushesthelimitsofwhatmightbevalid"

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", invalidEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol not found in output: %s", output)
	}

	if !strings.Contains(output, "Invalid email format") {
		t.Errorf("Expected 'Invalid email format' message not found in output: %s", output)
	}
}
