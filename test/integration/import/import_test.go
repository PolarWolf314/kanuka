package importtest

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

// setupImportTestProject initializes a complete project with user access for import tests.
func setupImportTestProject(t *testing.T, tempDir, tempUserDir string) {
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
func createEncryptedEnvFile(t *testing.T, tempDir, filename string, content string) {
	// Create a .env file.
	envPath := filepath.Join(tempDir, filename)
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
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

// exportProject runs the export command and returns the archive path.
func exportProject(t *testing.T, tempDir string) string {
	archivePath := filepath.Join(tempDir, "backup.tar.gz")
	_, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("export", []string{"-o", archivePath}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to export project: %v", err)
	}
	return archivePath
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

func TestImport_EmptyProject(t *testing.T) {
	// Create source project and export.
	sourceDir, err := os.MkdirTemp("", "kanuka-source-*")
	if err != nil {
		t.Fatalf("Failed to create source temp directory: %v", err)
	}
	defer os.RemoveAll(sourceDir)

	sourceUserDir, err := os.MkdirTemp("", "kanuka-source-user-*")
	if err != nil {
		t.Fatalf("Failed to create source temp user directory: %v", err)
	}
	defer os.RemoveAll(sourceUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, sourceDir, sourceUserDir, originalWd, originalUserSettings)

	// Initialize source project.
	setupImportTestProject(t, sourceDir, sourceUserDir)
	createEncryptedEnvFile(t, sourceDir, ".env", "SECRET=value123\n")

	// Export the project.
	archivePath := exportProject(t, sourceDir)

	// Copy archive to a location that won't be cleaned up yet.
	archiveCopy := filepath.Join(os.TempDir(), "kanuka-test-archive.tar.gz")
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Failed to read archive: %v", err)
	}
	if err := os.WriteFile(archiveCopy, archiveData, 0600); err != nil {
		t.Fatalf("Failed to copy archive: %v", err)
	}
	defer os.Remove(archiveCopy)

	// Now create target project (empty).
	targetDir, err := os.MkdirTemp("", "kanuka-target-*")
	if err != nil {
		t.Fatalf("Failed to create target temp directory: %v", err)
	}
	defer os.RemoveAll(targetDir)

	targetUserDir, err := os.MkdirTemp("", "kanuka-target-user-*")
	if err != nil {
		t.Fatalf("Failed to create target temp user directory: %v", err)
	}
	defer os.RemoveAll(targetUserDir)

	// Change to target directory - manually handle cleanup since we already set up source.
	if err := os.Chdir(targetDir); err != nil {
		t.Fatalf("Failed to change to target directory: %v", err)
	}

	// Set up target user settings.
	userConfigsPath := filepath.Join(targetUserDir, "config")
	if err := os.MkdirAll(userConfigsPath, 0755); err != nil {
		t.Fatalf("Failed to create user config directory: %v", err)
	}
	configs.UserKanukaSettings = &configs.UserSettings{
		UserKeysPath:    filepath.Join(targetUserDir, "keys"),
		UserConfigsPath: userConfigsPath,
		Username:        "testuser",
	}

	// Save user config.
	userConfig := &configs.UserConfig{
		User: configs.User{
			UUID:  shared.TestUserUUID,
			Email: shared.TestUserEmail,
		},
		Projects: make(map[string]configs.UserProjectEntry),
	}
	if err := configs.SaveUserConfig(userConfig); err != nil {
		t.Fatalf("Failed to save user config: %v", err)
	}

	// Import into empty directory (no .kanuka exists).
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("import", []string{archiveCopy}, nil, nil, true, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Import command failed: %v\nOutput: %s", err, output)
	}

	// Verify success message.
	if !strings.Contains(output, "Imported secrets from") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Verify .kanuka directory was created.
	kanukaDir := filepath.Join(targetDir, ".kanuka")
	if _, err := os.Stat(kanukaDir); os.IsNotExist(err) {
		t.Errorf(".kanuka directory was not created")
	}

	// Verify config.toml exists.
	configPath := filepath.Join(kanukaDir, "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("config.toml was not imported")
	}

	// Verify encrypted .env file exists.
	envKanukaPath := filepath.Join(targetDir, ".env.kanuka")
	if _, err := os.Stat(envKanukaPath); os.IsNotExist(err) {
		t.Errorf(".env.kanuka was not imported")
	}

	// Restore original directory.
	if err := os.Chdir(originalWd); err != nil {
		t.Fatalf("Failed to restore directory: %v", err)
	}
}

func TestImport_MergeMode(t *testing.T) {
	// Create source project and export.
	sourceDir, err := os.MkdirTemp("", "kanuka-source-*")
	if err != nil {
		t.Fatalf("Failed to create source temp directory: %v", err)
	}
	defer os.RemoveAll(sourceDir)

	sourceUserDir, err := os.MkdirTemp("", "kanuka-source-user-*")
	if err != nil {
		t.Fatalf("Failed to create source temp user directory: %v", err)
	}
	defer os.RemoveAll(sourceUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, sourceDir, sourceUserDir, originalWd, originalUserSettings)

	// Initialize source project with two .env files.
	setupImportTestProject(t, sourceDir, sourceUserDir)
	createEncryptedEnvFile(t, sourceDir, ".env", "SECRET1=value1\n")

	// Create a subdirectory with another env file.
	configDir := filepath.Join(sourceDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}
	envPath := filepath.Join(configDir, ".env.production")
	if err := os.WriteFile(envPath, []byte("PROD_SECRET=prodvalue\n"), 0600); err != nil {
		t.Fatalf("Failed to create .env.production: %v", err)
	}
	_, err = shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLI("encrypt", nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt files: %v", err)
	}

	// Export the project.
	archivePath := exportProject(t, sourceDir)

	// Copy archive.
	archiveCopy := filepath.Join(os.TempDir(), "kanuka-merge-archive.tar.gz")
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Failed to read archive: %v", err)
	}
	if err := os.WriteFile(archiveCopy, archiveData, 0600); err != nil {
		t.Fatalf("Failed to copy archive: %v", err)
	}
	defer os.Remove(archiveCopy)

	// Create target project with existing .kanuka but only the root .env.kanuka.
	targetDir, err := os.MkdirTemp("", "kanuka-target-*")
	if err != nil {
		t.Fatalf("Failed to create target temp directory: %v", err)
	}
	defer os.RemoveAll(targetDir)

	targetUserDir, err := os.MkdirTemp("", "kanuka-target-user-*")
	if err != nil {
		t.Fatalf("Failed to create target temp user directory: %v", err)
	}
	defer os.RemoveAll(targetUserDir)

	if err := os.Chdir(targetDir); err != nil {
		t.Fatalf("Failed to change to target directory: %v", err)
	}

	userConfigsPath := filepath.Join(targetUserDir, "config")
	if err := os.MkdirAll(userConfigsPath, 0755); err != nil {
		t.Fatalf("Failed to create user config directory: %v", err)
	}
	configs.UserKanukaSettings = &configs.UserSettings{
		UserKeysPath:    filepath.Join(targetUserDir, "keys"),
		UserConfigsPath: userConfigsPath,
		Username:        "testuser",
	}

	userConfig := &configs.UserConfig{
		User: configs.User{
			UUID:  shared.TestUserUUID,
			Email: shared.TestUserEmail,
		},
		Projects: make(map[string]configs.UserProjectEntry),
	}
	if err := configs.SaveUserConfig(userConfig); err != nil {
		t.Fatalf("Failed to save user config: %v", err)
	}

	// Initialize target project with only root .env.
	setupImportTestProject(t, targetDir, targetUserDir)
	createEncryptedEnvFile(t, targetDir, ".env", "EXISTING_SECRET=existingvalue\n")

	// Read original .env.kanuka content for comparison.
	existingEnvContent, err := os.ReadFile(filepath.Join(targetDir, ".env.kanuka"))
	if err != nil {
		t.Fatalf("Failed to read existing .env.kanuka: %v", err)
	}

	// Import with merge mode.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("import", []string{archiveCopy, "--merge"}, nil, nil, true, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Import merge command failed: %v\nOutput: %s", err, output)
	}

	// Verify merge mode was used.
	if !strings.Contains(output, "Merge") {
		t.Errorf("Expected Merge mode in output, got: %s", output)
	}

	// Verify existing .env.kanuka was NOT replaced (skipped).
	currentEnvContent, err := os.ReadFile(filepath.Join(targetDir, ".env.kanuka"))
	if err != nil {
		t.Fatalf("Failed to read current .env.kanuka: %v", err)
	}
	if string(currentEnvContent) != string(existingEnvContent) {
		t.Errorf("Existing .env.kanuka should not have been replaced in merge mode")
	}

	// Verify new file was added.
	prodEnvPath := filepath.Join(targetDir, "config", ".env.production.kanuka")
	if _, err := os.Stat(prodEnvPath); os.IsNotExist(err) {
		t.Errorf("New file config/.env.production.kanuka should have been added")
	}

	// Verify output shows skipped files.
	if !strings.Contains(output, "Skipped") {
		t.Errorf("Expected 'Skipped' in output, got: %s", output)
	}

	if err := os.Chdir(originalWd); err != nil {
		t.Fatalf("Failed to restore directory: %v", err)
	}
}

