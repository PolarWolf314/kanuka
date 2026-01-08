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

// TestUserUUID is a fixed UUID used for testing purposes.
const TestUserUUID = "test-user-uuid-1234-5678-abcdefghijkl"

// TestUser2UUID is a second fixed UUID used for multi-user testing purposes.
const TestUser2UUID = "test-user-2-uuid-5678-1234-abcdefghijkl"

// TestProjectUUID is a fixed UUID used for testing purposes.
const TestProjectUUID = "test-proj-uuid-1234-5678-abcdefghijkl"

// TestUserEmail is a fixed email used for testing purposes.
const TestUserEmail = "testuser@example.com"

// TestUser2Email is a second fixed email used for multi-user testing purposes.
const TestUser2Email = "testuser2@example.com"

// SetupTestEnvironment sets up the test environment with temporary directories.
func SetupTestEnvironment(t *testing.T, tempDir, tempUserDir, originalWd string, originalUserSettings *configs.UserSettings) {
	SetupTestEnvironmentWithUUID(t, tempDir, tempUserDir, originalWd, originalUserSettings, TestUserUUID, "testuser", "testuser@example.com")
}

// SetupTestEnvironmentWithUUID sets up the test environment with a specific user UUID.
// This is useful for multi-user tests where each user needs a different UUID.
func SetupTestEnvironmentWithUUID(t *testing.T, tempDir, tempUserDir, originalWd string, originalUserSettings *configs.UserSettings, userUUID, username, email string) {
	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create user config directory and set up user config with UUID
	userConfigsPath := filepath.Join(tempUserDir, "config")
	if err := os.MkdirAll(userConfigsPath, 0755); err != nil {
		t.Fatalf("Failed to create user config directory: %v", err)
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
		configs.GlobalUserConfig = nil
		configs.GlobalProjectConfig = nil
	})

	// Override user settings to use temp directory
	configs.UserKanukaSettings = &configs.UserSettings{
		UserKeysPath:    filepath.Join(tempUserDir, "keys"),
		UserConfigsPath: userConfigsPath,
		Username:        username,
	}

	// Create user config with the specified UUID
	userConfig := &configs.UserConfig{
		User: configs.User{
			UUID:  userUUID,
			Email: email,
		},
		Projects: make(map[string]configs.UserProjectEntry),
	}
	if err := configs.SaveUserConfig(userConfig); err != nil {
		t.Fatalf("Failed to save user config: %v", err)
	}
}

