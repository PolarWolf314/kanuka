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
//
//nolint:unused
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
//
//nolint:unused
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
//
//nolint:unused
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
	defer func() {
		_ = os.Chdir(originalWd)
	}()

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

	output, _ := shared.CaptureOutput(func() error {
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
	defer func() {
		_ = os.Chdir(originalWd)
	}()

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