func TestImport_ReplaceMode(t *testing.T) {
	// Create source project and export.
	sourceDir, err := os.MkdirTemp("", "kanuka-source-*")
	if err != nil {
		t.Fatalf("Failed to create source temp directory: %v", err)
	}
	defer os.RemoveAll(sourceDir)

	sourceUserDir, err := os.MkdirTemp("", "kanuka-source-user-*")
	if err != nil {
		t.Fatalf("Failed to create source temp user directory: %v", err)
	}
	defer os.RemoveAll(sourceUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, sourceDir, sourceUserDir, originalWd, originalUserSettings)

	setupImportTestProject(t, sourceDir, sourceUserDir)
	createEncryptedEnvFile(t, sourceDir, ".env", "SOURCE_SECRET=sourcevalue\n")

	archivePath := exportProject(t, sourceDir)

	archiveCopy := filepath.Join(os.TempDir(), "kanuka-replace-archive.tar.gz")
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Failed to read archive: %v", err)
	}
	if err := os.WriteFile(archiveCopy, archiveData, 0600); err != nil {
		t.Fatalf("Failed to copy archive: %v", err)
	}
	defer os.Remove(archiveCopy)

	// Create target project with existing content.
	targetDir, err := os.MkdirTemp("", "kanuka-target-*")
	if err != nil {
		t.Fatalf("Failed to create target temp directory: %v", err)
	}
	defer os.RemoveAll(targetDir)

	targetUserDir, err := os.MkdirTemp("", "kanuka-target-user-*")
	if err != nil {
		t.Fatalf("Failed to create target temp user directory: %v", err)
	}
	defer os.RemoveAll(targetUserDir)

	if err := os.Chdir(targetDir); err != nil {
		t.Fatalf("Failed to change to target directory: %v", err)
	}

	userConfigsPath := filepath.Join(targetUserDir, "config")
	if err := os.MkdirAll(userConfigsPath, 0755); err != nil {
		t.Fatalf("Failed to create user config directory: %v", err)
	}
	configs.UserKanukaSettings = &configs.UserSettings{
		UserKeysPath:    filepath.Join(targetUserDir, "keys"),
		UserConfigsPath: userConfigsPath,
		Username:        "testuser",
	}

	userConfig := &configs.UserConfig{
		User: configs.User{
			UUID:  shared.TestUserUUID,
			Email: shared.TestUserEmail,
		},
		Projects: make(map[string]configs.UserProjectEntry),
	}
	if err := configs.SaveUserConfig(userConfig); err != nil {
		t.Fatalf("Failed to save user config: %v", err)
	}

	// Initialize target project with different content.
	setupImportTestProject(t, targetDir, targetUserDir)
	createEncryptedEnvFile(t, targetDir, ".env", "TARGET_SECRET=targetvalue\n")

	// Create an extra file in .kanuka that should be removed.
	extraFile := filepath.Join(targetDir, ".kanuka", "extra-file.txt")
	if err := os.WriteFile(extraFile, []byte("extra content"), 0600); err != nil {
		t.Fatalf("Failed to create extra file: %v", err)
	}

	// Import with replace mode.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("import", []string{archiveCopy, "--replace"}, nil, nil, true, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Import replace command failed: %v\nOutput: %s", err, output)
	}

	// Verify replace mode was used.
	if !strings.Contains(output, "Replace") {
		t.Errorf("Expected Replace mode in output, got: %s", output)
	}

	// Verify extra file was removed (since .kanuka was replaced).
	if _, err := os.Stat(extraFile); !os.IsNotExist(err) {
		t.Errorf("Extra file should have been removed in replace mode")
	}

	// Verify config.toml exists (from archive).
	configPath := filepath.Join(targetDir, ".kanuka", "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("config.toml should exist after replace")
	}

	if err := os.Chdir(originalWd); err != nil {
		t.Fatalf("Failed to restore directory: %v", err)
	}
}

