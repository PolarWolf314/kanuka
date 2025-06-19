// Package shared contains testing utilities shared between integration tests.
// This file provides common functions for setting up test environments,
// capturing output, and verifying expected project structures.
package shared

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
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

	// Reset global state before creating command to avoid shared state
	cmd.ResetGlobalState()

	// Use the actual SecretsCmd but with reset state
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

	// Check that encrypted symmetric key was created (only for init command, not create)
	// The create command only creates user keys, not the project symmetric key
	secretKeyFile := filepath.Join(secretsDir, "testuser.kanuka")
	if _, err := os.Stat(secretKeyFile); os.IsNotExist(err) {
		// This is expected for create command - only init creates the symmetric key
		t.Logf("Encrypted symmetric key file was not created at %s", secretKeyFile)
	}
}

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

	VerifyProjectStructure(t, tempDir)
}

// InitializeProjectStructureOnly initializes just the project structure without creating user keys.
func InitializeProjectStructureOnly(t *testing.T, tempDir, tempUserDir string) {
	// Create .kanuka directory structure manually
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
		t.Fatalf("Failed to create public keys directory: %v", err)
	}
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		t.Fatalf("Failed to create secrets directory: %v", err)
	}
}

// RegisterTestUser registers a test user with the project.
func RegisterTestUser(t *testing.T, username string) {
	// Reset command state
	cmd.ResetGlobalState()

	// Create and register user
	secretsCmd := cmd.GetSecretsCmd()
	secretsCmd.SetArgs([]string{"register", "--user", username})

	err := secretsCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to register test user %s: %v", username, err)
	}
}

// GetFileMode returns the file mode of a file or directory.
func GetFileMode(path string) (os.FileMode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Mode(), nil
}

// GenerateRSAKeyPair creates a new RSA key pair and saves them to disk.
// This is a utility function for tests that need to create RSA key pairs.
func GenerateRSAKeyPair(privatePath string, publicPath string) error {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create private key PEM
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	// Save private key to file
	privateKeyFile, err := os.Create(privatePath)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer privateKeyFile.Close()

	if err := pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
		return fmt.Errorf("failed to encode private key: %w", err)
	}

	// Create public key PEM
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	// Save public key to file
	publicKeyFile, err := os.Create(publicPath)
	if err != nil {
		return fmt.Errorf("failed to create public key file: %w", err)
	}
	defer publicKeyFile.Close()

	if err := pem.Encode(publicKeyFile, publicKeyPEM); err != nil {
		return fmt.Errorf("failed to encode public key: %w", err)
	}

	return nil
}
