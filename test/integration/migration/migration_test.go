package migration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
)

// TestMigrationFromLegacyFormat tests that legacy projects are properly migrated.
func TestMigrationFromLegacyFormat(t *testing.T) {
	t.Run("DetectsLegacyProject", func(t *testing.T) {
		testDetectsLegacyProject(t)
	})

	t.Run("MigrateLegacyProjectDirectly", func(t *testing.T) {
		testMigrateLegacyProjectDirectly(t)
	})

	t.Run("MigrationCreatesBackup", func(t *testing.T) {
		testMigrationCreatesBackup(t)
	})

	t.Run("MigrationPreservesFileContent", func(t *testing.T) {
		testMigrationPreservesFileContent(t)
	})

	t.Run("MigrationCreatesConfigToml", func(t *testing.T) {
		testMigrationCreatesConfigToml(t)
	})
}

// testDetectsLegacyProject tests that legacy projects are properly detected.
func testDetectsLegacyProject(t *testing.T) {
	tempDir := t.TempDir()
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")

	// Empty directory is not legacy.
	if configs.IsLegacyProject(tempDir) {
		t.Fatal("Empty directory should not be legacy")
	}

	// Create .kanuka structure.
	if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}

	// Directory with no .pub files is not legacy.
	if configs.IsLegacyProject(tempDir) {
		t.Fatal("Directory with no .pub files should not be legacy")
	}

	// Add a .pub file - now it should be legacy.
	if err := os.WriteFile(filepath.Join(publicKeysDir, "alice.pub"), []byte("key"), 0600); err != nil {
		t.Fatalf("Failed to create .pub file: %v", err)
	}

	if !configs.IsLegacyProject(tempDir) {
		t.Fatal("Directory with .pub file but no config.toml should be legacy")
	}

	// Add config.toml - now it should NOT be legacy.
	configPath := filepath.Join(kanukaDir, "config.toml")
	if err := os.WriteFile(configPath, []byte("[project]\n"), 0600); err != nil {
		t.Fatalf("Failed to create config.toml: %v", err)
	}

	if configs.IsLegacyProject(tempDir) {
		t.Fatal("Directory with config.toml should not be legacy")
	}
}

// testMigrateLegacyProjectDirectly tests direct migration of a legacy project.
func testMigrateLegacyProjectDirectly(t *testing.T) {
	tempDir := t.TempDir()
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
		t.Fatalf("Failed to create public_keys: %v", err)
	}
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		t.Fatalf("Failed to create secrets: %v", err)
	}

	// Create legacy user files.
	if err := os.WriteFile(filepath.Join(publicKeysDir, "alice.pub"), []byte("alice-key"), 0600); err != nil {
		t.Fatalf("Failed to create alice.pub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(secretsDir, "alice.kanuka"), []byte("alice-secret"), 0600); err != nil {
		t.Fatalf("Failed to create alice.kanuka: %v", err)
	}
	if err := os.WriteFile(filepath.Join(publicKeysDir, "bob.pub"), []byte("bob-key"), 0600); err != nil {
		t.Fatalf("Failed to create bob.pub: %v", err)
	}

	// Verify it's detected as legacy.
	if !configs.IsLegacyProject(tempDir) {
		t.Fatal("Project should be detected as legacy before migration")
	}

	// Run migration.
	result, err := configs.MigrateProject(tempDir)
	if err != nil {
		t.Fatalf("MigrateProject failed: %v", err)
	}

	// Verify project UUID was generated.
	if result.ProjectUUID == "" {
		t.Fatal("Expected project UUID")
	}
	if len(result.ProjectUUID) != 36 {
		t.Fatalf("Expected UUID length 36, got %d", len(result.ProjectUUID))
	}

	// Verify users were migrated.
	if len(result.MigratedUsers) != 2 {
		t.Fatalf("Expected 2 migrated users, got %d", len(result.MigratedUsers))
	}

	// Verify project is no longer detected as legacy.
	if configs.IsLegacyProject(tempDir) {
		t.Fatal("Project should not be detected as legacy after migration")
	}

	// Verify old username-based files were renamed.
	if _, err := os.Stat(filepath.Join(publicKeysDir, "alice.pub")); !os.IsNotExist(err) {
		t.Fatal("alice.pub should have been renamed during migration")
	}
	if _, err := os.Stat(filepath.Join(publicKeysDir, "bob.pub")); !os.IsNotExist(err) {
		t.Fatal("bob.pub should have been renamed during migration")
	}

	// Verify UUID-based files were created.
	entries, err := os.ReadDir(publicKeysDir)
	if err != nil {
		t.Fatalf("Failed to read public_keys: %v", err)
	}

	uuidFiles := 0
	for _, entry := range entries {
		name := strings.TrimSuffix(entry.Name(), ".pub")
		if !configs.IsLegacyUserKey(name) {
			uuidFiles++
		}
	}
	if uuidFiles != 2 {
		t.Fatalf("Expected 2 UUID-based .pub files, got %d", uuidFiles)
	}
}

