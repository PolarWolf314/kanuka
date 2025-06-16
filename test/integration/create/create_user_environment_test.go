package create

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsCreateUserEnvironment contains user settings and environment tests for the `kanuka secrets create` command.
func TestSecretsCreateUserEnvironment(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("UsernameDetection", func(t *testing.T) {
		testUsernameDetection(t, originalWd, originalUserSettings)
	})

	t.Run("CustomDataDirectories", func(t *testing.T) {
		testCustomDataDirectories(t, originalWd, originalUserSettings)
	})

	t.Run("UserDirectoryPermissions", func(t *testing.T) {
		testUserDirectoryPermissions(t, originalWd, originalUserSettings)
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		testConcurrentAccess(t, originalWd, originalUserSettings)
	})
}

// Tests username detection with different system usernames.
func testUsernameDetection(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	testCases := []struct {
		name     string
		username string
		valid    bool
	}{
		{"SimpleUsername", "testuser", true},
		{"UsernameWithNumbers", "user123", true},
		{"UsernameWithUnderscore", "test_user", true},
		{"UsernameWithDash", "test-user", true},
		{"EmptyUsername", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.username == "" {
				// Skip empty username test as it would require special setup
				t.Skip("Empty username test requires special environment setup")
				return
			}

			tempDir, err := os.MkdirTemp("", "kanuka-test-username-*")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
			if err != nil {
				t.Fatalf("Failed to create temp user directory: %v", err)
			}
			defer os.RemoveAll(tempUserDir)

			// Setup test environment with custom username
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			t.Cleanup(func() {
				if err := os.Chdir(originalWd); err != nil {
					t.Fatalf("Failed to change to original directory: %v", err)
				}
				configs.UserKanukaSettings = originalUserSettings
				configs.ProjectKanukaSettings = &configs.ProjectSettings{
					ProjectName:          "",
					ProjectPath:          "",
					ProjectPublicKeyPath: "",
					ProjectSecretsPath:   "",
				}
			})

			// Override user settings with custom username
			configs.UserKanukaSettings = &configs.UserSettings{
				UserKeysPath:    filepath.Join(tempUserDir, "keys"),
				UserConfigsPath: filepath.Join(tempUserDir, "config"),
				Username:        tc.username,
			}

			shared.InitializeProject(t, tempDir, tempUserDir)

			output, err := shared.CaptureOutput(func() error {
				cmd := shared.CreateTestCLI("create", nil, nil, true, false)
				return cmd.Execute()
			})

			if tc.valid && err != nil {
				t.Errorf("Expected success for username %s but got error: %v", tc.username, err)
				t.Errorf("Output: %s", output)
			} else if !tc.valid && err == nil {
				t.Errorf("Expected failure for username %s but got success", tc.username)
			}

			if tc.valid {
				// Verify files were created with correct username
				publicKeyPath := filepath.Join(tempDir, ".kanuka", "public_keys", tc.username+".pub")
				if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
					t.Errorf("Public key not created for username %s", tc.username)
				}

				// Verify output contains username
				if !strings.Contains(output, tc.username) {
					t.Errorf("Output doesn't contain username %s: %s", tc.username, output)
				}
			}
		})
	}
}

// Tests custom data directories with custom XDG_DATA_HOME settings.
func testCustomDataDirectories(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-custom-data-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create custom data directory
	customDataDir, err := os.MkdirTemp("", "kanuka-custom-data-*")
	if err != nil {
		t.Fatalf("Failed to create custom data directory: %v", err)
	}
	defer os.RemoveAll(customDataDir)

	shared.SetupTestEnvironment(t, tempDir, customDataDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, customDataDir)

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed with custom data directory: %v", err)
	}

	// Verify keys were created in custom directory
	projectName := filepath.Base(tempDir)
	privateKeyPath := filepath.Join(customDataDir, "keys", projectName)
	publicKeyPath := filepath.Join(customDataDir, "keys", projectName+".pub")

	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		t.Errorf("Private key not created in custom data directory")
	}
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		t.Errorf("Public key not created in custom data directory")
	}

	// Verify directory structure was created
	keysDir := filepath.Join(customDataDir, "keys")
	configDir := filepath.Join(customDataDir, "config")

	if _, err := os.Stat(keysDir); os.IsNotExist(err) {
		t.Errorf("Keys directory not created in custom data directory")
	}
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Errorf("Config directory not created in custom data directory")
	}
}

