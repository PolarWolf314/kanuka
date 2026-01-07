package configs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsLegacyProject(t *testing.T) {
	t.Run("EmptyPathReturnsFalse", func(t *testing.T) {
		if IsLegacyProject("") {
			t.Fatal("Expected false for empty path")
		}
	})

	t.Run("NonExistentPathReturnsFalse", func(t *testing.T) {
		if IsLegacyProject("/nonexistent/path") {
			t.Fatal("Expected false for non-existent path")
		}
	})

	t.Run("ProjectWithConfigTomlNotLegacy", func(t *testing.T) {
		tempDir := t.TempDir()
		kanukaDir := filepath.Join(tempDir, ".kanuka")
		publicKeysDir := filepath.Join(kanukaDir, "public_keys")

		if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
			t.Fatalf("Failed to create directories: %v", err)
		}

		// Create config.toml.
		configPath := filepath.Join(kanukaDir, "config.toml")
		if err := os.WriteFile(configPath, []byte("[project]\n"), 0600); err != nil {
			t.Fatalf("Failed to create config.toml: %v", err)
		}

		// Create a .pub file.
		pubPath := filepath.Join(publicKeysDir, "alice.pub")
		if err := os.WriteFile(pubPath, []byte("key"), 0600); err != nil {
			t.Fatalf("Failed to create .pub file: %v", err)
		}

		if IsLegacyProject(tempDir) {
			t.Fatal("Expected false for project with config.toml")
		}
	})

	t.Run("ProjectWithPubFilesNoConfigIsLegacy", func(t *testing.T) {
		tempDir := t.TempDir()
		kanukaDir := filepath.Join(tempDir, ".kanuka")
		publicKeysDir := filepath.Join(kanukaDir, "public_keys")

		if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
			t.Fatalf("Failed to create directories: %v", err)
		}

		// Create a .pub file (no config.toml).
		pubPath := filepath.Join(publicKeysDir, "alice.pub")
		if err := os.WriteFile(pubPath, []byte("key"), 0600); err != nil {
			t.Fatalf("Failed to create .pub file: %v", err)
		}

		if !IsLegacyProject(tempDir) {
			t.Fatal("Expected true for project with .pub files but no config.toml")
		}
	})

	t.Run("EmptyKanukaDirNotLegacy", func(t *testing.T) {
		tempDir := t.TempDir()
		kanukaDir := filepath.Join(tempDir, ".kanuka")
		publicKeysDir := filepath.Join(kanukaDir, "public_keys")

		if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
			t.Fatalf("Failed to create directories: %v", err)
		}

		// No .pub files.
		if IsLegacyProject(tempDir) {
			t.Fatal("Expected false for empty public_keys directory")
		}
	})
}

func TestIsLegacyUserKey(t *testing.T) {
	t.Run("ValidUUIDNotLegacy", func(t *testing.T) {
		if IsLegacyUserKey("550e8400-e29b-41d4-a716-446655440000") {
			t.Fatal("Expected false for valid UUID")
		}
	})

	t.Run("ProjectNameIsLegacy", func(t *testing.T) {
		if !IsLegacyUserKey("my-project") {
			t.Fatal("Expected true for project name")
		}
	})

	t.Run("UsernameIsLegacy", func(t *testing.T) {
		if !IsLegacyUserKey("alice") {
			t.Fatal("Expected true for username")
		}
	})

	t.Run("EmptyStringIsLegacy", func(t *testing.T) {
		if !IsLegacyUserKey("") {
			t.Fatal("Expected true for empty string")
		}
	})

	t.Run("InvalidUUIDIsLegacy", func(t *testing.T) {
		if !IsLegacyUserKey("550e8400-e29b-41d4") {
			t.Fatal("Expected true for invalid UUID")
		}
	})
}