// SetupTestEnvironmentWithoutUserConfig sets up the test environment without creating user config.
// This is useful for permission tests where the user directory may be read-only.
func SetupTestEnvironmentWithoutUserConfig(t *testing.T, tempDir, tempUserDir, originalWd string, originalUserSettings *configs.UserSettings) {
	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	userConfigsPath := filepath.Join(tempUserDir, "config")

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
		configs.GlobalUserConfig = nil
		configs.GlobalProjectConfig = nil
	})

	// Override user settings to use temp directory
	configs.UserKanukaSettings = &configs.UserSettings{
		UserKeysPath:    filepath.Join(tempUserDir, "keys"),
		UserConfigsPath: userConfigsPath,
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
// By default, passes --yes flag to avoid interactive prompts in tests.
func CreateTestCLI(subcommand string, stdout, stderr io.Writer, verboseFlag, debugFlag bool) *cobra.Command {
	// Default to non-interactive mode for init command.
	extraArgs := []string{}
	if subcommand == "init" {
		extraArgs = append(extraArgs, "--yes")
	}
	return CreateTestCLIWithArgs(subcommand, extraArgs, stdout, stderr, verboseFlag, debugFlag)
}

// CreateTestCLIWithArgs creates a CLI instance for testing with additional command arguments.
func CreateTestCLIWithArgs(subcommand string, extraArgs []string, stdout, stderr io.Writer, verboseFlag, debugFlag bool) *cobra.Command {
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

	// Build args: secrets <subcommand> [extraArgs...]
	args := []string{"secrets", subcommand}
	args = append(args, extraArgs...)
	rootCmd.SetArgs(args)

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

	// Check that public key was copied to project (using test user UUID)
	publicKeyFile := filepath.Join(publicKeysDir, TestUserUUID+".pub")
	if _, err := os.Stat(publicKeyFile); os.IsNotExist(err) {
		t.Errorf("Public key file was not created at %s", publicKeyFile)
	}

	// Check that encrypted symmetric key was created (only for init command, not create)
	// The create command only creates user keys, not the project symmetric key
	secretKeyFile := filepath.Join(secretsDir, TestUserUUID+".kanuka")
	if _, err := os.Stat(secretKeyFile); os.IsNotExist(err) {
		// This is expected for create command - only init creates the symmetric key
		t.Logf("Encrypted symmetric key file was not created at %s", secretKeyFile)
	}
}

func VerifyUserKeys(t *testing.T, tempUserDir string) {
	keysDir := filepath.Join(tempUserDir, "keys")

	// Load project config to get the project UUID
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectUUID := projectConfig.Project.UUID
	if projectUUID == "" {
		// Fallback to test project UUID if not set
		projectUUID = TestProjectUUID
	}

	// Check key directory exists (named by project UUID)
	keyDir := filepath.Join(keysDir, projectUUID)
	if _, err := os.Stat(keyDir); os.IsNotExist(err) {
		t.Errorf("Key directory was not created at %s", keyDir)
	}

	// Check private key exists inside key directory
	privateKeyFile := filepath.Join(keyDir, "privkey")
	if _, err := os.Stat(privateKeyFile); os.IsNotExist(err) {
		t.Errorf("Private key file was not created at %s", privateKeyFile)
	}

	// Check public key exists inside key directory
	publicKeyFile := filepath.Join(keyDir, "pubkey.pub")
	if _, err := os.Stat(publicKeyFile); os.IsNotExist(err) {
		t.Errorf("Public key file was not created at %s", publicKeyFile)
	}

	// Check metadata.toml exists inside key directory
	metadataFile := filepath.Join(keyDir, "metadata.toml")
	if _, err := os.Stat(metadataFile); os.IsNotExist(err) {
		t.Errorf("Metadata file was not created at %s", metadataFile)
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
// This creates the .kanuka directory, subdirectories, and a project config with a test UUID.
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

	// Create project config with test UUID so create command can find project UUID
	projectConfig := &configs.ProjectConfig{
		Project: configs.Project{
			UUID: TestProjectUUID,
			Name: filepath.Base(tempDir),
		},
	}

	// Set the project settings so SaveProjectConfig knows where to save
	configs.ProjectKanukaSettings = &configs.ProjectSettings{
		ProjectName:          filepath.Base(tempDir),
		ProjectPath:          tempDir,
		ProjectPublicKeyPath: publicKeysDir,
		ProjectSecretsPath:   secretsDir,
	}

	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}
}

// RegisterTestUser registers a test user with the project using their email.
func RegisterTestUser(t *testing.T, email string) {
	// Reset command state
	cmd.ResetGlobalState()

	// Create and register user
	secretsCmd := cmd.GetSecretsCmd()
	secretsCmd.SetArgs([]string{"register", "--user", email})

	err := secretsCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to register test user %s: %v", email, err)
	}
}

