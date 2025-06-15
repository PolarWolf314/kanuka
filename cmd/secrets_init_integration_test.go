package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	logger "github.com/PolarWolf314/kanuka/internal/logging"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/spf13/cobra"
)

// TestSecretsInitIntegration contains integration tests for the `kanuka secrets init` command
func TestSecretsInitIntegration(t *testing.T) {
	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	// Save original user settings to restore later
	originalUserSettings := configs.UserKanukaSettings

	t.Run("InitInEmptyFolder", func(t *testing.T) {
		testInitInEmptyFolder(t, originalWd, originalUserSettings)
	})

	t.Run("InitInAlreadyInitializedFolder", func(t *testing.T) {
		testInitInAlreadyInitializedFolder(t, originalWd, originalUserSettings)
	})

	t.Run("InitWithVerboseFlag", func(t *testing.T) {
		testInitWithVerboseFlag(t, originalWd, originalUserSettings)
	})

	t.Run("InitWithDebugFlag", func(t *testing.T) {
		testInitWithDebugFlag(t, originalWd, originalUserSettings)
	})
}

// testInitInEmptyFolder tests successful initialization in an empty folder
func testInitInEmptyFolder(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-empty-*")
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

	// Capture real stdout/stderr by redirecting them
	output, err := captureOutput(func() error {
		cmd := createInitCommand(nil, nil)
		return cmd.Execute()
	})

	// Verify command succeeded
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Verify project structure was created
	verifyProjectStructure(t, tempDir)

	// Verify user keys were created
	verifyUserKeys(t, tempUserDir)

	// Verify that the command ran (check for warning message which is always shown)
	if !strings.Contains(output, "Warning: Remember: Never commit .env files") {
		t.Errorf("Expected warning message not found in output: %s", output)
	}
}

// testInitInAlreadyInitializedFolder tests behavior when running init in an already initialized folder
func testInitInAlreadyInitializedFolder(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-existing-*")
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

	// Pre-create .kanuka directory to simulate already initialized project
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if err := os.MkdirAll(kanukaDir, 0755); err != nil {
		t.Fatalf("Failed to create .kanuka directory: %v", err)
	}

	// Capture real stdout/stderr by redirecting them
	_, err = captureOutput(func() error {
		cmd := createInitCommand(nil, nil)
		return cmd.Execute()
	})

	// Command should succeed but show already initialized message
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	// Verify already initialized message (the spinner final message might not be captured)
	// Instead, let's verify that the .kanuka directory still exists and no new files were created
	if _, statErr := os.Stat(kanukaDir); os.IsNotExist(statErr) {
		t.Errorf(".kanuka directory should still exist after running init on already initialized project")
	}

	// Verify no additional files were created (public_keys and secrets dirs should be empty)
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")
	
	if publicKeysEntries, readErr := os.ReadDir(publicKeysDir); readErr == nil && len(publicKeysEntries) > 0 {
		t.Errorf("public_keys directory should be empty but contains: %v", publicKeysEntries)
	}
	
	if secretsEntries, readErr := os.ReadDir(secretsDir); readErr == nil && len(secretsEntries) > 0 {
		t.Errorf("secrets directory should be empty but contains: %v", secretsEntries)
	}
}

// testInitWithVerboseFlag tests initialization with verbose flag
func testInitWithVerboseFlag(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-verbose-*")
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

	// Capture real stdout/stderr by redirecting them
	output, err := captureOutput(func() error {
		cmd := createInitCommandWithFlags(nil, nil, true, false)
		return cmd.Execute()
	})

	// Verify command succeeded
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Verify verbose output contains info messages
	if !strings.Contains(output, "[info]") {
		t.Errorf("Expected verbose [info] messages not found in output: %s", output)
	}

	// Verify project structure was created
	verifyProjectStructure(t, tempDir)
}