func TestMigrateProject(t *testing.T) {
	t.Run("MigratesLegacyProject", func(t *testing.T) {
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

		result, err := MigrateProject(tempDir)
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

		// Verify backup was created.
		if result.BackupPath == "" {
			t.Fatal("Expected backup path")
		}
		if _, err := os.Stat(result.BackupPath); os.IsNotExist(err) {
			t.Fatal("Backup directory was not created")
		}

		// Verify users were migrated.
		if len(result.MigratedUsers) != 2 {
			t.Fatalf("Expected 2 migrated users, got %d", len(result.MigratedUsers))
		}

		// Verify config.toml was created.
		configPath := filepath.Join(kanukaDir, "config.toml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Fatal("config.toml was not created")
		}

		// Verify old files were renamed.
		if _, err := os.Stat(filepath.Join(publicKeysDir, "alice.pub")); !os.IsNotExist(err) {
			t.Fatal("alice.pub should have been renamed")
		}

		// Verify new UUID-named files exist.
		entries, _ := os.ReadDir(publicKeysDir)
		uuidFiles := 0
		for _, entry := range entries {
			name := strings.TrimSuffix(entry.Name(), ".pub")
			if !IsLegacyUserKey(name) {
				uuidFiles++
			}
		}
		if uuidFiles != 2 {
			t.Fatalf("Expected 2 UUID-named .pub files, got %d", uuidFiles)
		}
	})

	t.Run("FailsForNonLegacyProject", func(t *testing.T) {
		tempDir := t.TempDir()
		kanukaDir := filepath.Join(tempDir, ".kanuka")

		if err := os.MkdirAll(kanukaDir, 0755); err != nil {
			t.Fatalf("Failed to create .kanuka: %v", err)
		}

		// Create config.toml to make it non-legacy.
		configPath := filepath.Join(kanukaDir, "config.toml")
		if err := os.WriteFile(configPath, []byte("[project]\n"), 0600); err != nil {
			t.Fatalf("Failed to create config.toml: %v", err)
		}

		_, err := MigrateProject(tempDir)
		if err == nil {
			t.Fatal("Expected error for non-legacy project")
		}
	})

	t.Run("FailsForEmptyPath", func(t *testing.T) {
		_, err := MigrateProject("")
		if err == nil {
			t.Fatal("Expected error for empty path")
		}
	})
}

