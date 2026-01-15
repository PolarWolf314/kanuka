package export

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// setupExportTestProject initializes a complete project with user access for export tests.
func setupExportTestProject(t *testing.T, tempDir, tempUserDir string) {
	// Initialize project using init command.
	_, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLI("init", nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}
}

// createEncryptedEnvFile creates a .env file and encrypts it.
func createEncryptedEnvFile(t *testing.T, tempDir, filename string) {
	// Create a .env file.
	envPath := filepath.Join(tempDir, filename)
	if err := os.WriteFile(envPath, []byte("SECRET=value123\n"), 0600); err != nil {
		t.Fatalf("Failed to create %s file: %v", filename, err)
	}

	// Encrypt it.
	_, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLI("encrypt", nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt files: %v", err)
	}
}

// getArchiveContents reads a tar.gz archive and returns a list of file paths it contains.
func getArchiveContents(t *testing.T, archivePath string) []string {
	file, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("Failed to open archive: %v", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	var files []string

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read tar header: %v", err)
		}
		files = append(files, header.Name)
	}

	return files
}

func TestExport_Basic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize the project.
	setupExportTestProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file.
	createEncryptedEnvFile(t, tempDir, ".env")

	// Run export command (use verbose to ensure output is captured).
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLI("export", nil, nil, true, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Export command failed with error: %v\nOutput: %s", err, output)
	}

	// Verify success message.
	if !strings.Contains(output, "Exported secrets to") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Find the created archive (default name includes date).
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp directory: %v", err)
	}

	var archivePath string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "kanuka-secrets-") && strings.HasSuffix(entry.Name(), ".tar.gz") {
			archivePath = filepath.Join(tempDir, entry.Name())
			break
		}
	}

	if archivePath == "" {
		t.Fatal("Archive file was not created")
	}

	// Verify archive exists.
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Fatalf("Archive file does not exist at %s", archivePath)
	}
}

func TestExport_ContainsExpectedFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize the project.
	setupExportTestProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file.
	createEncryptedEnvFile(t, tempDir, ".env")

	// Run export with custom output path.
	archivePath := filepath.Join(tempDir, "test-export.tar.gz")
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("export", []string{"-o", archivePath}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Export command failed with error: %v\nOutput: %s", err, output)
	}

	// Get archive contents.
	files := getArchiveContents(t, archivePath)

	// Verify expected files are present.
	expectedFiles := []string{
		".kanuka/config.toml",
	}

	for _, expected := range expectedFiles {
		found := false
		for _, f := range files {
			if f == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected file %s not found in archive. Archive contains: %v", expected, files)
		}
	}

	// Verify public key files are present.
	hasPublicKey := false
	for _, f := range files {
		if strings.HasPrefix(f, ".kanuka/public_keys/") && strings.HasSuffix(f, ".pub") {
			hasPublicKey = true
			break
		}
	}
	if !hasPublicKey {
		t.Errorf("Expected public key file in archive. Archive contains: %v", files)
	}

	// Verify user kanuka files are present.
	hasUserKanuka := false
	for _, f := range files {
		if strings.HasPrefix(f, ".kanuka/secrets/") && strings.HasSuffix(f, ".kanuka") {
			hasUserKanuka = true
			break
		}
	}
	if !hasUserKanuka {
		t.Errorf("Expected user kanuka file in archive. Archive contains: %v", files)
	}

	// Verify encrypted .env file is present.
	hasEncryptedEnv := false
	for _, f := range files {
		if f == ".env.kanuka" {
			hasEncryptedEnv = true
			break
		}
	}
	if !hasEncryptedEnv {
		t.Errorf("Expected .env.kanuka file in archive. Archive contains: %v", files)
	}
}

func TestExport_ExcludesPrivateKey(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize the project.
	setupExportTestProject(t, tempDir, tempUserDir)

	// Run export with custom output path.
	archivePath := filepath.Join(tempDir, "test-export.tar.gz")
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("export", []string{"-o", archivePath}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Export command failed with error: %v\nOutput: %s", err, output)
	}

	// Get archive contents.
	files := getArchiveContents(t, archivePath)

	// Verify no private key files are present.
	for _, f := range files {
		if strings.Contains(f, "privkey") || strings.Contains(f, "private") {
			t.Errorf("Private key file should not be in archive: %s", f)
		}
	}
}

func TestExport_ExcludesPlaintext(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize the project.
	setupExportTestProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file.
	createEncryptedEnvFile(t, tempDir, ".env")

	// Run export with custom output path.
	archivePath := filepath.Join(tempDir, "test-export.tar.gz")
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("export", []string{"-o", archivePath}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Export command failed with error: %v\nOutput: %s", err, output)
	}

	// Get archive contents.
	files := getArchiveContents(t, archivePath)

	// Verify plaintext .env file is NOT present (only .env.kanuka should be).
	for _, f := range files {
		if f == ".env" {
			t.Errorf("Plaintext .env file should not be in archive")
		}
	}
}

func TestExport_CustomOutput(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize the project.
	setupExportTestProject(t, tempDir, tempUserDir)

	// Create a custom output directory.
	customOutputDir := filepath.Join(tempDir, "backups")
	if err := os.MkdirAll(customOutputDir, 0755); err != nil {
		t.Fatalf("Failed to create custom output directory: %v", err)
	}

	customOutputPath := filepath.Join(customOutputDir, "my-backup.tar.gz")

	// Run export with custom output path (use verbose to ensure output is captured).
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("export", []string{"-o", customOutputPath}, nil, nil, true, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Export command failed with error: %v\nOutput: %s", err, output)
	}

	// Verify success message mentions the custom path.
	if !strings.Contains(output, "my-backup.tar.gz") {
		t.Errorf("Expected output to mention custom path, got: %s", output)
	}

	// Verify archive exists at custom path.
	if _, err := os.Stat(customOutputPath); os.IsNotExist(err) {
		t.Fatalf("Archive file does not exist at custom path %s", customOutputPath)
	}
}

