// Package cmd contains testing utilities shared between integration tests.
// This file provides common functions for setting up test environments,
// capturing output, and verifying expected project structures.
package cmd

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	logger "github.com/PolarWolf314/kanuka/internal/logging"
	"github.com/spf13/cobra"
)

// setupTestEnvironment sets up the test environment with temporary directories.
func setupTestEnvironment(t *testing.T, tempDir, tempUserDir, originalWd string, originalUserSettings *configs.UserSettings) {
	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Cleanup function to restore original state
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

	// Override user settings to use temp directory
	configs.UserKanukaSettings = &configs.UserSettings{
		UserKeysPath:    filepath.Join(tempUserDir, "keys"),
		UserConfigsPath: filepath.Join(tempUserDir, "config"),
		Username:        "testuser",
	}
}

// captureOutput captures both stdout and stderr during function execution.
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
		_, err := io.Copy(&buf, stdoutReader)
		if err != nil {
			log.Fatalf("Failed to run copy command: %s", err)
		}
		outputChan <- buf.String()
	}()

	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, stderrReader)
		if err != nil {
			log.Fatalf("Failed to run copy command: %s", err)
		}
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

// initializeProject initializes a kanuka project in the current directory.
func initializeProject(t *testing.T) {
	// Run the init command to set up the project
	_, err := captureOutput(func() error {
		cmd := createInitCommand(nil, nil)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}
}

// createTestCLI creates a complete CLI instance for testing with the specified command and flags.
func createTestCLI(subcommand string, stdout, stderr io.Writer, verboseFlag, debugFlag bool) *cobra.Command {
	// Set global flags for the actual command (needed for the real command implementations)
	verbose = verboseFlag
	debug = debugFlag

	// Initialize the logger with the test flags
	Logger = logger.Logger{
		Verbose: verbose,
		Debug:   debug,
	}

	// Create a fresh root command for this test
	rootCmd := &cobra.Command{
		Use:   "kanuka",
		Short: "Kanuka - A CLI for package management, cloud provisioning, and secrets management.",
		Long: `Kanuka is a powerful command-line tool for managing infrastructure, 
handling project packages using a nix shell environment, and securely storing environment secrets.`,
	}

	// Use the actual SecretsCmd but reset its state
	rootCmd.AddCommand(SecretsCmd)

	// Set output streams
	if stdout != nil {
		rootCmd.SetOut(stdout)
		SecretsCmd.SetOut(stdout)
		// Set output on all subcommands
		encryptCmd.SetOut(stdout)
		decryptCmd.SetOut(stdout)
		createCmd.SetOut(stdout)
		registerCmd.SetOut(stdout)
		removeCmd.SetOut(stdout)
		initCmd.SetOut(stdout)
		purgeCmd.SetOut(stdout)
	}
	if stderr != nil {
		rootCmd.SetErr(stderr)
		SecretsCmd.SetErr(stderr)
		// Set error output on all subcommands
		encryptCmd.SetErr(stderr)
		decryptCmd.SetErr(stderr)
		createCmd.SetErr(stderr)
		registerCmd.SetErr(stderr)
		removeCmd.SetErr(stderr)
		initCmd.SetErr(stderr)
		purgeCmd.SetErr(stderr)
	}

	// Set args to run the specified subcommand
	rootCmd.SetArgs([]string{"secrets", subcommand})

	// Set the flags on the secrets command
	if err := SecretsCmd.PersistentFlags().Set("verbose", fmt.Sprintf("%t", verboseFlag)); err != nil {
		log.Fatalf("Failed to set verbose flag for testing: %s", err)
	}
	if err := SecretsCmd.PersistentFlags().Set("debug", fmt.Sprintf("%t", debugFlag)); err != nil {
		log.Fatalf("Failed to set debug flag for testing: %s", err)
	}

	return rootCmd
}

// createInitCommand creates a command that uses the actual init command.
func createInitCommand(stdout, stderr io.Writer) *cobra.Command {
	return createTestCLI("init", stdout, stderr, false, false)
}

// verifyProjectStructure verifies that the expected project structure was created.
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

// verifyUserKeys verifies that user keys were created in the user directory.
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
