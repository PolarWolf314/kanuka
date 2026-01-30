package ci_init_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/workflows"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestCIInitBasic tests the ci-init command in various scenarios.
func TestCIInitBasic(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	t.Run("CIInitNotInitialized", func(t *testing.T) {
		testCIInitNotInitialized(t, originalWd, originalUserSettings)
	})

	t.Run("CIInitAlreadyConfigured", func(t *testing.T) {
		testCIInitAlreadyConfigured(t, originalWd, originalUserSettings)
	})

	t.Run("IsCIUserRegisteredFunction", func(t *testing.T) {
		testIsCIUserRegisteredFunction(t, originalWd, originalUserSettings)
	})
}

// testCIInitNotInitialized tests that ci-init fails when project is not initialized.
func testCIInitNotInitialized(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-ci-init-notinit-*")
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

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("ci-init", []string{}, nil, nil, false, false)
		return cmd.Execute()
	})

	// Command should not return an error (expected errors are handled internally).
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	// Check output contains expected error message.
	if !strings.Contains(output, "has not been initialized") {
		t.Errorf("Expected 'not initialized' error, got: %s", output)
	}
}

// testCIInitAlreadyConfigured tests that ci-init fails when CI is already set up.
// Note: In a non-TTY test environment, the TTY check happens first, so we test
// the IsCIUserRegistered function directly instead of through the CLI.
func testCIInitAlreadyConfigured(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-ci-init-already-*")
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

	// Initialize project first.
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Add CI user to project config manually to simulate already configured.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}

	ciUUID := "ci-test-uuid-1234-5678-abcdefghijkl"
	if projectConfig.Users == nil {
		projectConfig.Users = make(map[string]string)
	}
	projectConfig.Users[ciUUID] = workflows.CIUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// Clear the cached config so IsCIUserRegistered picks up the changes.
	configs.GlobalProjectConfig = nil

	// Since we don't have TTY in test environment, test the underlying function directly.
	registered, err := workflows.IsCIUserRegistered()
	if err != nil {
		t.Fatalf("IsCIUserRegistered failed: %v", err)
	}
	if !registered {
		t.Error("Expected CI user to be registered (already configured)")
	}

	// Also verify CLI shows TTY error (since we can't mock TTY).
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("ci-init", []string{}, nil, nil, false, false)
		return cmd.Execute()
	})

	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	// In non-TTY environment, we expect the TTY error.
	if !strings.Contains(output, "interactive terminal") {
		t.Errorf("Expected 'interactive terminal' error in non-TTY env, got: %s", output)
	}
}

// testIsCIUserRegisteredFunction tests the IsCIUserRegistered helper function.
func testIsCIUserRegisteredFunction(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-ci-check-*")
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

	// Initialize project.
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Test that CI user is not registered initially.
	registered, err := workflows.IsCIUserRegistered()
	if err != nil {
		t.Fatalf("IsCIUserRegistered failed: %v", err)
	}
	if registered {
		t.Error("Expected CI user to NOT be registered initially")
	}

	// Add CI user to project config.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}

	ciUUID := "ci-test-uuid-1234-5678-abcdefghijkl"
	if projectConfig.Users == nil {
		projectConfig.Users = make(map[string]string)
	}
	projectConfig.Users[ciUUID] = workflows.CIUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// Clear the cached config.
	configs.GlobalProjectConfig = nil

	// Test that CI user is now registered.
	registered, err = workflows.IsCIUserRegistered()
	if err != nil {
		t.Fatalf("IsCIUserRegistered failed: %v", err)
	}
	if !registered {
		t.Error("Expected CI user to be registered after adding to config")
	}
}

// TestCIInitWorkflowExists tests that workflow file existence is detected correctly.
func TestCIInitWorkflowExists(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	tempDir, err := os.MkdirTemp("", "kanuka-test-ci-init-workflow-*")
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

	// Create workflow file manually.
	workflowDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		t.Fatalf("Failed to create workflow directory: %v", err)
	}

	workflowPath := filepath.Join(workflowDir, "kanuka-decrypt.yml")
	// #nosec G306 -- Test workflow file, not sensitive data.
	if err := os.WriteFile(workflowPath, []byte("existing workflow"), 0644); err != nil {
		t.Fatalf("Failed to create workflow file: %v", err)
	}

	// Verify workflow file exists.
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		t.Errorf("Expected workflow file to exist at %s", workflowPath)
	}

	// Verify path matches the constant.
	expectedPath := filepath.Join(tempDir, workflows.CIWorkflowPath)
	if workflowPath != expectedPath {
		t.Errorf("Workflow path mismatch: got %s, expected %s", workflowPath, expectedPath)
	}
}

// TestCIUserEmailConstant verifies the CI user email constant is correct.
func TestCIUserEmailConstant(t *testing.T) {
	expected := "41898282+github-actions[bot]@users.noreply.github.com"
	if workflows.CIUserEmail != expected {
		t.Errorf("CIUserEmail = %q, want %q", workflows.CIUserEmail, expected)
	}
}

// TestCIWorkflowPathConstant verifies the workflow path constant is correct.
func TestCIWorkflowPathConstant(t *testing.T) {
	expected := ".github/workflows/kanuka-decrypt.yml"
	if workflows.CIWorkflowPath != expected {
		t.Errorf("CIWorkflowPath = %q, want %q", workflows.CIWorkflowPath, expected)
	}
}