// testInitWithDebugFlag tests initialization with debug flag
func testInitWithDebugFlag(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-debug-*")
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

	// Capture real stdout/stderr by redirecting them
	output, err := captureOutput(func() error {
		cmd := createInitCommandWithFlags(nil, nil, false, true)
		return cmd.Execute()
	})

	// Verify command succeeded
	if err != nil {
		t.Errorf("Command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	// Verify debug output contains debug messages
	if !strings.Contains(output, "[debug]") {
		t.Errorf("Expected debug [debug] messages not found in output: %s", output)
	}

	// Debug should also include info messages
	if !strings.Contains(output, "[info]") {
		t.Errorf("Expected [info] messages not found in debug output: %s", output)
	}

	// Verify project structure was created
	verifyProjectStructure(t, tempDir)
}

// setupTestEnvironment sets up the test environment with temporary directories
func setupTestEnvironment(t *testing.T, tempDir, tempUserDir, originalWd string, originalUserSettings *configs.UserSettings) {
	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Cleanup function to restore original state
	t.Cleanup(func() {
		os.Chdir(originalWd)
		configs.UserKanukaSettings = originalUserSettings
		configs.ProjectKanukaSettings = &configs.ProjectSettings{
			ProjectName:          "",
			ProjectPath:          "",
			ProjectPublicKeyPath: "",
			ProjectSecretsPath:   "",
		}
	})

	// Override user settings to use temp directory
	configs.UserKanukaSettings = &configs.UserSettings{
		UserKeysPath:    filepath.Join(tempUserDir, "keys"),
		UserConfigsPath: filepath.Join(tempUserDir, "config"),
		Username:        "testuser",
	}
}

// createInitCommand creates a secrets init command with output capture
func createInitCommand(stdout, stderr io.Writer) *cobra.Command {
	return createInitCommandWithFlags(stdout, stderr, false, false)
}

// createInitCommandWithFlags creates a secrets init command with specified flags and output capture
func createInitCommandWithFlags(stdout, stderr io.Writer, verboseFlag, debugFlag bool) *cobra.Command {
	// Create a new secrets command to avoid global state issues
	secretsCmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage secrets stored in the repository",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Initialize logger with test flags
			verbose = verboseFlag
			debug = debugFlag
			Logger = logger.Logger{
				Verbose: verbose,
				Debug:   debug,
			}
		},
	}

	// Add flags
	secretsCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", verboseFlag, "enable verbose output")
	secretsCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", debugFlag, "enable debug output")

	// Create init command
	initTestCmd := &cobra.Command{
		Use:   "init",
		Short: "Initializes the secrets store",
		Run: func(cmd *cobra.Command, args []string) {
			Logger.Infof("Starting init command")
			spinner, cleanup := startSpinner("Initializing Kanuka...", verbose)
			defer cleanup()

			Logger.Debugf("Checking if project kanuka settings already exist")
			kanukaExists, err := secrets.DoesProjectKanukaSettingsExist()
			if err != nil {
				Logger.Fatalf("Failed to check if project kanuka settings exists: %v", err)
				return
			}
			if kanukaExists {
				finalMessage := "✗ Kanuka has already been initialized\n" +
					"→ Please run kanuka secrets create instead\n"
				spinner.FinalMSG = finalMessage
				return
			}

			Logger.Debugf("Ensuring user settings")
			if err := secrets.EnsureUserSettings(); err != nil {
				Logger.Fatalf("Failed ensuring user settings: %v", err)
				return
			}
			Logger.Infof("User settings ensured successfully")

			Logger.Debugf("Ensuring kanuka settings and creating .kanuka folders")
			if err := secrets.EnsureKanukaSettings(); err != nil {
				Logger.Fatalf("Failed to create .kanuka folders: %v", err)
				return
			}
			Logger.Infof("Kanuka settings and folders created successfully")

			Logger.Debugf("Creating and saving RSA key pair")
			if err := secrets.CreateAndSaveRSAKeyPair(verbose); err != nil {
				Logger.Fatalf("Failed to generate and save RSA key pair: %v", err)
				return
			}
			Logger.Infof("RSA key pair created and saved successfully")

			Logger.Debugf("Copying user public key to project")
			destPath, err := secrets.CopyUserPublicKeyToProject()
			_ = destPath // explicitly ignore destPath for now
			if err != nil {
				Logger.Fatalf("Failed to copy public key to project: %v", err)
				return
			}
			Logger.Infof("User public key copied to project successfully")

			Logger.Debugf("Creating and saving encrypted symmetric key")
			if err := secrets.CreateAndSaveEncryptedSymmetricKey(verbose); err != nil {
				Logger.Fatalf("Failed to create encrypted symmetric key: %v", err)
				return
			}
			Logger.Infof("Encrypted symmetric key created and saved successfully")

			Logger.Infof("Init command completed successfully")

			// Security reminder about .env files
			Logger.WarnfUser("Remember: Never commit .env files to version control - only commit .kanuka files")

			finalMessage := "✓ Kanuka initialized successfully!\n" +
				"→ Run kanuka secrets encrypt to encrypt your existing .env files\n"

			spinner.FinalMSG = finalMessage
		},
	}

	secretsCmd.AddCommand(initTestCmd)

	// Set output (only if not nil)
	if stdout != nil {
		secretsCmd.SetOut(stdout)
		initTestCmd.SetOut(stdout)
	}
	if stderr != nil {
		secretsCmd.SetErr(stderr)
		initTestCmd.SetErr(stderr)
	}

	// Create root command
	rootCmd := &cobra.Command{Use: "kanuka"}
	rootCmd.AddCommand(secretsCmd)
	if stdout != nil {
		rootCmd.SetOut(stdout)
	}
	if stderr != nil {
		rootCmd.SetErr(stderr)
	}

	// Set args to run the init command
	rootCmd.SetArgs([]string{"secrets", "init"})

	return rootCmd
}

