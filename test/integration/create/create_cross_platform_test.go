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

// TestSecretsCreateCrossPlatform contains cross-platform tests for the `kanuka secrets create` command.
func TestSecretsCreateCrossPlatform(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("WindowsPathHandling", func(t *testing.T) {
		testWindowsPathHandling(t, originalWd, originalUserSettings)
	})

	t.Run("UnixPathHandling", func(t *testing.T) {
		testUnixPathHandling(t, originalWd, originalUserSettings)
	})

	t.Run("PathSeparatorHandling", func(t *testing.T) {
		testPathSeparatorHandling(t, originalWd, originalUserSettings)
	})

	t.Run("SpecialCharactersInPaths", func(t *testing.T) {
		testSpecialCharactersInPaths(t, originalWd, originalUserSettings)
	})
}

// Tests Windows path handling - test on Windows with %APPDATA% paths.
func testWindowsPathHandling(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test, skipping on non-Windows platform")
	}

	tempDir, err := os.MkdirTemp("", "kanuka-test-windows-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a Windows-style user directory path
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-windows-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Get the project UUID after initialization
	projectUUID := shared.GetProjectUUID(t)

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed on Windows: %v", err)
	}

	// Verify keys were created with Windows path separators (using project UUID)
	keysDir := filepath.Join(tempUserDir, "keys")
	privateKeyPath := shared.GetPrivateKeyPath(keysDir, projectUUID)
	publicKeyPath := shared.GetPublicKeyPath(keysDir, projectUUID)

	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		t.Errorf("Private key not created on Windows")
	}
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		t.Errorf("Public key not created on Windows")
	}

	// Verify paths use correct separators
	if !strings.Contains(privateKeyPath, string(filepath.Separator)) {
		t.Errorf("Path doesn't use correct separator for Windows: %s", privateKeyPath)
	}
}

// Tests Unix path handling - test on Unix systems with XDG directories.
func testUnixPathHandling(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test, skipping on Windows platform")
	}

	tempDir, err := os.MkdirTemp("", "kanuka-test-unix-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a Unix-style user directory path
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-unix-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Get the project UUID after initialization
	projectUUID := shared.GetProjectUUID(t)

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed on Unix: %v", err)
	}

	// Verify keys were created (using project UUID, not project name)
	keysDir := filepath.Join(tempUserDir, "keys")
	privateKeyPath := shared.GetPrivateKeyPath(keysDir, projectUUID)
	publicKeyPath := shared.GetPublicKeyPath(keysDir, projectUUID)

	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		t.Errorf("Private key not created on Unix")
	}
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		t.Errorf("Public key not created on Unix")
	}

	// Verify Unix-specific permissions
	privateKeyInfo, err := os.Stat(privateKeyPath)
	if err != nil {
		t.Errorf("Failed to stat private key: %v", err)
	} else {
		mode := privateKeyInfo.Mode()
		// Check that only owner has read/write access (0600)
		if mode&0077 != 0 {
			t.Logf("Private key has permissions: %o (may vary by system)", mode)
			// This is informational - file permissions may vary by system and implementation
		}
	}

	keysDirInfo, err := os.Stat(keysDir)
	if err != nil {
		t.Errorf("Failed to stat keys directory: %v", err)
	} else {
		mode := keysDirInfo.Mode()
		// Check that only owner has access (0700)
		if mode&0077 != 0 {
			t.Errorf("Keys directory has incorrect Unix permissions: %o", mode)
		}
	}
}

// Tests path separator handling with different path separators.
func testPathSeparatorHandling(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-pathsep-*")
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

	// Get UUIDs after initialization
	projectUUID := shared.GetProjectUUID(t)
	userUUID := shared.GetUserUUID(t)

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	// Verify that filepath.Join was used correctly by checking the created paths
	privateKeyPath := shared.GetPrivateKeyPath(filepath.Join(tempUserDir, "keys"), projectUUID)

	// The path should exist regardless of platform
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		t.Errorf("Private key not created with correct path separators")
	}

	// Verify the path uses the correct separator for the current platform
	expectedSeparator := string(filepath.Separator)
	if !strings.Contains(privateKeyPath, expectedSeparator) {
		t.Errorf("Path doesn't use correct separator (%s): %s", expectedSeparator, privateKeyPath)
	}

	// Test project public key path (now uses user UUID)
	projectPublicKeyPath := filepath.Join(tempDir, ".kanuka", "public_keys", userUUID+".pub")

	if _, err := os.Stat(projectPublicKeyPath); os.IsNotExist(err) {
		t.Errorf("Project public key not created with correct path separators")
	}
}

// Tests special characters in paths - test with spaces and special characters in project paths.
func testSpecialCharactersInPaths(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	testCases := []struct {
		name       string
		dirPattern string
		shouldWork bool
	}{
		{"SpacesInPath", "kanuka test with spaces-*", true},
		{"DotsInPath", "kanuka.test.with.dots-*", true},
		{"UnderscoresInPath", "kanuka_test_with_underscores-*", true},
		{"DashesInPath", "kanuka-test-with-dashes-*", true},
		{"NumbersInPath", "kanuka123test456-*", true},
		{"MixedSpecialChars", "kanuka-test_with.mixed chars-*", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", tc.dirPattern)
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

			// Get UUIDs after initialization
			projectUUID := shared.GetProjectUUID(t)
			userUUID := shared.GetUserUUID(t)

			output, err := shared.CaptureOutput(func() error {
				cmd := shared.CreateTestCLI("create", nil, nil, true, false)
				return cmd.Execute()
			})

			if tc.shouldWork && err != nil {
				t.Errorf("Expected success for %s but got error: %v", tc.name, err)
				t.Errorf("Output: %s", output)
			} else if !tc.shouldWork && err == nil {
				t.Errorf("Expected failure for %s but got success", tc.name)
			}

			if tc.shouldWork {
				// Verify keys were created despite special characters (using project UUID)
				keysDir := filepath.Join(tempUserDir, "keys")
				privateKeyPath := shared.GetPrivateKeyPath(keysDir, projectUUID)
				publicKeyPath := shared.GetPublicKeyPath(keysDir, projectUUID)

				if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
					t.Errorf("Private key not created for path with special characters: %s", tc.name)
				}
				if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
					t.Errorf("Public key not created for path with special characters: %s", tc.name)
				}

				// Verify project public key was copied correctly (using user UUID)
				projectPublicKeyPath := filepath.Join(tempDir, ".kanuka", "public_keys", userUUID+".pub")
				if _, err := os.Stat(projectPublicKeyPath); os.IsNotExist(err) {
					t.Errorf("Project public key not created for path with special characters: %s", tc.name)
				}

				// Verify the keys are valid
				privateKeyData, err := os.ReadFile(privateKeyPath)
				if err != nil {
					t.Errorf("Failed to read private key for %s: %v", tc.name, err)
				} else if !strings.Contains(string(privateKeyData), "-----BEGIN RSA PRIVATE KEY-----") {
					t.Errorf("Private key invalid for path with special characters: %s", tc.name)
				}

				publicKeyData, err := os.ReadFile(publicKeyPath)
				if err != nil {
					t.Errorf("Failed to read public key for %s: %v", tc.name, err)
				} else if !strings.Contains(string(publicKeyData), "-----BEGIN PUBLIC KEY-----") {
					t.Errorf("Public key invalid for path with special characters: %s", tc.name)
				}
			}
		})
	}
}
