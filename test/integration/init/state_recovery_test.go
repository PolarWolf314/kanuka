package init_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsInitStateRecovery contains state recovery tests for the `kanuka secrets init` command.
func TestSecretsInitStateRecovery(t *testing.T) {
	// Save original working directory and settings
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	// Category 5: Corrupted/Invalid State Recovery
	t.Run("InitWithPartialKanukaDirectory", func(t *testing.T) {
		testInitWithPartialKanukaDirectory(t, originalWd, originalUserSettings)
	})

	t.Run("InitAfterPartialFailure", func(t *testing.T) {
		testInitAfterPartialFailure(t, originalWd, originalUserSettings)
	})

	// Category 12: Recovery and Cleanup Scenarios
	t.Run("InitIdempotencyAfterFailure", func(t *testing.T) {
		testInitIdempotencyAfterFailure(t, originalWd, originalUserSettings)
	})

	t.Run("InitCleanupAfterUserKeyFailure", func(t *testing.T) {
		testInitCleanupAfterUserKeyFailure(t, originalWd, originalUserSettings)
	})
}

// Category 5: Corrupted/Invalid State Recovery.
func testInitWithPartialKanukaDirectory(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-partial-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Create partial .kanuka directory structure (missing some subdirectories)
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if err := os.Mkdir(kanukaDir, 0755); err != nil {
		t.Fatalf("Failed to create .kanuka directory: %v", err)
	}

	// Create only one of the required subdirectories
	secretsDir := filepath.Join(kanukaDir, "secrets")
	if err := os.Mkdir(secretsDir, 0755); err != nil {
		t.Fatalf("Failed to create secrets directory: %v", err)
	}
	// Deliberately omit public_keys directory

	// Capture output - should detect existing .kanuka and report already initialized
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})
	// Command should detect existing .kanuka directory
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Should contain message about already being initialized
	if !strings.Contains(output, "already been initialized") {
		t.Errorf("Expected message about already initialized, got: %s", output)
	}
}

func testInitAfterPartialFailure(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-after-failure-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Simulate partial failure by creating .kanuka directory but making it read-only
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if err := os.Mkdir(kanukaDir, 0755); err != nil {
		t.Fatalf("Failed to create .kanuka directory: %v", err)
	}

	// First attempt should detect existing .kanuka
	output1, err1 := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	// Should report already initialized
	if err1 != nil {
		t.Errorf("First init attempt failed unexpectedly: %v", err1)
	}
	if !strings.Contains(output1, "already been initialized") {
		t.Errorf("Expected already initialized message, got: %s", output1)
	}

	// Remove the .kanuka directory to simulate cleanup
	if err := os.RemoveAll(kanukaDir); err != nil {
		t.Fatalf("Failed to remove .kanuka directory: %v", err)
	}

	// Second attempt should succeed
	output2, err2 := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	if err2 != nil {
		t.Errorf("Second init attempt failed: %v", err2)
		t.Errorf("Output: %s", output2)
	}

	// Should contain success message
	if !strings.Contains(output2, "initialized successfully") {
		t.Errorf("Expected success message, got: %s", output2)
	}

	// Verify project structure was created
	shared.VerifyProjectStructure(t, tempDir)
}

// Category 12: Recovery and Cleanup Scenarios.
func testInitIdempotencyAfterFailure(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-idempotency-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// First init should succeed
	output1, err1 := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	if err1 != nil {
		t.Errorf("First init failed: %v", err1)
		t.Errorf("Output: %s", output1)
	}

	// Second init should detect existing initialization
	output2, err2 := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	if err2 != nil {
		t.Errorf("Second init failed: %v", err2)
	}

	// Should contain message about already being initialized
	if !strings.Contains(output2, "already been initialized") {
		t.Errorf("Expected already initialized message, got: %s", output2)
	}

	// Project structure should still be intact
	shared.VerifyProjectStructure(t, tempDir)
}

func testInitCleanupAfterUserKeyFailure(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-cleanup-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Create a file where the keys directory should be to cause failure
	keysPath := filepath.Join(tempUserDir, "keys")
	if err := os.WriteFile(keysPath, []byte("blocking file"), 0600); err != nil {
		t.Fatalf("Failed to create blocking file: %v", err)
	}

	// Capture output and expect failure
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, true, false)
		return cmd.Execute()
	})

	// Command should fail
	if err == nil {
		t.Errorf("Expected command to fail due to blocked keys directory, but it succeeded")
		t.Errorf("Output: %s", output)
	}

	// Should contain error message
	if !strings.Contains(output, "failed") {
		t.Errorf("Expected error message, got: %s", output)
	}

	// .kanuka directory should not exist (cleanup should have occurred)
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if _, err := os.Stat(kanukaDir); !os.IsNotExist(err) {
		t.Errorf("Expected .kanuka directory to not exist after failure, but it does")
	}
}