// testMigrationCreatesBackup tests that migration creates a backup of the original files.
func testMigrationCreatesBackup(t *testing.T) {
	tempDir := t.TempDir()
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
		t.Fatalf("Failed to create public_keys: %v", err)
	}
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		t.Fatalf("Failed to create secrets: %v", err)
	}

	// Create legacy user files.
	if err := os.WriteFile(filepath.Join(publicKeysDir, "charlie.pub"), []byte("charlie-key"), 0600); err != nil {
		t.Fatalf("Failed to create charlie.pub: %v", err)
	}

	// Run migration.
	result, err := configs.MigrateProject(tempDir)
	if err != nil {
		t.Fatalf("MigrateProject failed: %v", err)
	}

	// Verify backup path is set.
	if result.BackupPath == "" {
		t.Fatal("Expected backup path in result")
	}

	// Verify backup directory exists.
	if _, err := os.Stat(result.BackupPath); os.IsNotExist(err) {
		t.Fatal("Backup directory was not created")
	}

	// Verify backup contains the original files.
	backupPubKeysDir := filepath.Join(result.BackupPath, "public_keys")
	if _, err := os.Stat(filepath.Join(backupPubKeysDir, "charlie.pub")); os.IsNotExist(err) {
		t.Fatal("Backup should contain original charlie.pub file")
	}

	// Verify backup has original content.
	content, err := os.ReadFile(filepath.Join(backupPubKeysDir, "charlie.pub"))
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}
	if string(content) != "charlie-key" {
		t.Fatalf("Backup content mismatch: expected %q, got %q", "charlie-key", content)
	}
}

// testMigrationPreservesFileContent tests that migration preserves the content of files.
func testMigrationPreservesFileContent(t *testing.T) {
	tempDir := t.TempDir()
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
		t.Fatalf("Failed to create public_keys: %v", err)
	}
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		t.Fatalf("Failed to create secrets: %v", err)
	}

	// Create legacy user files with specific content.
	originalPubContent := []byte("dave-public-key-content-12345")
	originalSecretContent := []byte("dave-secret-content-67890")

	if err := os.WriteFile(filepath.Join(publicKeysDir, "dave.pub"), originalPubContent, 0600); err != nil {
		t.Fatalf("Failed to create dave.pub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(secretsDir, "dave.kanuka"), originalSecretContent, 0600); err != nil {
		t.Fatalf("Failed to create dave.kanuka: %v", err)
	}

	// Run migration.
	result, err := configs.MigrateProject(tempDir)
	if err != nil {
		t.Fatalf("MigrateProject failed: %v", err)
	}

	// Find the migrated user.
	if len(result.MigratedUsers) != 1 {
		t.Fatalf("Expected 1 migrated user, got %d", len(result.MigratedUsers))
	}

	migratedUser := result.MigratedUsers[0]
	if migratedUser.OldUsername != "dave" {
		t.Fatalf("Expected migrated user 'dave', got %q", migratedUser.OldUsername)
	}

	// Verify the migrated public key content.
	newPubPath := filepath.Join(publicKeysDir, migratedUser.NewUUID+".pub")
	content, err := os.ReadFile(newPubPath)
	if err != nil {
		t.Fatalf("Failed to read migrated pub file: %v", err)
	}
	if string(content) != string(originalPubContent) {
		t.Errorf("Public key content was not preserved. Expected %q, got %q", originalPubContent, content)
	}

	// Verify the migrated secret content.
	newSecretPath := filepath.Join(secretsDir, migratedUser.NewUUID+".kanuka")
	content, err = os.ReadFile(newSecretPath)
	if err != nil {
		t.Fatalf("Failed to read migrated secret file: %v", err)
	}
	if string(content) != string(originalSecretContent) {
		t.Errorf("Secret content was not preserved. Expected %q, got %q", originalSecretContent, content)
	}
}

// testMigrationCreatesConfigToml tests that migration creates a config.toml.
func testMigrationCreatesConfigToml(t *testing.T) {
	tempDir := t.TempDir()
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")
	secretsDir := filepath.Join(kanukaDir, "secrets")

	if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
		t.Fatalf("Failed to create public_keys: %v", err)
	}
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		t.Fatalf("Failed to create secrets: %v", err)
	}

	// Create legacy user files.
	if err := os.WriteFile(filepath.Join(publicKeysDir, "eve.pub"), []byte("eve-key"), 0600); err != nil {
		t.Fatalf("Failed to create eve.pub: %v", err)
	}

	// Run migration.
	result, err := configs.MigrateProject(tempDir)
	if err != nil {
		t.Fatalf("MigrateProject failed: %v", err)
	}

	// Verify config.toml was created.
	configPath := filepath.Join(kanukaDir, "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config.toml was not created during migration")
	}

	// Load and verify the config.
	oldProjectPath := configs.ProjectKanukaSettings.ProjectPath
	configs.ProjectKanukaSettings.ProjectPath = tempDir
	defer func() { configs.ProjectKanukaSettings.ProjectPath = oldProjectPath }()

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}

	if projectConfig.Project.UUID != result.ProjectUUID {
		t.Errorf("Config UUID mismatch: expected %q, got %q", result.ProjectUUID, projectConfig.Project.UUID)
	}

	// Verify migrated user is in the config.
	if len(projectConfig.Users) != 1 {
		t.Fatalf("Expected 1 user in config, got %d", len(projectConfig.Users))
	}

	// Verify the user's email is set to the placeholder.
	for _, email := range projectConfig.Users {
		if !strings.HasSuffix(email, "@unknown.local") {
			t.Errorf("Expected placeholder email ending with @unknown.local, got %q", email)
		}
	}
}

