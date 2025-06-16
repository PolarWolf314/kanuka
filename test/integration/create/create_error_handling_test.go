package create

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsCreateErrorHandling contains error handling tests for the `kanuka secrets create` command.
func TestSecretsCreateErrorHandling(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("ReadOnlyProjectDirectory", func(t *testing.T) {
		testReadOnlyProjectDirectory(t, originalWd, originalUserSettings)
	})

	t.Run("ReadOnlyUserDirectory", func(t *testing.T) {
		testReadOnlyUserDirectory(t, originalWd, originalUserSettings)
	})

	t.Run("InvalidProjectStructure", func(t *testing.T) {
		testInvalidProjectStructure(t, originalWd, originalUserSettings)
	})

	t.Run("PermissionDeniedScenarios", func(t *testing.T) {
		testPermissionDeniedScenarios(t, originalWd, originalUserSettings)
	})
}

// Tests read-only project directory - should fail gracefully.
func testReadOnlyProjectDirectory(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-readonly-project-*")
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

	// Make project directory read-only
	if err := os.Chmod(tempDir, 0555); err != nil {
		t.Fatalf("Failed to make project directory read-only: %v", err)
	}
	defer os.Chmod(tempDir, 0755) // Restore permissions for cleanup

	// Try to create keys - should fail gracefully
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})

	// Should fail with appropriate error
	if err == nil {
		t.Errorf("Expected failure with read-only project directory but got success")
	}

	// Should have meaningful error message
	if !strings.Contains(output, "permission denied") && !strings.Contains(output, "failed") {
		t.Errorf("Expected permission error message, got: %s", output)
	}
}

// Tests read-only user directory - should fail gracefully.
func testReadOnlyUserDirectory(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-readonly-user-*")
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

	// Make user directory read-only
	if err := os.Chmod(tempUserDir, 0555); err != nil {
		t.Fatalf("Failed to make user directory read-only: %v", err)
	}
	defer os.Chmod(tempUserDir, 0755) // Restore permissions for cleanup

	// Try to create keys - should fail gracefully
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})

	// Should fail with appropriate error
	if err == nil {
		t.Errorf("Expected failure with read-only user directory but got success")
	}

	// Should have meaningful error message
	if !strings.Contains(output, "permission denied") && !strings.Contains(output, "failed") {
		t.Errorf("Expected permission error message, got: %s", output)
	}
}

// Tests invalid project structure - should handle corrupted .kanuka directories.
func testInvalidProjectStructure(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	testCases := []struct {
		name        string
		setupFunc   func(string) error
		expectError bool
	}{
		{
			name: "KanukaAsFile",
			setupFunc: func(tempDir string) error {
				return os.WriteFile(filepath.Join(tempDir, ".kanuka"), []byte("not a directory"), 0644)
			},
			expectError: true,
		},
		{
			name: "PublicKeysAsFile",
			setupFunc: func(tempDir string) error {
				kanukaDir := filepath.Join(tempDir, ".kanuka")
				if err := os.MkdirAll(kanukaDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(kanukaDir, "public_keys"), []byte("not a directory"), 0644)
			},
			expectError: true,
		},
		{
			name: "SecretsAsFile",
			setupFunc: func(tempDir string) error {
				kanukaDir := filepath.Join(tempDir, ".kanuka")
				publicKeysDir := filepath.Join(kanukaDir, "public_keys")
				if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(kanukaDir, "secrets"), []byte("not a directory"), 0644)
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "kanuka-test-invalid-*")
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

			// Setup invalid structure
			if err := tc.setupFunc(tempDir); err != nil {
				t.Fatalf("Failed to setup invalid structure: %v", err)
			}

			// Try to create keys
			output, err := shared.CaptureOutput(func() error {
				cmd := shared.CreateTestCLI("create", nil, nil, true, false)
				return cmd.Execute()
			})

			if tc.expectError && err == nil {
				t.Errorf("Expected error for %s but got success", tc.name)
			} else if !tc.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tc.name, err)
			}

			if tc.expectError {
				// Should have meaningful error message
				if !strings.Contains(output, "not been initialized") && !strings.Contains(output, "failed") {
					t.Errorf("Expected error message for %s, got: %s", tc.name, output)
				}
			}
		})
	}
}

// Tests various permission restriction scenarios.
func testPermissionDeniedScenarios(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	testCases := []struct {
		name        string
		setupFunc   func(string, string) error
		expectError bool
	}{
		{
			name: "ReadOnlyKanukaDir",
			setupFunc: func(tempDir, tempUserDir string) error {
				kanukaDir := filepath.Join(tempDir, ".kanuka")
				publicKeysDir := filepath.Join(kanukaDir, "public_keys")
				secretsDir := filepath.Join(kanukaDir, "secrets")
				
				if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
					return err
				}
				if err := os.MkdirAll(secretsDir, 0755); err != nil {
					return err
				}
				
				// Make .kanuka directory read-only
				return os.Chmod(kanukaDir, 0555)
			},
			expectError: true,
		},
		{
			name: "ReadOnlyPublicKeysDir",
			setupFunc: func(tempDir, tempUserDir string) error {
				kanukaDir := filepath.Join(tempDir, ".kanuka")
				publicKeysDir := filepath.Join(kanukaDir, "public_keys")
				secretsDir := filepath.Join(kanukaDir, "secrets")
				
				if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
					return err
				}
				if err := os.MkdirAll(secretsDir, 0755); err != nil {
					return err
				}
				
				// Make public_keys directory read-only
				return os.Chmod(publicKeysDir, 0555)
			},
			expectError: true,
		},
		{
			name: "ReadOnlyKeysDir",
			setupFunc: func(tempDir, tempUserDir string) error {
				keysDir := filepath.Join(tempUserDir, "keys")
				if err := os.MkdirAll(keysDir, 0755); err != nil {
					return err
				}
				
				// Make keys directory read-only
				return os.Chmod(keysDir, 0555)
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "kanuka-test-perms-*")
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

			// Setup permission restriction
			if err := tc.setupFunc(tempDir, tempUserDir); err != nil {
				t.Fatalf("Failed to setup permission restriction: %v", err)
			}

			// Cleanup function to restore permissions
			defer func() {
				// Restore permissions for cleanup
				os.Chmod(filepath.Join(tempDir, ".kanuka"), 0755)
				os.Chmod(filepath.Join(tempDir, ".kanuka", "public_keys"), 0755)
				os.Chmod(filepath.Join(tempDir, ".kanuka", "secrets"), 0755)
				os.Chmod(filepath.Join(tempUserDir, "keys"), 0755)
			}()

			// Try to create keys
			output, err := shared.CaptureOutput(func() error {
				cmd := shared.CreateTestCLI("create", nil, nil, true, false)
				return cmd.Execute()
			})

			if tc.expectError && err == nil {
				t.Errorf("Expected error for %s but got success", tc.name)
			} else if !tc.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tc.name, err)
			}

			if tc.expectError {
				// Should have meaningful error message
				if !strings.Contains(output, "permission denied") && !strings.Contains(output, "failed") {
					t.Errorf("Expected permission error message for %s, got: %s", tc.name, output)
				}
			}
		})
	}
}