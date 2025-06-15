package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	logger "github.com/PolarWolf314/kanuka/internal/logging"
	"github.com/spf13/cobra"
)

// TestSecretsEncryptIntegration contains integration tests for the `kanuka secrets encrypt` command.
func TestSecretsEncryptIntegration(t *testing.T) {
	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	// Save original user settings to restore later
	originalUserSettings := configs.UserKanukaSettings

	t.Run("EncryptInEmptyFolder", func(t *testing.T) {
		testEncryptInEmptyFolder(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptInInitializedFolderWithNoEnvFiles", func(t *testing.T) {
		testEncryptInInitializedFolderWithNoEnvFiles(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptInInitializedFolderWithOneEnvFile", func(t *testing.T) {
		testEncryptInInitializedFolderWithOneEnvFile(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptInInitializedFolderWithMultipleEnvFiles", func(t *testing.T) {
		testEncryptInInitializedFolderWithMultipleEnvFiles(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptWithoutAccess", func(t *testing.T) {
		testEncryptWithoutAccess(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptFromSubfolderWithOneEnvFile", func(t *testing.T) {
		testEncryptFromSubfolderWithOneEnvFile(t, originalWd, originalUserSettings)
	})

	t.Run("EncryptFromSubfolderWithMultipleEnvFiles", func(t *testing.T) {
		testEncryptFromSubfolderWithMultipleEnvFiles(t, originalWd, originalUserSettings)
	})
}

// testEncryptInEmptyFolder tests encrypt command in an empty folder (should fail).
func testEncryptInEmptyFolder(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-empty-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory for kanuka settings
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Setup test environment
	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Capture output (run in verbose mode to capture final messages)
	output, err := captureOutput(func() error {
		cmd := createEncryptCommandWithFlags(nil, nil, true, false)
		return cmd.Execute()
	})
	// Command should fail because kanuka is not initialized
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	// Verify error message about not being initialized
	if !strings.Contains(output, "Kanuka has not been initialized") {
		t.Errorf("Expected 'not initialized' message not found in output: %s", output)
	}
}

// testEncryptInInitializedFolderWithNoEnvFiles tests encrypt in initialized folder with no .env files.
func testEncryptInInitializedFolderWithNoEnvFiles(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-no-env-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory for kanuka settings
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Setup test environment and initialize project
	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	initializeProject(t)

	// Capture output (run in verbose mode to capture final messages)
	output, err := captureOutput(func() error {
		cmd := createEncryptCommandWithFlags(nil, nil, true, false)
		return cmd.Execute()
	})
	// Command should succeed but report no files found
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	// Verify message about no environment files found
	if !strings.Contains(output, "No environment files found") {
		t.Errorf("Expected 'no environment files found' message not found in output: %s", output)
	}
}

// testEncryptInInitializedFolderWithOneEnvFile tests encrypt with one .env file.
func testEncryptInInitializedFolderWithOneEnvFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-one-env-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory for kanuka settings
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Setup test environment and initialize project
	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	initializeProject(t)

	// Create a .env file
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\nAPI_KEY=secret123\n"
	envPath := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Capture output (run in verbose mode to capture final messages)
	output, err := captureOutput(func() error {
		cmd := createEncryptCommandWithFlags(nil, nil, true, false)
		return cmd.Execute()
	})
	// Command should succeed
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Verify success message
	if !strings.Contains(output, "Environment files encrypted successfully") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	// Verify .env.kanuka file was created
	kanukaPath := envPath + ".kanuka"
	if _, err := os.Stat(kanukaPath); os.IsNotExist(err) {
		t.Errorf(".env.kanuka file was not created at %s", kanukaPath)
	}

	// Verify the encrypted file is not empty and different from original
	kanukaContent, err := os.ReadFile(kanukaPath)
	if err != nil {
		t.Errorf("Failed to read .env.kanuka file: %v", err)
	}
	if len(kanukaContent) == 0 {
		t.Errorf(".env.kanuka file is empty")
	}
	if string(kanukaContent) == envContent {
		t.Errorf(".env.kanuka file content is the same as .env file (not encrypted)")
	}
}

// testEncryptInInitializedFolderWithMultipleEnvFiles tests encrypt with multiple .env files.
func testEncryptInInitializedFolderWithMultipleEnvFiles(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-multi-env-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory for kanuka settings
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Setup test environment and initialize project
	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	initializeProject(t)

	// Create multiple .env files
	envFiles := map[string]string{
		".env":                   "DATABASE_URL=postgres://localhost:5432/mydb\n",
		".env.local":             "API_KEY=secret123\n",
		"config/.env.production": "PROD_API_KEY=prod_secret\n",
		"services/.env.test":     "TEST_DB=test_database\n",
	}

	for filePath, content := range envFiles {
		fullPath := filepath.Join(tempDir, filePath)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create .env file %s: %v", fullPath, err)
		}
	}

	// Capture output (run in verbose mode to capture final messages)
	output, err := captureOutput(func() error {
		cmd := createEncryptCommandWithFlags(nil, nil, true, false)
		return cmd.Execute()
	})
	// Command should succeed
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Verify success message
	if !strings.Contains(output, "Environment files encrypted successfully") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	// Verify all .env.kanuka files were created
	for filePath := range envFiles {
		kanukaPath := filepath.Join(tempDir, filePath+".kanuka")
		if _, err := os.Stat(kanukaPath); os.IsNotExist(err) {
			t.Errorf(".env.kanuka file was not created at %s", kanukaPath)
		}
	}
}

// testEncryptWithoutAccess tests encrypt when user doesn't have access (missing private key).
func testEncryptWithoutAccess(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-no-access-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory for kanuka settings
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Setup test environment and initialize project
	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	initializeProject(t)

	// Create a .env file
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\n"
	envPath := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Remove the user's private key to simulate no access
	projectName := filepath.Base(tempDir)
	privateKeyPath := filepath.Join(tempUserDir, "keys", projectName)
	if err := os.Remove(privateKeyPath); err != nil {
		t.Fatalf("Failed to remove private key: %v", err)
	}

	// Capture output (run in verbose mode to capture final messages)
	output, err := captureOutput(func() error {
		cmd := createEncryptCommandWithFlags(nil, nil, true, false)
		return cmd.Execute()
	})
	// Command should fail
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	// Verify error message about access
	if !strings.Contains(output, "Failed to get your private key file") {
		t.Errorf("Expected access error message not found in output: %s", output)
	}
}

// testEncryptFromSubfolderWithOneEnvFile tests encrypt from subfolder with one .env file.
func testEncryptFromSubfolderWithOneEnvFile(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-subfolder-one-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory for kanuka settings
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Setup test environment and initialize project
	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	initializeProject(t)

	// Create a .env file in root
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\n"
	envPath := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Create a subfolder and change to it
	subDir := filepath.Join(tempDir, "subfolder")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subfolder: %v", err)
	}
	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("Failed to change to subfolder: %v", err)
	}

	// Capture output (run in verbose mode to capture final messages)
	output, err := captureOutput(func() error {
		cmd := createEncryptCommandWithFlags(nil, nil, true, false)
		return cmd.Execute()
	})
	// Command should succeed
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Verify success message
	if !strings.Contains(output, "Environment files encrypted successfully") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	// Verify .env.kanuka file was created in the root
	kanukaPath := envPath + ".kanuka"
	if _, err := os.Stat(kanukaPath); os.IsNotExist(err) {
		t.Errorf(".env.kanuka file was not created at %s", kanukaPath)
	}
}

// testEncryptFromSubfolderWithMultipleEnvFiles tests encrypt from subfolder with multiple .env files.
func testEncryptFromSubfolderWithMultipleEnvFiles(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-subfolder-multi-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary user directory for kanuka settings
	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	// Setup test environment and initialize project
	setupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	initializeProject(t)

	// Create multiple .env files
	envFiles := map[string]string{
		".env":                   "DATABASE_URL=postgres://localhost:5432/mydb\n",
		".env.local":             "API_KEY=secret123\n",
		"config/.env.production": "PROD_API_KEY=prod_secret\n",
		"services/.env.test":     "TEST_DB=test_database\n",
	}

	for filePath, content := range envFiles {
		fullPath := filepath.Join(tempDir, filePath)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create .env file %s: %v", fullPath, err)
		}
	}

	// Create a subfolder and change to it
	subDir := filepath.Join(tempDir, "subfolder")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subfolder: %v", err)
	}
	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("Failed to change to subfolder: %v", err)
	}

	// Capture output (run in verbose mode to capture final messages)
	output, err := captureOutput(func() error {
		cmd := createEncryptCommandWithFlags(nil, nil, true, false)
		return cmd.Execute()
	})
	// Command should succeed
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Verify success message
	if !strings.Contains(output, "Environment files encrypted successfully") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	// Verify all .env.kanuka files were created
	for filePath := range envFiles {
		kanukaPath := filepath.Join(tempDir, filePath+".kanuka")
		if _, err := os.Stat(kanukaPath); os.IsNotExist(err) {
			t.Errorf(".env.kanuka file was not created at %s", kanukaPath)
		}
	}
}

