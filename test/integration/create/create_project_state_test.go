package create

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsCreateProjectState contains project state edge case tests for the `kanuka secrets create` command.
func TestSecretsCreateProjectState(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("MultipleProjectSupport", func(t *testing.T) {
		testMultipleProjectSupport(t, originalWd, originalUserSettings)
	})

	t.Run("ProjectNameHandling", func(t *testing.T) {
		testProjectNameHandling(t, originalWd, originalUserSettings)
	})

	t.Run("ExistingProjectStructure", func(t *testing.T) {
		testExistingProjectStructure(t, originalWd, originalUserSettings)
	})

	t.Run("CorruptedProjectState", func(t *testing.T) {
		testCorruptedProjectState(t, originalWd, originalUserSettings)
	})
}

// Tests multiple project support - create keys for different projects, verify isolation.
func testMultipleProjectSupport(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Create first project
	tempDir1, err := os.MkdirTemp("", "kanuka-test-project1-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory 1: %v", err)
	}
	defer os.RemoveAll(tempDir1)

	// Create second project
	tempDir2, err := os.MkdirTemp("", "kanuka-test-project2-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory 2: %v", err)
	}
	defer os.RemoveAll(tempDir2)

	// Setup and initialize first project
	shared.SetupTestEnvironment(t, tempDir1, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir1, tempUserDir)

	// Create keys for first project
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Failed to create keys for project 1: %v", err)
	}

	project1UUID := shared.GetProjectUUID(t)
	keysDir := filepath.Join(tempUserDir, "keys")
	project1PrivateKey := shared.GetPrivateKeyPath(keysDir, project1UUID)
	project1PublicKey := shared.GetPublicKeyPath(keysDir, project1UUID)

	// Verify first project keys exist
	if _, err := os.Stat(project1PrivateKey); os.IsNotExist(err) {
		t.Errorf("Project 1 private key not created")
	}
	if _, err := os.Stat(project1PublicKey); os.IsNotExist(err) {
		t.Errorf("Project 1 public key not created")
	}

	// Setup and initialize second project
	shared.SetupTestEnvironment(t, tempDir2, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir2, tempUserDir)

	// Create keys for second project
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Failed to create keys for project 2: %v", err)
	}

	project2UUID := shared.GetProjectUUID(t)
	project2PrivateKey := shared.GetPrivateKeyPath(keysDir, project2UUID)
	project2PublicKey := shared.GetPublicKeyPath(keysDir, project2UUID)

	// Verify second project keys exist
	if _, err := os.Stat(project2PrivateKey); os.IsNotExist(err) {
		t.Errorf("Project 2 private key not created")
	}
	if _, err := os.Stat(project2PublicKey); os.IsNotExist(err) {
		t.Errorf("Project 2 public key not created")
	}

	// Verify keys are different (isolation)
	key1Data, err := os.ReadFile(project1PrivateKey)
	if err != nil {
		t.Errorf("Failed to read project 1 private key: %v", err)
	}
	key2Data, err := os.ReadFile(project2PrivateKey)
	if err != nil {
		t.Errorf("Failed to read project 2 private key: %v", err)
	}

	if string(key1Data) == string(key2Data) {
		t.Errorf("Project keys are not isolated - same key generated for different projects")
	}
}

// Tests project name handling with various project directory names and structures.
func testProjectNameHandling(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	testCases := []struct {
		name       string
		dirPattern string
		shouldWork bool
	}{
		{"SimpleProjectName", "simple-project-*", true},
		{"ProjectWithSpaces", "project with spaces-*", true},
		{"ProjectWithDots", "project.with.dots-*", true},
		{"ProjectWithUnderscores", "project_with_underscores-*", true},
		{"ProjectWithNumbers", "project123-*", true},
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

			_, err = shared.CaptureOutput(func() error {
				cmd := shared.CreateTestCLI("create", nil, nil, true, false)
				return cmd.Execute()
			})

			if tc.shouldWork && err != nil {
				t.Errorf("Expected success for %s but got error: %v", tc.name, err)
			} else if !tc.shouldWork && err == nil {
				t.Errorf("Expected failure for %s but got success", tc.name)
			}

			if tc.shouldWork {
				projectUUID := shared.GetProjectUUID(t)
				privateKeyPath := shared.GetPrivateKeyPath(filepath.Join(tempUserDir, "keys"), projectUUID)
				if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
					t.Errorf("Private key not created for project %s", tc.name)
				}
			}
		})
	}
}

// Tests when .kanuka directories already exist.
func testExistingProjectStructure(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-existing-*")
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

	// Create .kanuka structure manually before init
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
		t.Fatalf("Failed to create public keys directory: %v", err)
	}
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		t.Fatalf("Failed to create secrets directory: %v", err)
	}

	// Add some existing files
	existingFile := filepath.Join(publicKeysDir, "existing.pub")
	if err := os.WriteFile(existingFile, []byte("existing key"), 0600); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	// Since .kanuka directory already exists, init will say "already initialized".
	// We need to create a project config manually to simulate a valid existing project.
	configs.ProjectKanukaSettings = &configs.ProjectSettings{
		ProjectName:          filepath.Base(tempDir),
		ProjectPath:          tempDir,
		ProjectPublicKeyPath: publicKeysDir,
		ProjectSecretsPath:   secretsDir,
	}

	projectConfig := &configs.ProjectConfig{
		Project: configs.Project{
			UUID: shared.TestProjectUUID,
			Name: filepath.Base(tempDir),
		},
		Users:   make(map[string]string),
		Devices: make(map[string]configs.DeviceConfig),
	}
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to create project config: %v", err)
	}

	// Create keys using create command (since project already has config, this should work)
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "create", "--force"})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Create failed with existing structure: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Verify existing file is still there
	if _, err := os.Stat(existingFile); os.IsNotExist(err) {
		t.Errorf("Existing file was removed")
	}

	// Verify new keys were created
	userUUID := shared.GetUserUUID(t)
	newPublicKey := filepath.Join(publicKeysDir, userUUID+".pub")
	if _, err := os.Stat(newPublicKey); os.IsNotExist(err) {
		t.Errorf("New public key was not created")
	}
}

// Tests behavior with malformed .kanuka directory.
func testCorruptedProjectState(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-corrupted-*")
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

	// Create .kanuka as a file instead of directory (corrupted state)
	kanukaPath := filepath.Join(tempDir, ".kanuka")
	if err := os.WriteFile(kanukaPath, []byte("corrupted"), 0600); err != nil {
		t.Fatalf("Failed to create corrupted .kanuka file: %v", err)
	}

	// Try to create - should handle corruption gracefully
	output, _ := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})

	// The create command should detect that the project is not properly initialized
	// and show an appropriate error message
	if !strings.Contains(output, "not been initialized") {
		t.Errorf("Expected 'not been initialized' message for corrupted state, got: %s", output)
	}

	// The command should suggest running init instead
	if !strings.Contains(output, "kanuka secrets init") {
		t.Errorf("Expected suggestion to run init command, got: %s", output)
	}

	// Verify the corrupted file still exists (wasn't overwritten)
	if _, err := os.Stat(kanukaPath); os.IsNotExist(err) {
		t.Errorf("Corrupted .kanuka file was removed when it should have been left alone")
	}
}
