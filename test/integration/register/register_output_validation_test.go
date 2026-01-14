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

	t.Run("RegisterErrorMessageFormat", func(t *testing.T) {
		testRegisterErrorMessageFormat(t, originalWd, originalUserSettings)
	})
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

	// Test error when user email is invalid format
	invalidEmail := "notanemail"

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", invalidEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	// Verify error symbol
	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol '✗' not found in output: %s", output)
	}

	// Verify email is mentioned
	if !strings.Contains(output, invalidEmail) {
		t.Errorf("Expected email '%s' not found in output: %s", invalidEmail, output)
	}

	// Verify error message content - now checks for invalid email format
	if !strings.Contains(output, "Invalid email format") {
		t.Errorf("Expected 'Invalid email format' message not found in output: %s", output)
	}

	// Verify helpful instruction
	if !strings.Contains(output, "valid email address") {
		t.Errorf("Expected helpful instruction about valid email not found in output: %s", output)
	}
}