// createEncryptCommand creates a command that uses the actual encrypt command.
func createEncryptCommand(stdout, stderr io.Writer) *cobra.Command {
	return createEncryptCommandWithFlags(stdout, stderr, false, false)
}

// createEncryptCommandWithFlags creates a command that uses the actual encrypt command with specified flags.
func createEncryptCommandWithFlags(stdout, stderr io.Writer, verboseFlag, debugFlag bool) *cobra.Command {
	// Set the global flags for the actual command
	verbose = verboseFlag
	debug = debugFlag

	// Initialize the logger with the test flags
	Logger = logger.Logger{
		Verbose: verbose,
		Debug:   debug,
	}

	// Create a root command that uses the actual SecretsCmd
	rootCmd := &cobra.Command{Use: "kanuka"}
	rootCmd.AddCommand(SecretsCmd)

	// Set output streams
	if stdout != nil {
		rootCmd.SetOut(stdout)
		SecretsCmd.SetOut(stdout)
		encryptCmd.SetOut(stdout)
	}
	if stderr != nil {
		rootCmd.SetErr(stderr)
		SecretsCmd.SetErr(stderr)
		encryptCmd.SetErr(stderr)
	}

	// Set args to run the encrypt command
	rootCmd.SetArgs([]string{"secrets", "encrypt"})

	// Set the flags on the actual command
	SecretsCmd.PersistentFlags().Set("verbose", fmt.Sprintf("%t", verboseFlag))
	SecretsCmd.PersistentFlags().Set("debug", fmt.Sprintf("%t", debugFlag))

	return rootCmd
}