// Tests when user directories have restricted permissions.
func testUserDirectoryPermissions(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-user-perms-*")
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

	// Create keys directory with restricted permissions
	keysDir := filepath.Join(tempUserDir, "keys")
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		t.Fatalf("Failed to create keys directory: %v", err)
	}

	// Test with directory that has correct permissions (should work)
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed with proper permissions: %v", err)
	}

	// Verify keys were created
	projectName := filepath.Base(tempDir)
	privateKeyPath := filepath.Join(keysDir, projectName)
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		t.Errorf("Private key not created with proper permissions")
	}

	// Test directory permissions are secure
	keysDirInfo, err := os.Stat(keysDir)
	if err != nil {
		t.Errorf("Failed to stat keys directory: %v", err)
	} else {
		mode := keysDirInfo.Mode()
		// On Unix systems, check that only owner has access
		if runtime.GOOS != "windows" && mode&0077 != 0 {
			t.Errorf("Keys directory has insecure permissions: %o", mode)
		}
	}

	// Check private key permissions
	privateKeyInfo, err := os.Stat(privateKeyPath)
	if err != nil {
		t.Errorf("Failed to stat private key: %v", err)
	} else {
		mode := privateKeyInfo.Mode()
		// On Unix systems, check that only owner can read/write
		if runtime.GOOS != "windows" && mode&0077 != 0 {
			t.Errorf("Private key has insecure permissions: %o", mode)
		}
	}
}

// Tests behavior when multiple processes access user directories.
func testConcurrentAccess(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// This test simulates concurrent access by creating multiple projects
	// that share the same user directory
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-concurrent-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Create multiple projects concurrently
	numProjects := 3
	results := make(chan error, numProjects)

	for i := 0; i < numProjects; i++ {
		go func(projectNum int) {
			tempDir, err := os.MkdirTemp("", "kanuka-test-concurrent-*")
			if err != nil {
				results <- err
				return
			}
			defer os.RemoveAll(tempDir)

			// Each goroutine needs its own working directory context
			originalWd, err := os.Getwd()
			if err != nil {
				results <- err
				return
			}

			if err := os.Chdir(tempDir); err != nil {
				results <- err
				return
			}
			defer os.Chdir(originalWd)

			// Setup user settings for this project
			configs.UserKanukaSettings = &configs.UserSettings{
				UserKeysPath:    filepath.Join(tempUserDir, "keys"),
				UserConfigsPath: filepath.Join(tempUserDir, "config"),
				Username:        "testuser",
			}

			// Initialize project
			_, err = shared.CaptureOutput(func() error {
				cmd := shared.CreateTestCLI("init", nil, nil, false, false)
				return cmd.Execute()
			})
			if err != nil {
				results <- err
				return
			}

			// Create keys
			_, err = shared.CaptureOutput(func() error {
				cmd := shared.CreateTestCLI("create", nil, nil, true, false)
				return cmd.Execute()
			})
			results <- err
		}(i)
	}

	// Wait for all goroutines to complete
	var errors []error
	for i := 0; i < numProjects; i++ {
		if err := <-results; err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		t.Errorf("Concurrent access failed with errors: %v", errors)
	}

	// Verify that all projects created their keys
	keysDir := filepath.Join(tempUserDir, "keys")
	entries, err := os.ReadDir(keysDir)
	if err != nil {
		t.Errorf("Failed to read keys directory: %v", err)
	}

	// Should have at least numProjects * 2 files (private + public key for each project)
	if len(entries) < numProjects*2 {
		t.Errorf("Expected at least %d key files, got %d", numProjects*2, len(entries))
	}

	// Verify each project has unique keys
	privateKeys := make(map[string]bool)
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".pub") {
			// This is a private key
			keyPath := filepath.Join(keysDir, entry.Name())
			keyData, err := os.ReadFile(keyPath)
			if err != nil {
				t.Errorf("Failed to read key file %s: %v", entry.Name(), err)
				continue
			}
			keyContent := string(keyData)
			if privateKeys[keyContent] {
				t.Errorf("Duplicate private key found: %s", entry.Name())
			}
			privateKeys[keyContent] = true
		}
	}
}