func TestImport_DryRun(t *testing.T) {
	// Create source project and export.
	sourceDir, err := os.MkdirTemp("", "kanuka-source-*")
	if err != nil {
		t.Fatalf("Failed to create source temp directory: %v", err)
	}
	defer os.RemoveAll(sourceDir)

	sourceUserDir, err := os.MkdirTemp("", "kanuka-source-user-*")
	if err != nil {
		t.Fatalf("Failed to create source temp user directory: %v", err)
	}
	defer os.RemoveAll(sourceUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, sourceDir, sourceUserDir, originalWd, originalUserSettings)

	setupImportTestProject(t, sourceDir, sourceUserDir)
	createEncryptedEnvFile(t, sourceDir, ".env", "SECRET=value123\n")

	archivePath := exportProject(t, sourceDir)

	archiveCopy := filepath.Join(os.TempDir(), "kanuka-dryrun-archive.tar.gz")
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Failed to read archive: %v", err)
	}
	if err := os.WriteFile(archiveCopy, archiveData, 0600); err != nil {
		t.Fatalf("Failed to copy archive: %v", err)
	}
	defer os.Remove(archiveCopy)

	// Create empty target project.
	targetDir, err := os.MkdirTemp("", "kanuka-target-*")
	if err != nil {
		t.Fatalf("Failed to create target temp directory: %v", err)
	}
	defer os.RemoveAll(targetDir)

	targetUserDir, err := os.MkdirTemp("", "kanuka-target-user-*")
	if err != nil {
		t.Fatalf("Failed to create target temp user directory: %v", err)
	}
	defer os.RemoveAll(targetUserDir)

	if err := os.Chdir(targetDir); err != nil {
		t.Fatalf("Failed to change to target directory: %v", err)
	}

	userConfigsPath := filepath.Join(targetUserDir, "config")
	if err := os.MkdirAll(userConfigsPath, 0755); err != nil {
		t.Fatalf("Failed to create user config directory: %v", err)
	}
	configs.UserKanukaSettings = &configs.UserSettings{
		UserKeysPath:    filepath.Join(targetUserDir, "keys"),
		UserConfigsPath: userConfigsPath,
		Username:        "testuser",
	}

	userConfig := &configs.UserConfig{
		User: configs.User{
			UUID:  shared.TestUserUUID,
			Email: shared.TestUserEmail,
		},
		Projects: make(map[string]configs.UserProjectEntry),
	}
	if err := configs.SaveUserConfig(userConfig); err != nil {
		t.Fatalf("Failed to save user config: %v", err)
	}

	// Import with dry-run.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("import", []string{archiveCopy, "--dry-run"}, nil, nil, true, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Import dry-run command failed: %v\nOutput: %s", err, output)
	}

	// Verify dry-run message.
	if !strings.Contains(output, "Dry run") {
		t.Errorf("Expected 'Dry run' in output, got: %s", output)
	}

	// Verify no changes were made.
	kanukaDir := filepath.Join(targetDir, ".kanuka")
	if _, err := os.Stat(kanukaDir); !os.IsNotExist(err) {
		t.Errorf(".kanuka directory should NOT have been created in dry-run mode")
	}

	if err := os.Chdir(originalWd); err != nil {
		t.Fatalf("Failed to restore directory: %v", err)
	}
}

