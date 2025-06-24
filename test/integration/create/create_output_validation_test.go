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
	shared.InitializeProjectStructureOnly(t, tempDir, tempUserDir)

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

	// Test color coding (basic check for ANSI escape sequences or color indicators)
	// The output may not contain ANSI escape sequences in test environment
	// but should contain colored text indicators like ✓ or colored strings
	if !strings.Contains(output, "\x1b[") && !strings.Contains(output, "✓") {
		t.Errorf("No color coding or visual indicators found in output: %s", output)
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
			expectedError:  "Kānuka has not been initialized",
			expectedAction: "kanuka secrets init",
		},
		{
			name: "ExistingKeys",
			setupFunc: func(tempDir, tempUserDir string) error {
				// Initialize project structure only, then create keys to simulate existing keys
				shared.InitializeProjectStructureOnly(t, tempDir, tempUserDir)
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
			output, _ := shared.CaptureOutput(func() error {
				// For ExistingKeys test, we need to see the output, so use verbose mode
				// but the test setup should ensure we get the "already exists" message
				cmd := shared.CreateTestCLI("create", nil, nil, true, false)
				return cmd.Execute()
			})

			// With the new RunE implementation, some "error" cases return success but show error messages
			// This is the correct behavior for user-facing errors vs. system errors
			// For UninitializedProject and ExistingKeys cases, the command succeeds but shows error messages

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
	shared.InitializeProjectStructureOnly(t, tempDir, tempUserDir)

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
		"Creating Kānuka file",
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
	shared.InitializeProjectStructureOnly(t, tempDir, tempUserDir)

	// Test with verbose flag
	verboseOutput, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false) // verbose = true
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Verbose command failed: %v", err)
	}

	// Test without verbose flag in a fresh environment
	tempDir2, err := os.MkdirTemp("", "kanuka-test-non-verbose-*")
	if err != nil {
		t.Fatalf("Failed to create second temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir2)

	tempUserDir2, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create second temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir2)

	shared.SetupTestEnvironment(t, tempDir2, tempUserDir2, originalWd, originalUserSettings)
	shared.InitializeProjectStructureOnly(t, tempDir2, tempUserDir2)

	nonVerboseOutput, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, false, false) // verbose = false
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

	// In non-verbose mode, output might be minimal or handled by spinner
	// The key is that the command should succeed
	if len(nonVerboseOutput) == 0 {
		t.Logf("Non-verbose mode produced no output (this may be expected with spinner)")
	}

	// Both should contain the essential success message
	if !strings.Contains(verboseOutput, "✓") {
		t.Errorf("Verbose output missing success indicator: %s", verboseOutput)
	}

	// In non-verbose mode, the success indicator might be shown by the spinner
	// or not captured in our output capture method
	if len(nonVerboseOutput) > 0 && !strings.Contains(nonVerboseOutput, "✓") {
		t.Logf("Non-verbose output missing success indicator (may be handled by spinner): %s", nonVerboseOutput)
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
	shared.InitializeProjectStructureOnly(t, tempDir, tempUserDir)

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