func TestMigrateUserKeys(t *testing.T) {
	t.Run("MigratesLegacyProjectNameKeys", func(t *testing.T) {
		tempDir := t.TempDir()
		oldKeysPath := UserKanukaSettings.UserKeysPath
		UserKanukaSettings.UserKeysPath = tempDir
		defer func() {
			UserKanukaSettings.UserKeysPath = oldKeysPath
		}()

		projectName := "my-project"
		projectUUID := "550e8400-e29b-41d4-a716-446655440000"

		// Create legacy project-name based key files.
		if err := os.WriteFile(filepath.Join(tempDir, projectName), []byte("private-key"), 0600); err != nil {
			t.Fatalf("Failed to create private key: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tempDir, projectName+".pub"), []byte("public-key"), 0600); err != nil {
			t.Fatalf("Failed to create public key: %v", err)
		}

		err := MigrateUserKeys(projectName, projectUUID)
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

	t.Run("MigratesUUIDFlatFilesToDirectory", func(t *testing.T) {
		tempDir := t.TempDir()
		oldKeysPath := UserKanukaSettings.UserKeysPath
		UserKanukaSettings.UserKeysPath = tempDir
		defer func() {
			UserKanukaSettings.UserKeysPath = oldKeysPath
		}()

		projectName := "my-project"
		projectUUID := "550e8400-e29b-41d4-a716-446655440000"

		// Create UUID-based flat key files (intermediate migration state).
		if err := os.WriteFile(filepath.Join(tempDir, projectUUID), []byte("private-key"), 0600); err != nil {
			t.Fatalf("Failed to create private key: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tempDir, projectUUID+".pub"), []byte("public-key"), 0600); err != nil {
			t.Fatalf("Failed to create public key: %v", err)
		}

		err := MigrateUserKeys(projectName, projectUUID)
		if err != nil {
			t.Fatalf("MigrateUserKeys failed: %v", err)
		}

		// Verify old flat public key file was removed.
		if _, err := os.Stat(filepath.Join(tempDir, projectUUID+".pub")); !os.IsNotExist(err) {
			t.Fatal("Old flat public key should have been removed")
		}

		// Verify the path is now a directory (not a flat file).
		keyDir := filepath.Join(tempDir, projectUUID)
		info, err := os.Stat(keyDir)
		if err != nil {
			t.Fatalf("Failed to stat key directory: %v", err)
		}
		if !info.IsDir() {
			t.Fatal("Expected path to be a directory after migration")
		}
		if _, err := os.Stat(filepath.Join(keyDir, "privkey")); os.IsNotExist(err) {
			t.Fatal("New private key was not created")
		}
		if _, err := os.Stat(filepath.Join(keyDir, "pubkey.pub")); os.IsNotExist(err) {
			t.Fatal("New public key was not created")
		}
	})

	t.Run("DoesNotOverwriteExistingDirectoryKeys", func(t *testing.T) {
		tempDir := t.TempDir()
		oldKeysPath := UserKanukaSettings.UserKeysPath
		UserKanukaSettings.UserKeysPath = tempDir
		defer func() {
			UserKanukaSettings.UserKeysPath = oldKeysPath
		}()

		projectName := "my-project"
		projectUUID := "550e8400-e29b-41d4-a716-446655440000"

		// Create legacy key files.
		if err := os.WriteFile(filepath.Join(tempDir, projectName), []byte("old-private"), 0600); err != nil {
			t.Fatalf("Failed to create legacy private key: %v", err)
		}

		// Create new directory structure that should NOT be overwritten.
		keyDir := filepath.Join(tempDir, projectUUID)
		if err := os.MkdirAll(keyDir, 0700); err != nil {
			t.Fatalf("Failed to create key directory: %v", err)
		}
		if err := os.WriteFile(filepath.Join(keyDir, "privkey"), []byte("new-private"), 0600); err != nil {
			t.Fatalf("Failed to create new private key: %v", err)
		}

		err := MigrateUserKeys(projectName, projectUUID)
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

	t.Run("HandlesNonExistentLegacyKeys", func(t *testing.T) {
		tempDir := t.TempDir()
		oldKeysPath := UserKanukaSettings.UserKeysPath
		UserKanukaSettings.UserKeysPath = tempDir
		defer func() {
			UserKanukaSettings.UserKeysPath = oldKeysPath
		}()

		// No legacy keys exist.
		err := MigrateUserKeys("nonexistent-project", "some-uuid")
		if err != nil {
			t.Fatalf("MigrateUserKeys should not fail for non-existent keys: %v", err)
		}
	})
}

func TestUpdateUserConfigWithProjectUUID(t *testing.T) {
	t.Run("UpdatesProjectKeyToUUID", func(t *testing.T) {
		tempDir := t.TempDir()
		oldConfigsPath := UserKanukaSettings.UserConfigsPath
		UserKanukaSettings.UserConfigsPath = tempDir
		defer func() {
			UserKanukaSettings.UserConfigsPath = oldConfigsPath
		}()

		projectName := "my-project"
		projectUUID := "550e8400-e29b-41d4-a716-446655440000"
		deviceName := "my-device"

		// Create user config with project name as key (old format).
		config := &UserConfig{
			User: User{Email: "test@example.com", UUID: "user-uuid"},
			Projects: map[string]UserProjectEntry{
				projectName: {DeviceName: deviceName, ProjectName: ""},
			},
		}
		if err := SaveUserConfig(config); err != nil {
			t.Fatalf("Failed to save user config: %v", err)
		}

		err := UpdateUserConfigWithProjectUUID(projectName, projectUUID)
		if err != nil {
			t.Fatalf("UpdateUserConfigWithProjectUUID failed: %v", err)
		}

		// Reload and verify.
		loadedConfig, err := LoadUserConfig()
		if err != nil {
			t.Fatalf("Failed to load user config: %v", err)
		}

		if _, exists := loadedConfig.Projects[projectName]; exists {
			t.Fatal("Project name key should have been removed")
		}

		entry := loadedConfig.Projects[projectUUID]
		if entry.DeviceName != deviceName {
			t.Fatalf("Expected device name %q, got %q", deviceName, entry.DeviceName)
		}
		if entry.ProjectName != projectName {
			t.Fatalf("Expected project name %q, got %q", projectName, entry.ProjectName)
		}
	})

	t.Run("NoChangeIfProjectNameNotFound", func(t *testing.T) {
		tempDir := t.TempDir()
		oldConfigsPath := UserKanukaSettings.UserConfigsPath
		UserKanukaSettings.UserConfigsPath = tempDir
		defer func() {
			UserKanukaSettings.UserConfigsPath = oldConfigsPath
		}()

		// Create user config without the project.
		config := &UserConfig{
			User:     User{Email: "test@example.com", UUID: "user-uuid"},
			Projects: map[string]UserProjectEntry{},
		}
		if err := SaveUserConfig(config); err != nil {
			t.Fatalf("Failed to save user config: %v", err)
		}

		err := UpdateUserConfigWithProjectUUID("nonexistent-project", "some-uuid")
		if err != nil {
			t.Fatalf("UpdateUserConfigWithProjectUUID should not fail: %v", err)
		}
	})
}
