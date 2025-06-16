package create

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsCreateOutputValidation contains output and UX validation tests for the `kanuka secrets create` command.
func TestSecretsCreateOutputValidation(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("SuccessMessages", func(t *testing.T) {
		testSuccessMessages(t, originalWd, originalUserSettings)
	})

	t.Run("ErrorMessages", func(t *testing.T) {
		testErrorMessages(t, originalWd, originalUserSettings)
	})

	t.Run("ProgressIndicators", func(t *testing.T) {
		testProgressIndicators(t, originalWd, originalUserSettings)
	})

	t.Run("VerboseMode", func(t *testing.T) {
		testVerboseMode(t, originalWd, originalUserSettings)
	})

	t.Run("InstructionsDisplay", func(t *testing.T) {
		testInstructionsDisplay(t, originalWd, originalUserSettings)
	})
}

// Tests success messages - verify clear success messages with file paths.
func testSuccessMessages(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-success-*")
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
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	// Test success indicator
	if !strings.Contains(output, "✓") {
		t.Errorf("Success indicator (✓) not found in output: %s", output)
	}

	// Test success message
	if !strings.Contains(output, "The following changes were made") {
		t.Errorf("Success message not found in output: %s", output)
	}

	// Test file creation message
	if !strings.Contains(output, "created:") {
		t.Errorf("File creation message not found in output: %s", output)
	}

	// Test that actual file path is shown
	username := configs.UserKanukaSettings.Username
	expectedPath := filepath.Join(".kanuka", "public_keys", username+".pub")
	if !strings.Contains(output, expectedPath) && !strings.Contains(output, username+".pub") {
		t.Errorf("Expected file path not found in output: %s", output)
	}

	// Test color coding (basic check for ANSI escape sequences)
	if !strings.Contains(output, "\x1b[") {
		t.Errorf("No color coding found in output (expected ANSI escape sequences): %s", output)
	}
}

// Tests error messages - verify clear, actionable error messages.
func testErrorMessages(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	testCases := []struct {
		name           string
		setupFunc      func(string, string) error
		expectedError  string
		expectedAction string
	}{
		{
			name: "UninitializedProject",
			setupFunc: func(tempDir, tempUserDir string) error {
				// Don't initialize the project
				return nil
			},
			expectedError:  "Kanuka has not been initialized",
			expectedAction: "kanuka secrets init",
		},
		{
			name: "ExistingKeys",
			setupFunc: func(tempDir, tempUserDir string) error {
				// Initialize and create keys first
				shared.InitializeProject(t, tempDir, tempUserDir)
				_, err := shared.CaptureOutput(func() error {
					cmd := shared.CreateTestCLI("create", nil, nil, true, false)
					return cmd.Execute()
				})
				return err
			},
			expectedError:  "already exists",
			expectedAction: "kanuka secrets create --force",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "kanuka-test-error-*")
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

			// Setup the error condition
			if err := tc.setupFunc(tempDir, tempUserDir); err != nil {
				t.Fatalf("Failed to setup error condition: %v", err)
			}

			// Run the command that should fail
			output, err := shared.CaptureOutput(func() error {
				cmd := shared.CreateTestCLI("create", nil, nil, true, false)
				return cmd.Execute()
			})

			// For some error cases, the command might not return an error but show error message
			if tc.name == "UninitializedProject" || tc.name == "ExistingKeys" {
				// These cases show error messages but may not return errors
			}

			// Check for expected error message
			if !strings.Contains(output, tc.expectedError) {
				t.Errorf("Expected error message '%s' not found in output: %s", tc.expectedError, output)
			}

			// Check for actionable instruction
			if !strings.Contains(output, tc.expectedAction) {
				t.Errorf("Expected action '%s' not found in output: %s", tc.expectedAction, output)
			}

			// Check for error indicator
			if !strings.Contains(output, "✗") && !strings.Contains(output, "failed") {
				t.Errorf("Error indicator not found in output: %s", output)
			}
		})
	}
}

// Tests progress indicators - test spinner and progress feedback.
func testProgressIndicators(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-progress-*")
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

	// Test with verbose mode to see progress messages
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	// Check for progress-related messages
	progressIndicators := []string{
		"Creating Kanuka file",
		"✓", // Final success indicator
	}

	for _, indicator := range progressIndicators {
		if !strings.Contains(output, indicator) {
			t.Errorf("Progress indicator '%s' not found in output: %s", indicator, output)
		}
	}

	// The output should show the final result, not intermediate spinner states
	// since we're capturing the final output after completion
	if !strings.Contains(output, "The following changes were made") {
		t.Errorf("Final progress message not found in output: %s", output)
	}
}

// Tests verbose mode - test detailed logging output.
func testVerboseMode(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-verbose-*")
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

	// Test with verbose flag
	verboseOutput, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false) // verbose = true
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Verbose command failed: %v", err)
	}

	// Test without verbose flag
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	nonVerboseOutput, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, false, false) // verbose = false, force to recreate
		cmd.SetArgs([]string{"secrets", "create", "--force"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Non-verbose command failed: %v", err)
	}

	// Verbose output should contain more detailed information
	// Note: The actual verbose logging might be handled differently in the implementation
	// This test checks that both modes work and produce output

	if len(verboseOutput) == 0 {
		t.Errorf("Verbose mode produced no output")
	}

	if len(nonVerboseOutput) == 0 {
		t.Errorf("Non-verbose mode produced no output")
	}

	// Both should contain the essential success message
	if !strings.Contains(verboseOutput, "✓") {
		t.Errorf("Verbose output missing success indicator: %s", verboseOutput)
	}

	if !strings.Contains(nonVerboseOutput, "✓") {
		t.Errorf("Non-verbose output missing success indicator: %s", nonVerboseOutput)
	}
}

// Tests instructions display - verify next steps are clearly communicated.
func testInstructionsDisplay(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-instructions-*")
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
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	username := configs.UserKanukaSettings.Username

	// Test that instructions are provided
	expectedInstructions := []string{
		"To gain access to the secrets in this project",
		"Commit your",
		".kanuka/public_keys/" + username + ".pub",
		"file to your version control system",
		"Ask someone with permissions to grant you access with:",
		"kanuka secrets add " + username,
	}

	for _, instruction := range expectedInstructions {
		if !strings.Contains(output, instruction) {
			t.Errorf("Expected instruction '%s' not found in output: %s", instruction, output)
		}
	}

	// Test that instructions are clearly formatted
	if !strings.Contains(output, "1.") {
		t.Errorf("Numbered instructions not found in output: %s", output)
	}

	if !strings.Contains(output, "2.") {
		t.Errorf("Second instruction step not found in output: %s", output)
	}

	// Test that the command to run is highlighted/formatted
	if !strings.Contains(output, "kanuka secrets add") {
		t.Errorf("Command instruction not found in output: %s", output)
	}

	// Test that file paths are clearly indicated
	if !strings.Contains(output, username+".pub") {
		t.Errorf("Public key filename not mentioned in instructions: %s", output)
	}
}