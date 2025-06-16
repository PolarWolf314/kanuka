// Package shared contains testing utilities shared between integration tests.
// This file provides common functions for setting up test environments,
// capturing output, and verifying expected project structures.
package shared

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/PolarWolf314/kanuka/cmd"
	"github.com/PolarWolf314/kanuka/internal/configs"
	logger "github.com/PolarWolf314/kanuka/internal/logging"
	"github.com/spf13/cobra"
)

// SetupTestEnvironment sets up the test environment with temporary directories.
func SetupTestEnvironment(t *testing.T, tempDir, tempUserDir, originalWd string, originalUserSettings *configs.UserSettings) {
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

// CaptureOutput captures both stdout and stderr during function execution.
func CaptureOutput(fn func() error) (string, error) {
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

// CreateTestCLI creates a complete CLI instance for testing with the specified command and flags.
func CreateTestCLI(subcommand string, stdout, stderr io.Writer, verboseFlag, debugFlag bool) *cobra.Command {
	// Set global flags for the actual command (needed for the real command implementations)
	cmd.SetVerbose(verboseFlag)
	cmd.SetDebug(debugFlag)

	// Initialize the logger with the test flags
	cmd.SetLogger(logger.Logger{
		Verbose: verboseFlag,
		Debug:   debugFlag,
	})

	// Create a fresh root command for this test
	rootCmd := &cobra.Command{
		Use:   "kanuka",
		Short: "Kanuka - A CLI for package management, cloud provisioning, and secrets management.",
		Long: `Kanuka is a powerful command-line tool for managing infrastructure, 
handling project packages using a nix shell environment, and securely storing environment secrets.`,
	}

	// Use the actual SecretsCmd but reset its state
	rootCmd.AddCommand(cmd.GetSecretsCmd())

	// Set output streams
	if stdout != nil {
		rootCmd.SetOut(stdout)
		cmd.GetSecretsCmd().SetOut(stdout)
		// Set output on all subcommands
		for _, subcmd := range cmd.GetSecretsCmd().Commands() {
			subcmd.SetOut(stdout)
		}
	}
	if stderr != nil {
		rootCmd.SetErr(stderr)
		cmd.GetSecretsCmd().SetErr(stderr)
		// Set error output on all subcommands
		for _, subcmd := range cmd.GetSecretsCmd().Commands() {
			subcmd.SetErr(stderr)
		}
	}

	// Set args to run the specified subcommand
	rootCmd.SetArgs([]string{"secrets", subcommand})

	// Set the flags on the secrets command
	if err := cmd.GetSecretsCmd().PersistentFlags().Set("verbose", fmt.Sprintf("%t", verboseFlag)); err != nil {
		log.Fatalf("Failed to set verbose flag for testing: %s", err)
	}
	if err := cmd.GetSecretsCmd().PersistentFlags().Set("debug", fmt.Sprintf("%t", debugFlag)); err != nil {
		log.Fatalf("Failed to set debug flag for testing: %s", err)
	}

	return rootCmd
}

// VerifyProjectStructure verifies that the expected project structure was created.
func VerifyProjectStructure(t *testing.T, tempDir string) {
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

// VerifyUserKeys verifies that user keys were created in the user directory.
func VerifyUserKeys(t *testing.T, tempUserDir string) {
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

// InitializeProject initializes a project by running the init command first.
func InitializeProject(t *testing.T, tempDir, tempUserDir string) {
	// Run init command to set up the project
	_, err := CaptureOutput(func() error {
		cmd := CreateTestCLI("init", nil, nil, false, false)
		return cmd.Execute()
	})
	
	if err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}
	
	// Verify the project was initialized correctly
	VerifyProjectStructure(t, tempDir)
}