// RegisterTestUserByUUID registers a test user using the --file flag with their public key file.
// This is useful for tests that need to register users by UUID when they don't have an email.
func RegisterTestUserByUUID(t *testing.T, publicKeyPath string) {
	// Reset command state
	cmd.ResetGlobalState()

	// Create and register user
	secretsCmd := cmd.GetSecretsCmd()
	secretsCmd.SetArgs([]string{"register", "--file", publicKeyPath})

	err := secretsCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to register test user with key %s: %v", publicKeyPath, err)
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

// GetProjectUUID returns the project UUID from the project config.
// This is useful for tests that need to verify files are created with the correct UUID-based names.
func GetProjectUUID(t *testing.T) string {
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	if projectConfig.Project.UUID == "" {
		t.Fatalf("Project UUID not found in project config")
	}
	return projectConfig.Project.UUID
}

// GetUserUUID returns the user UUID from the user config.
// This is useful for tests that need to verify files are created with the correct UUID-based names.
func GetUserUUID(t *testing.T) string {
	userConfig, err := configs.LoadUserConfig()
	if err != nil {
		t.Fatalf("Failed to load user config: %v", err)
	}
	if userConfig.User.UUID == "" {
		return TestUserUUID
	}
	return userConfig.User.UUID
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

// CreateConfigTestCLI creates a CLI instance for testing config commands.
func CreateConfigTestCLI(subcommand string, stdout, stderr io.Writer, verboseFlag, debugFlag bool) *cobra.Command {
	return CreateConfigTestCLIWithArgs(subcommand, []string{}, stdout, stderr, verboseFlag, debugFlag)
}

// CreateConfigTestCLIWithArgs creates a CLI instance for testing config commands with extra args.
func CreateConfigTestCLIWithArgs(subcommand string, extraArgs []string, stdout, stderr io.Writer, verboseFlag, debugFlag bool) *cobra.Command {
	// Reset config command state
	cmd.ResetConfigState()

	// Create a fresh root command for this test
	rootCmd := &cobra.Command{
		Use:   "kanuka",
		Short: "Kanuka - A CLI for package management, cloud provisioning, and secrets management.",
	}

	// Add the config command
	rootCmd.AddCommand(cmd.GetConfigCmd())

	// Set output streams
	if stdout != nil {
		rootCmd.SetOut(stdout)
		cmd.GetConfigCmd().SetOut(stdout)
		for _, subcmd := range cmd.GetConfigCmd().Commands() {
			subcmd.SetOut(stdout)
		}
	}
	if stderr != nil {
		rootCmd.SetErr(stderr)
		cmd.GetConfigCmd().SetErr(stderr)
		for _, subcmd := range cmd.GetConfigCmd().Commands() {
			subcmd.SetErr(stderr)
		}
	}

	// Build args: config <subcommand> [extraArgs...]
	args := []string{"config", subcommand}
	args = append(args, extraArgs...)
	rootCmd.SetArgs(args)

	// Set the flags on the config command
	if err := cmd.GetConfigCmd().PersistentFlags().Set("verbose", fmt.Sprintf("%t", verboseFlag)); err != nil {
		log.Fatalf("Failed to set verbose flag for testing: %s", err)
	}
	if err := cmd.GetConfigCmd().PersistentFlags().Set("debug", fmt.Sprintf("%t", debugFlag)); err != nil {
		log.Fatalf("Failed to set debug flag for testing: %s", err)
	}

	return rootCmd
}

// GetKeyDirPath returns the path to the key directory for a given project UUID.
// This follows the new directory structure: {keysDir}/{projectUUID}/.
func GetKeyDirPath(keysDir, projectUUID string) string {
	return filepath.Join(keysDir, projectUUID)
}

// GetPrivateKeyPath returns the path to the private key for a given project UUID.
// This follows the new directory structure: {keysDir}/{projectUUID}/privkey.
func GetPrivateKeyPath(keysDir, projectUUID string) string {
	return filepath.Join(GetKeyDirPath(keysDir, projectUUID), "privkey")
}

// GetPublicKeyPath returns the path to the public key for a given project UUID.
// This follows the new directory structure: {keysDir}/{projectUUID}/pubkey.pub.
func GetPublicKeyPath(keysDir, projectUUID string) string {
	return filepath.Join(GetKeyDirPath(keysDir, projectUUID), "pubkey.pub")
}

// GetMetadataPath returns the path to the metadata file for a given project UUID.
// This follows the new directory structure: {keysDir}/{projectUUID}/metadata.toml.
func GetMetadataPath(keysDir, projectUUID string) string {
	return filepath.Join(GetKeyDirPath(keysDir, projectUUID), "metadata.toml")
}