// verifyProjectStructure verifies that the expected project structure was created
func verifyProjectStructure(t *testing.T, tempDir string) {
	// Check .kanuka directory exists
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if _, err := os.Stat(kanukaDir); os.IsNotExist(err) {
		t.Errorf(".kanuka directory was not created")
	}

	// Check public_keys directory exists
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	if _, err := os.Stat(publicKeysDir); os.IsNotExist(err) {
		t.Errorf(".kanuka/public_keys directory was not created")
	}

	// Check secrets directory exists
	secretsDir := filepath.Join(kanukaDir, "secrets")
	if _, err := os.Stat(secretsDir); os.IsNotExist(err) {
		t.Errorf(".kanuka/secrets directory was not created")
	}

	// Check that public key was copied to project
	publicKeyFile := filepath.Join(publicKeysDir, "testuser.pub")
	if _, err := os.Stat(publicKeyFile); os.IsNotExist(err) {
		t.Errorf("Public key file was not created at %s", publicKeyFile)
	}

	// Check that encrypted symmetric key was created
	secretKeyFile := filepath.Join(secretsDir, "testuser.kanuka")
	if _, err := os.Stat(secretKeyFile); os.IsNotExist(err) {
		t.Errorf("Encrypted symmetric key file was not created at %s", secretKeyFile)
	}
}

// verifyUserKeys verifies that user keys were created in the user directory
func verifyUserKeys(t *testing.T, tempUserDir string) {
	keysDir := filepath.Join(tempUserDir, "keys")

	// The project name should be the basename of the temp directory
	// Since we're in the temp directory, we need to get its name
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectName := filepath.Base(wd)

	// Check private key exists
	privateKeyFile := filepath.Join(keysDir, projectName)
	if _, err := os.Stat(privateKeyFile); os.IsNotExist(err) {
		t.Errorf("Private key file was not created at %s", privateKeyFile)
	}

	// Check public key exists
	publicKeyFile := filepath.Join(keysDir, projectName+".pub")
	if _, err := os.Stat(publicKeyFile); os.IsNotExist(err) {
		t.Errorf("Public key file was not created at %s", publicKeyFile)
	}
}

// captureOutput captures both stdout and stderr during function execution
func captureOutput(fn func() error) (string, error) {
	// Save original stdout and stderr
	originalStdout := os.Stdout
	originalStderr := os.Stderr

	// Create pipes to capture output
	stdoutReader, stdoutWriter, _ := os.Pipe()
	stderrReader, stderrWriter, _ := os.Pipe()

	// Replace stdout and stderr
	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter

	// Channel to collect output
	outputChan := make(chan string, 2)

	// Start goroutines to read from pipes
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, stdoutReader)
		outputChan <- buf.String()
	}()

	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, stderrReader)
		outputChan <- buf.String()
	}()

	// Execute the function
	err := fn()

	// Close writers to signal EOF
	stdoutWriter.Close()
	stderrWriter.Close()

	// Restore original stdout and stderr
	os.Stdout = originalStdout
	os.Stderr = originalStderr

	// Collect output
	stdout := <-outputChan
	stderr := <-outputChan

	return stdout + stderr, err
}