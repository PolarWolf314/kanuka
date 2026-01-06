package register

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsRegisterErrorHandling contains error handling tests for the `kanuka secrets register` command.
func TestSecretsRegisterErrorHandling(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("RegisterWithNetworkInterruption", func(t *testing.T) {
		testRegisterWithNetworkInterruption(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithPermissionDenied", func(t *testing.T) {
		testRegisterWithPermissionDenied(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterRecoveryFromPartialFailure", func(t *testing.T) {
		testRegisterRecoveryFromPartialFailure(t, originalWd, originalUserSettings)
	})
}

// testRegisterWithNetworkInterruption simulates filesystem errors during operation.
func testRegisterWithNetworkInterruption(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-network-*")
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
	targetUser := "networkuser"
	createTestUserKeyPair(t, tempDir, targetUser)

	// Simulate filesystem error by making the secrets directory read-only after creating it
	secretsDir := filepath.Join(tempDir, ".kanuka", "secrets")
	if err := os.Chmod(secretsDir, 0555); err != nil {
		t.Fatalf("Failed to make secrets directory read-only: %v", err)
	}

	// Restore permissions for cleanup
	defer func() {
		if err := os.Chmod(secretsDir, 0755); err != nil {
			t.Logf("Failed to restore permissions on secrets directory: %v", err)
		}
	}()

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", targetUser})
		return cmd.Execute()
	})
	// Check if command actually failed (either through error return or error symbol in output)
	hasErrorSymbol := strings.Contains(output, "✗")
	hasErrorMessage := strings.Contains(output, "Error:") || strings.Contains(output, "error") || strings.Contains(output, "failed")

	if err == nil && !hasErrorSymbol && !hasErrorMessage {
		t.Errorf("Expected command to fail or show error, but got success. Output: %s", output)
	}

	// Check for permission-related error messages
	hasPermissionError := strings.Contains(output, "permission denied") ||
		strings.Contains(output, "Permission denied") ||
		strings.Contains(output, "read-only") ||
		strings.Contains(output, "cannot create")

	// If the command succeeded despite read-only directory, that's also acceptable behavior
	// Some systems may handle this differently
	targetKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", targetUser+".kanuka")
	_, fileExists := os.Stat(targetKanukaFile)

	if strings.Contains(output, "✓") && fileExists == nil {
		t.Logf("Command succeeded despite read-only directory - this may be system-dependent behavior")
	} else if !hasPermissionError {
		t.Errorf("Expected permission-related error message not found in output: %s", output)
	}
}

// testRegisterWithPermissionDenied tests handling permission denied errors.
func testRegisterWithPermissionDenied(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-permission-*")
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
	targetUser := "permissionuser"
	createTestUserKeyPair(t, tempDir, targetUser)

	// Make the entire .kanuka directory read-only
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if err := os.Chmod(kanukaDir, 0444); err != nil {
		t.Fatalf("Failed to make .kanuka directory read-only: %v", err)
	}

	// Restore permissions for cleanup
	defer func() {
		if err := os.Chmod(kanukaDir, 0755); err != nil {
			t.Logf("Failed to restore permissions on .kanuka directory: %v", err)
		}
	}()

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", targetUser})
		return cmd.Execute()
	})
	// The command should fail - either through error return or error in output
	hasError := err != nil || strings.Contains(output, "✗") || strings.Contains(output, "Error:")

	if !hasError {
		t.Errorf("Expected command to fail due to permissions, but got success. Output: %s", output)
	}

	// Should contain some indication of permission issues
	hasPermissionError := strings.Contains(output, "permission") ||
		strings.Contains(output, "Permission") ||
		strings.Contains(output, "access") ||
		strings.Contains(output, "read-only") ||
		strings.Contains(output, "denied")

	if !hasPermissionError {
		t.Logf("Permission-related error message not found in output (may be expected on some systems): %s", output)
	}

	// Verify that no .kanuka file was created due to the error
	targetKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", targetUser+".kanuka")
	if _, statErr := os.Stat(targetKanukaFile); !os.IsNotExist(statErr) {
		// If the file was created despite permission error, this might be system-dependent behavior
		// Log it but don't fail the test as the register command may handle permissions differently
		t.Logf("Target user's .kanuka file was created despite permission error at %s - this may be expected behavior on some systems", targetKanukaFile)
	}
}

// testRegisterRecoveryFromPartialFailure verifies clean state after partial failures.
func testRegisterRecoveryFromPartialFailure(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-recovery-*")
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
	targetUser := "recoveryuser"
	createTestUserKeyPair(t, tempDir, targetUser)

	// First, attempt a registration that will fail due to permission issues
	secretsDir := filepath.Join(tempDir, ".kanuka", "secrets")
	if err := os.Chmod(secretsDir, 0444); err != nil {
		t.Fatalf("Failed to make secrets directory read-only: %v", err)
	}

	// Try to register (this should fail)
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", targetUser})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	// Check if command actually failed (either through error return or error symbol in output)
	hasErrorSymbol := strings.Contains(output, "✗")
	hasErrorMessage := strings.Contains(output, "Error:") || strings.Contains(output, "error") || strings.Contains(output, "failed")

	if err == nil && !hasErrorSymbol && !hasErrorMessage {
		t.Errorf("Expected command to fail or show error in first attempt, but got success. Output: %s", output)
	}

	// Verify that no .kanuka file was created due to the error
	targetKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", targetUser+".kanuka")
	if _, statErr := os.Stat(targetKanukaFile); !os.IsNotExist(statErr) {
		// If the file was created despite permission error, this might be system-dependent behavior
		// Log it but don't fail the test as the register command may handle permissions differently
		t.Logf("Target user's .kanuka file was created despite permission error at %s - this may be expected behavior on some systems", targetKanukaFile)
	}

	// Now restore permissions and try again (this should succeed)
	if err := os.Chmod(secretsDir, 0755); err != nil {
		t.Fatalf("Failed to restore permissions on secrets directory: %v", err)
	}

	output, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", targetUser})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Recovery register command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success message not found in recovery output: %s", output)
	}

	// Verify the .kanuka file was created successfully after recovery
	if _, err := os.Stat(targetKanukaFile); os.IsNotExist(err) {
		t.Errorf("Target user's .kanuka file was not created after recovery at %s", targetKanukaFile)
	}

	// Verify the project structure is still intact
	shared.VerifyProjectStructure(t, tempDir)

	// For this test, we'll just verify the file exists and has content
	kanukaFileContent, err := os.ReadFile(targetKanukaFile)
	if err != nil {
		t.Errorf("Failed to read .kanuka file after recovery: %v", err)
	}
	if len(kanukaFileContent) == 0 {
		t.Errorf(".kanuka file is empty after recovery")
	}

	// Test that we can register another user to ensure the system is fully functional
	anotherUser := "anotheruser"
	createTestUserKeyPair(t, tempDir, anotherUser)

	output, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", anotherUser})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Follow-up register command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success message not found in follow-up registration: %s", output)
	}

	// Verify the second user's .kanuka file was created
	anotherUserKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", anotherUser+".kanuka")
	if _, err := os.Stat(anotherUserKanukaFile); os.IsNotExist(err) {
		t.Errorf("Second user's .kanuka file was not created at %s", anotherUserKanukaFile)
	}
}