func TestImport_InvalidArchive(t *testing.T) {
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

	// Create an invalid archive (not a proper tar.gz).
	invalidArchive := filepath.Join(tempDir, "invalid.tar.gz")
	if err := os.WriteFile(invalidArchive, []byte("not a valid archive"), 0600); err != nil {
		t.Fatalf("Failed to create invalid archive: %v", err)
	}

	// Try to import invalid archive.
	_, err = shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("import", []string{invalidArchive}, nil, nil, false, false)
		return testCmd.Execute()
	})

	// Should fail.
	if err == nil {
		t.Errorf("Expected error for invalid archive, got none")
	}
}

func TestImport_MissingConfig(t *testing.T) {
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

	// Create an archive missing config.toml.
	archivePath := filepath.Join(tempDir, "incomplete.tar.gz")
	outFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("Failed to create archive file: %v", err)
	}
	gzWriter := gzip.NewWriter(outFile)
	tarWriter := tar.NewWriter(gzWriter)

	// Add only a .kanuka file (no config.toml).
	content := []byte("some encrypted content")
	header := &tar.Header{
		Name: ".env.kanuka",
		Mode: 0600,
		Size: int64(len(content)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("Failed to write tar header: %v", err)
	}
	if _, err := tarWriter.Write(content); err != nil {
		t.Fatalf("Failed to write tar content: %v", err)
	}

	tarWriter.Close()
	gzWriter.Close()
	outFile.Close()

	// Try to import incomplete archive.
	_, err = shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("import", []string{archivePath}, nil, nil, false, false)
		return testCmd.Execute()
	})

	// Should fail with validation error.
	if err == nil {
		t.Errorf("Expected error for archive missing config.toml, got none")
	}
}