// TestNonLegacyProjectNotMigrated tests that non-legacy projects are not migrated.
func TestNonLegacyProjectNotMigrated(t *testing.T) {
	tempDir := t.TempDir()
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")

	if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
		t.Fatalf("Failed to create public_keys: %v", err)
	}

	// Create config.toml to make it a modern project.
	configPath := filepath.Join(kanukaDir, "config.toml")
	configContent := `[project]
project_uuid = "existing-uuid-1234"
name = "test-project"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to create config.toml: %v", err)
	}

	// Verify it's NOT detected as legacy.
	if configs.IsLegacyProject(tempDir) {
		t.Fatal("Project with config.toml should not be detected as legacy")
	}

	// Attempting to migrate should fail.
	_, err := configs.MigrateProject(tempDir)
	if err == nil {
		t.Fatal("MigrateProject should fail for non-legacy project")
	}
}

// TestMigrateUserKeys tests the user keys migration functionality.
func TestMigrateUserKeys(t *testing.T) {
	t.Run("MigratesLegacyKeys", func(t *testing.T) {
		tempDir := t.TempDir()
		oldKeysPath := configs.UserKanukaSettings.UserKeysPath
		configs.UserKanukaSettings.UserKeysPath = tempDir
		defer func() { configs.UserKanukaSettings.UserKeysPath = oldKeysPath }()

		projectName := "my-project"
		projectUUID := "550e8400-e29b-41d4-a716-446655440000"

		// Create legacy key files.
		if err := os.WriteFile(filepath.Join(tempDir, projectName), []byte("private-key"), 0600); err != nil {
			t.Fatalf("Failed to create private key: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tempDir, projectName+".pub"), []byte("public-key"), 0600); err != nil {
			t.Fatalf("Failed to create public key: %v", err)
		}

		err := configs.MigrateUserKeys(projectName, projectUUID)
		if err != nil {
			t.Fatalf("MigrateUserKeys failed: %v", err)
		}

		// Verify old files were removed.
		if _, err := os.Stat(filepath.Join(tempDir, projectName)); !os.IsNotExist(err) {
			t.Fatal("Old private key should have been removed")
		}
		if _, err := os.Stat(filepath.Join(tempDir, projectName+".pub")); !os.IsNotExist(err) {
			t.Fatal("Old public key should have been removed")
		}

		// Verify new directory structure exists.
		keyDir := filepath.Join(tempDir, projectUUID)
		if _, err := os.Stat(keyDir); os.IsNotExist(err) {
			t.Fatal("New key directory was not created")
		}
		if _, err := os.Stat(filepath.Join(keyDir, "privkey")); os.IsNotExist(err) {
			t.Fatal("New private key was not created")
		}
		if _, err := os.Stat(filepath.Join(keyDir, "pubkey.pub")); os.IsNotExist(err) {
			t.Fatal("New public key was not created")
		}
	})

	t.Run("DoesNotOverwriteExistingKeys", func(t *testing.T) {
		tempDir := t.TempDir()
		oldKeysPath := configs.UserKanukaSettings.UserKeysPath
		configs.UserKanukaSettings.UserKeysPath = tempDir
		defer func() { configs.UserKanukaSettings.UserKeysPath = oldKeysPath }()

		projectName := "my-project"
		projectUUID := "550e8400-e29b-41d4-a716-446655440000"

		// Create legacy key.
		if err := os.WriteFile(filepath.Join(tempDir, projectName), []byte("old-private"), 0600); err != nil {
			t.Fatalf("Failed to create private key: %v", err)
		}

		// Create existing directory structure that should NOT be overwritten.
		keyDir := filepath.Join(tempDir, projectUUID)
		if err := os.MkdirAll(keyDir, 0700); err != nil {
			t.Fatalf("Failed to create key directory: %v", err)
		}
		if err := os.WriteFile(filepath.Join(keyDir, "privkey"), []byte("new-private"), 0600); err != nil {
			t.Fatalf("Failed to create new private key: %v", err)
		}

		err := configs.MigrateUserKeys(projectName, projectUUID)
		if err != nil {
			t.Fatalf("MigrateUserKeys failed: %v", err)
		}

		// Verify new key was NOT overwritten.
		content, err := os.ReadFile(filepath.Join(keyDir, "privkey"))
		if err != nil {
			t.Fatalf("Failed to read new private key: %v", err)
		}
		if string(content) != "new-private" {
			t.Fatal("New private key was overwritten")
		}
	})
}