func TestExport_NotInitialized(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Don't initialize the project - run export directly.
	output, _ := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLI("export", nil, nil, true, false)
		return testCmd.Execute()
	})

	// Verify error message about not initialized.
	if !strings.Contains(output, "not") && !strings.Contains(output, "init") {
		t.Errorf("Expected error message about project not initialized, got: %s", output)
	}
}

func TestExport_VerboseOutput(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize the project.
	setupExportTestProject(t, tempDir, tempUserDir)

	// Run export with verbose flag.
	archivePath := filepath.Join(tempDir, "test-export.tar.gz")
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("export", []string{"-o", archivePath}, nil, nil, true, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Export command with verbose failed: %v\nOutput: %s", err, output)
	}

	// Verify success message appears.
	if !strings.Contains(output, "Exported secrets to") {
		t.Errorf("Expected success message in verbose output, got: %s", output)
	}
}

func TestExport_NoSecretFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize the project but don't create any .env files.
	setupExportTestProject(t, tempDir, tempUserDir)

	// Run export with custom output path (use verbose to ensure output is captured).
	archivePath := filepath.Join(tempDir, "test-export.tar.gz")
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("export", []string{"-o", archivePath}, nil, nil, true, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Export command failed with error: %v\nOutput: %s", err, output)
	}

	// Should still succeed - there are still config, public keys, and user kanuka files.
	if !strings.Contains(output, "Exported secrets to") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Verify archive was created.
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Fatalf("Archive file should exist even without secret files")
	}
}

func TestExport_SubdirectorySecrets(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize the project.
	setupExportTestProject(t, tempDir, tempUserDir)

	// Create a subdirectory with a .env file.
	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	envPath := filepath.Join(configDir, ".env.production")
	if err := os.WriteFile(envPath, []byte("PROD_SECRET=value\n"), 0600); err != nil {
		t.Fatalf("Failed to create .env.production file: %v", err)
	}

	// Encrypt it.
	_, err = shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLI("encrypt", nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt files: %v", err)
	}

	// Run export with custom output path.
	archivePath := filepath.Join(tempDir, "test-export.tar.gz")
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("export", []string{"-o", archivePath}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Export command failed with error: %v\nOutput: %s", err, output)
	}

	// Get archive contents.
	files := getArchiveContents(t, archivePath)

	// Verify subdirectory encrypted file is present.
	hasSubdirSecret := false
	for _, f := range files {
		if f == "config/.env.production.kanuka" {
			hasSubdirSecret = true
			break
		}
	}
	if !hasSubdirSecret {
		t.Errorf("Expected config/.env.production.kanuka in archive. Archive contains: %v", files)
	}
}

func TestExport_MissingConfigToml_NoMigration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	defer func() {
		configs.UserKanukaSettings = originalUserSettings
	}()

	_ = configs.UserKanukaSettings

	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	if err := os.MkdirAll(publicKeysDir, 0700); err != nil {
		t.Fatalf("Failed to create .kanuka directory: %v", err)
	}
	if err := os.MkdirAll(secretsDir, 0700); err != nil {
		t.Fatalf("Failed to create secrets directory: %v", err)
	}

	pubKeyPath := filepath.Join(publicKeysDir, "user.pub")
	if err := os.WriteFile(pubKeyPath, []byte("fake public key"), 0600); err != nil {
		t.Fatalf("Failed to create public key file: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLI("export", nil, nil, true, false)
		return testCmd.Execute()
	})

	entries, _ := os.ReadDir(tempDir)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".kanuka-backup-") {
			t.Fatalf("Export should not create backup folder, but found: %s\nOutput: %s", entry.Name(), output)
		}
	}
}

func TestExport_InvalidConfigToml_ShouldError(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	defer func() {
		configs.UserKanukaSettings = originalUserSettings
	}()

	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if err := os.MkdirAll(kanukaDir, 0700); err != nil {
		t.Fatalf("Failed to create .kanuka directory: %v", err)
	}

	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	if err := os.MkdirAll(publicKeysDir, 0700); err != nil {
		t.Fatalf("Failed to create public_keys directory: %v", err)
	}

	secretsDir := filepath.Join(kanukaDir, "secrets")
	if err := os.MkdirAll(secretsDir, 0700); err != nil {
		t.Fatalf("Failed to create secrets directory: %v", err)
	}

	configPath := filepath.Join(kanukaDir, "config.toml")
	configContent := []byte("[invalid toml [unclosed bracket")
	if err := os.WriteFile(configPath, configContent, 0600); err != nil {
		t.Fatalf("Failed to create config.toml: %v", err)
	}

	pubKeyPath := filepath.Join(publicKeysDir, "user.pub")
	if err := os.WriteFile(pubKeyPath, []byte("fake public key"), 0600); err != nil {
		t.Fatalf("Failed to create public key file: %v", err)
	}

	archivePath := filepath.Join(tempDir, "test-export.tar.gz")
	output, _ := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("export", []string{"-o", archivePath}, nil, nil, true, false)
		return testCmd.Execute()
	})

	if !strings.Contains(strings.ToLower(output), "invalid") && !strings.Contains(strings.ToLower(output), "toml") {
		t.Errorf("Expected error message to mention invalid TOML, got: %s", output)
	}

	if _, err := os.Stat(archivePath); !os.IsNotExist(err) {
		t.Errorf("Archive should not have been created with invalid config.toml")
	}
}