func TestImport_ArchiveNotFound(t *testing.T) {
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

	// Try to import non-existent archive.
	_, err = shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("import", []string{"/path/to/nonexistent.tar.gz"}, nil, nil, false, false)
		return testCmd.Execute()
	})

	// Should fail.
	if err == nil {
		t.Errorf("Expected error for non-existent archive, got none")
	}
}

func TestImport_ConflictingFlags(t *testing.T) {
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

	// Create a dummy archive file so it exists.
	dummyArchive := filepath.Join(tempDir, "dummy.tar.gz")
	if err := os.WriteFile(dummyArchive, []byte{}, 0600); err != nil {
		t.Fatalf("Failed to create dummy archive: %v", err)
	}

	// Try to use both --merge and --replace.
	_, err = shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("import", []string{dummyArchive, "--merge", "--replace"}, nil, nil, false, false)
		return testCmd.Execute()
	})

	// Should fail.
	if err == nil {
		t.Errorf("Expected error for conflicting flags, got none")
	}
}

func TestImport_RoundTrip(t *testing.T) {
	// Create source project and export.
	sourceDir, err := os.MkdirTemp("", "kanuka-source-*")
	if err != nil {
		t.Fatalf("Failed to create source temp directory: %v", err)
	}
	defer os.RemoveAll(sourceDir)

	sourceUserDir, err := os.MkdirTemp("", "kanuka-source-user-*")
	if err != nil {
		t.Fatalf("Failed to create source temp user directory: %v", err)
	}
	defer os.RemoveAll(sourceUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, sourceDir, sourceUserDir, originalWd, originalUserSettings)

	setupImportTestProject(t, sourceDir, sourceUserDir)
	createEncryptedEnvFile(t, sourceDir, ".env", "ROUNDTRIP_SECRET=roundtripvalue\n")

	// Export the project.
	archivePath := exportProject(t, sourceDir)

	// Get archive contents.
	sourceArchiveContents := getArchiveContents(t, archivePath)

	archiveCopy := filepath.Join(os.TempDir(), "kanuka-roundtrip-archive.tar.gz")
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Failed to read archive: %v", err)
	}
	if err := os.WriteFile(archiveCopy, archiveData, 0600); err != nil {
		t.Fatalf("Failed to copy archive: %v", err)
	}
	defer os.Remove(archiveCopy)

	// Create target project.
	targetDir, err := os.MkdirTemp("", "kanuka-target-*")
	if err != nil {
		t.Fatalf("Failed to create target temp directory: %v", err)
	}
	defer os.RemoveAll(targetDir)

	targetUserDir, err := os.MkdirTemp("", "kanuka-target-user-*")
	if err != nil {
		t.Fatalf("Failed to create target temp user directory: %v", err)
	}
	defer os.RemoveAll(targetUserDir)

	if err := os.Chdir(targetDir); err != nil {
		t.Fatalf("Failed to change to target directory: %v", err)
	}

	userConfigsPath := filepath.Join(targetUserDir, "config")
	if err := os.MkdirAll(userConfigsPath, 0755); err != nil {
		t.Fatalf("Failed to create user config directory: %v", err)
	}
	configs.UserKanukaSettings = &configs.UserSettings{
		UserKeysPath:    filepath.Join(targetUserDir, "keys"),
		UserConfigsPath: userConfigsPath,
		Username:        "testuser",
	}

	userConfig := &configs.UserConfig{
		User: configs.User{
			UUID:  shared.TestUserUUID,
			Email: shared.TestUserEmail,
		},
		Projects: make(map[string]configs.UserProjectEntry),
	}
	if err := configs.SaveUserConfig(userConfig); err != nil {
		t.Fatalf("Failed to save user config: %v", err)
	}

	// Import into target.
	_, err = shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("import", []string{archiveCopy}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Import command failed: %v", err)
	}

	// Re-export from target.
	targetArchive := filepath.Join(targetDir, "re-export.tar.gz")
	_, err = shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("export", []string{"-o", targetArchive}, nil, nil, false, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Re-export command failed: %v", err)
	}

	// Compare archive contents.
	targetArchiveContents := getArchiveContents(t, targetArchive)

	// Both should have the same files.
	if len(sourceArchiveContents) != len(targetArchiveContents) {
		t.Errorf("Archive contents differ in length: source=%d, target=%d",
			len(sourceArchiveContents), len(targetArchiveContents))
	}

	// Check that all source files are in target.
	for _, srcFile := range sourceArchiveContents {
		found := false
		for _, tgtFile := range targetArchiveContents {
			if srcFile == tgtFile {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("File %s from source not found in re-exported archive", srcFile)
		}
	}

	if err := os.Chdir(originalWd); err != nil {
		t.Fatalf("Failed to restore directory: %v", err)
	}
}

