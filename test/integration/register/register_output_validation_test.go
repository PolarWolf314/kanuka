package register

import (
	"os"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsRegisterOutputValidation contains output validation tests for the `kanuka secrets register` command.
func TestSecretsRegisterOutputValidation(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("RegisterSuccessMessageFormat", func(t *testing.T) {
		testRegisterSuccessMessageFormat(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterErrorMessageFormat", func(t *testing.T) {
		testRegisterErrorMessageFormat(t, originalWd, originalUserSettings)
	})
}

// testRegisterSuccessMessageFormat tests verify success message format and content.
func testRegisterSuccessMessageFormat(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-success-format-*")
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

	// Create a target user's public key
	targetUser := "successuser"
	createTestUserKeyPair(t, tempDir, targetUser)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, false, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", targetUser})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Verify success symbol
	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success symbol '✓' not found in output: %s", output)
	}

	// Verify username is mentioned
	if !strings.Contains(output, targetUser) {
		t.Errorf("Expected username '%s' not found in output: %s", targetUser, output)
	}

	// Verify success message content
	if !strings.Contains(output, "registered successfully") {
		t.Errorf("Expected 'registered successfully' message not found in output: %s", output)
	}

	// Verify informational arrow
	if !strings.Contains(output, "→") {
		t.Errorf("Expected informational arrow '→' not found in output: %s", output)
	}

	// Verify access message
	if !strings.Contains(output, "access to decrypt") {
		t.Errorf("Expected access message not found in output: %s", output)
	}

	// Verify file extension is mentioned
	if !strings.Contains(output, ".pub") {
		t.Errorf("Expected '.pub' file extension not found in output: %s", output)
	}
}

// testRegisterErrorMessageFormat tests verify error message format and content.
func testRegisterErrorMessageFormat(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-error-format-*")
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

	// Test error when user doesn't exist
	nonExistentUser := "nonexistentuser"

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, false, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", nonExistentUser})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	// Verify error symbol
	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol '✗' not found in output: %s", output)
	}

	// Verify username is mentioned
	if !strings.Contains(output, nonExistentUser) {
		t.Errorf("Expected username '%s' not found in output: %s", nonExistentUser, output)
	}

	// Verify error message content
	if !strings.Contains(output, "not found") {
		t.Errorf("Expected 'not found' message not found in output: %s", output)
	}

	// Verify helpful instruction
	if !strings.Contains(output, "kanuka secrets create") {
		t.Errorf("Expected helpful instruction 'kanuka secrets create' not found in output: %s", output)
	}
}