func TestImport_EmptyConfigFile(t *testing.T) {
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

	// Create an archive with empty config.toml.
	archivePath := filepath.Join(tempDir, "empty-config.tar.gz")
	outFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("Failed to create archive file: %v", err)
	}
	gzWriter := gzip.NewWriter(outFile)
	tarWriter := tar.NewWriter(gzWriter)

	// Add empty config.toml.
	configContent := []byte("")
	configHeader := &tar.Header{
		Name: ".kanuka/config.toml",
		Mode: 0600,
		Size: int64(len(configContent)),
	}
	if err := tarWriter.WriteHeader(configHeader); err != nil {
		t.Fatalf("Failed to write tar header: %v", err)
	}
	if _, err := tarWriter.Write(configContent); err != nil {
		t.Fatalf("Failed to write tar content: %v", err)
	}

	// Add a dummy .kanuka file to pass structure validation.
	content := []byte("some encrypted content")
	header := &tar.Header{
		Name: ".env.kanuka",
		Mode: 0600,
		Size: int64(len(content)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("Failed to write tar header: %v", err)
	}
	if _, err := tarWriter.Write(content); err != nil {
		t.Fatalf("Failed to write tar content: %v", err)
	}

	tarWriter.Close()
	gzWriter.Close()
	outFile.Close()

	// Try to import archive with empty config.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("import", []string{archivePath}, nil, nil, false, false)
		return testCmd.Execute()
	})

	// Should fail with validation error.
	if err == nil {
		t.Errorf("Expected error for empty config.toml, got none")
	}

	// Verify error message mentions empty config.
	if !strings.Contains(output, "empty") && !strings.Contains(output, "invalid") {
		t.Errorf("Expected error message to mention empty/invalid config, got: %s", output)
	}

	// Verify .kanuka directory was cleaned up.
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if _, err := os.Stat(kanukaDir); !os.IsNotExist(err) {
		t.Errorf(".kanuka directory should have been cleaned up after validation failure")
	}

	if err := os.Chdir(originalWd); err != nil {
		t.Fatalf("Failed to restore directory: %v", err)
	}
}

func TestImport_InvalidTOMLConfigFile(t *testing.T) {
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

	// Create an archive with invalid TOML config.toml.
	archivePath := filepath.Join(tempDir, "invalid-toml.tar.gz")
	outFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("Failed to create archive file: %v", err)
	}
	gzWriter := gzip.NewWriter(outFile)
	tarWriter := tar.NewWriter(gzWriter)

	// Add invalid TOML config.toml.
	configContent := []byte("[invalid toml [unclosed bracket")
	configHeader := &tar.Header{
		Name: ".kanuka/config.toml",
		Mode: 0600,
		Size: int64(len(configContent)),
	}
	if err := tarWriter.WriteHeader(configHeader); err != nil {
		t.Fatalf("Failed to write tar header: %v", err)
	}
	if _, err := tarWriter.Write(configContent); err != nil {
		t.Fatalf("Failed to write tar content: %v", err)
	}

	// Add a dummy .kanuka file to pass structure validation.
	content := []byte("some encrypted content")
	header := &tar.Header{
		Name: ".env.kanuka",
		Mode: 0600,
		Size: int64(len(content)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("Failed to write tar header: %v", err)
	}
	if _, err := tarWriter.Write(content); err != nil {
		t.Fatalf("Failed to write tar content: %v", err)
	}

	tarWriter.Close()
	gzWriter.Close()
	outFile.Close()

	// Try to import archive with invalid TOML.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("import", []string{archivePath}, nil, nil, false, false)
		return testCmd.Execute()
	})

	// Should fail with validation error.
	if err == nil {
		t.Errorf("Expected error for invalid TOML config.toml, got none")
	}

	// Verify error message mentions invalid config.
	if !strings.Contains(output, "invalid") {
		t.Errorf("Expected error message to mention invalid config, got: %s", output)
	}

	// Verify .kanuka directory was cleaned up.
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if _, err := os.Stat(kanukaDir); !os.IsNotExist(err) {
		t.Errorf(".kanuka directory should have been cleaned up after validation failure")
	}

	if err := os.Chdir(originalWd); err != nil {
		t.Fatalf("Failed to restore directory: %v", err)
	}
}

func TestImport_VerboseOutput(t *testing.T) {
	// Create source project and export.
	sourceDir, err := os.MkdirTemp("", "kanuka-source-*")
	if err != nil {
		t.Fatalf("Failed to create source temp directory: %v", err)
	}
	defer os.RemoveAll(sourceDir)

	sourceUserDir, err := os.MkdirTemp("", "kanuka-source-user-*")
	if err != nil {
		t.Fatalf("Failed to create source temp user directory: %v", err)
	}
	defer os.RemoveAll(sourceUserDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings
	shared.SetupTestEnvironment(t, sourceDir, sourceUserDir, originalWd, originalUserSettings)

	setupImportTestProject(t, sourceDir, sourceUserDir)
	createEncryptedEnvFile(t, sourceDir, ".env", "SECRET=value123\n")

	archivePath := exportProject(t, sourceDir)

	archiveCopy := filepath.Join(os.TempDir(), "kanuka-verbose-archive.tar.gz")
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Failed to read archive: %v", err)
	}
	if err := os.WriteFile(archiveCopy, archiveData, 0600); err != nil {
		t.Fatalf("Failed to copy archive: %v", err)
	}
	defer os.Remove(archiveCopy)

	// Create target project.
	targetDir, err := os.MkdirTemp("", "kanuka-target-*")
	if err != nil {
		t.Fatalf("Failed to create target temp directory: %v", err)
	}
	defer os.RemoveAll(targetDir)

	targetUserDir, err := os.MkdirTemp("", "kanuka-target-user-*")
	if err != nil {
		t.Fatalf("Failed to create target temp user directory: %v", err)
	}
	defer os.RemoveAll(targetUserDir)

	if err := os.Chdir(targetDir); err != nil {
		t.Fatalf("Failed to change to target directory: %v", err)
	}

	userConfigsPath := filepath.Join(targetUserDir, "config")
	if err := os.MkdirAll(userConfigsPath, 0755); err != nil {
		t.Fatalf("Failed to create user config directory: %v", err)
	}
	configs.UserKanukaSettings = &configs.UserSettings{
		UserKeysPath:    filepath.Join(targetUserDir, "keys"),
		UserConfigsPath: userConfigsPath,
		Username:        "testuser",
	}

	userConfig := &configs.UserConfig{
		User: configs.User{
			UUID:  shared.TestUserUUID,
			Email: shared.TestUserEmail,
		},
		Projects: make(map[string]configs.UserProjectEntry),
	}
	if err := configs.SaveUserConfig(userConfig); err != nil {
		t.Fatalf("Failed to save user config: %v", err)
	}

	// Import with verbose flag.
	output, err := shared.CaptureOutput(func() error {
		testCmd := shared.CreateTestCLIWithArgs("import", []string{archiveCopy}, nil, nil, true, false)
		return testCmd.Execute()
	})
	if err != nil {
		t.Fatalf("Import verbose command failed: %v\nOutput: %s", err, output)
	}

	// Verify success message appears.
	if !strings.Contains(output, "Imported secrets from") {
		t.Errorf("Expected success message in verbose output, got: %s", output)
	}

	if err := os.Chdir(originalWd); err != nil {
		t.Fatalf("Failed to restore directory: %v", err)
	}
}
