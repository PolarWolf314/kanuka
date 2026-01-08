package configs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MigrationResult contains information about what was migrated.
type MigrationResult struct {
	ProjectUUID      string
	MigratedUsers    []MigratedUser
	MigratedKeyFiles []string
	BackupPath       string
}

// MigratedUser contains information about a migrated user.
type MigratedUser struct {
	OldUsername string
	NewUUID     string
	Email       string
	DeviceName  string
}

// IsLegacyProject checks if a project uses the old username-based file naming.
// A legacy project has .kanuka/public_keys/*.pub files but no .kanuka/config.toml.
func IsLegacyProject(projectPath string) bool {
	if projectPath == "" {
		return false
	}

	// If config.toml exists, it's not a legacy project.
	configPath := filepath.Join(projectPath, ".kanuka", "config.toml")
	if _, err := os.Stat(configPath); err == nil {
		return false
	}

	// Check if there are any .pub files in public_keys directory.
	publicKeysDir := filepath.Join(projectPath, ".kanuka", "public_keys")
	entries, err := os.ReadDir(publicKeysDir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".pub") {
			return true
		}
	}

	return false
}

// IsLegacyUserKey checks if a key file uses the old project-name-based naming.
// Legacy keys are named after the project directory name, not UUIDs.
func IsLegacyUserKey(keyName string) bool {
	// UUIDs have a specific format: 8-4-4-4-12 hex chars.
	// If the key name is not a valid UUID, it's a legacy key.
	_, err := uuid.Parse(keyName)
	return err != nil
}

// MigrateProject performs a full migration of a legacy project.
// It creates a backup, generates UUIDs, renames files, and creates config.toml.
func MigrateProject(projectPath string) (*MigrationResult, error) {
	if projectPath == "" {
		return nil, fmt.Errorf("project path is empty")
	}

	if !IsLegacyProject(projectPath) {
		return nil, fmt.Errorf("project is not a legacy project")
	}

	result := &MigrationResult{}

	// Create backup.
	backupPath, err := createBackup(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}
	result.BackupPath = backupPath

	// Generate project UUID.
	projectUUID := GenerateProjectUUID()
	result.ProjectUUID = projectUUID

	// Create project config.
	projectName := filepath.Base(projectPath)
	projectConfig := &ProjectConfig{
		Project: Project{
			UUID: projectUUID,
			Name: projectName,
		},
		Users:   make(map[string]string),
		Devices: make(map[string]DeviceConfig),
	}

	// Migrate user files.
	migratedUsers, err := migrateUserFiles(projectPath, projectConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate user files: %w", err)
	}
	result.MigratedUsers = migratedUsers

	// Save project config.
	// Temporarily set ProjectPath for SaveProjectConfig to work.
	originalProjectPath := ProjectKanukaSettings.ProjectPath
	ProjectKanukaSettings.ProjectPath = projectPath
	if err := SaveProjectConfig(projectConfig); err != nil {
		ProjectKanukaSettings.ProjectPath = originalProjectPath
		return nil, fmt.Errorf("failed to save project config: %w", err)
	}
	ProjectKanukaSettings.ProjectPath = originalProjectPath

	return result, nil
}

// createBackup creates a backup of the .kanuka directory.
func createBackup(projectPath string) (string, error) {
	kanukaDir := filepath.Join(projectPath, ".kanuka")
	backupDir := filepath.Join(projectPath, ".kanuka-backup-"+time.Now().Format("20060102-150405"))

	// Copy directory.
	if err := copyDir(kanukaDir, backupDir); err != nil {
		return "", fmt.Errorf("failed to copy directory: %w", err)
	}

	return backupDir, nil
}

// copyDir recursively copies a directory.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file.
func copyFile(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, srcInfo.Mode())
}

// migrateUserFiles renames user files from username-based to UUID-based naming.
func migrateUserFiles(projectPath string, projectConfig *ProjectConfig) ([]MigratedUser, error) {
	publicKeysDir := filepath.Join(projectPath, ".kanuka", "public_keys")
	secretsDir := filepath.Join(projectPath, ".kanuka", "secrets")

	var migratedUsers []MigratedUser

	// Find all .pub files and migrate them.
	entries, err := os.ReadDir(publicKeysDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read public keys directory: %w", err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".pub") {
			continue
		}

		oldUsername := strings.TrimSuffix(entry.Name(), ".pub")

		// Skip if it's already a UUID.
		if !IsLegacyUserKey(oldUsername) {
			continue
		}

		// Generate new UUID for this user.
		newUUID := GenerateUserUUID()

		// Rename public key file.
		oldPubPath := filepath.Join(publicKeysDir, oldUsername+".pub")
		newPubPath := filepath.Join(publicKeysDir, newUUID+".pub")
		if err := os.Rename(oldPubPath, newPubPath); err != nil {
			return nil, fmt.Errorf("failed to rename public key %s: %w", oldUsername, err)
		}

		// Rename .kanuka file if it exists.
		oldKanukaPath := filepath.Join(secretsDir, oldUsername+".kanuka")
		newKanukaPath := filepath.Join(secretsDir, newUUID+".kanuka")
		if _, err := os.Stat(oldKanukaPath); err == nil {
			if err := os.Rename(oldKanukaPath, newKanukaPath); err != nil {
				return nil, fmt.Errorf("failed to rename kanuka file %s: %w", oldUsername, err)
			}
		}

		// Add to project config.
		// For legacy users, we don't know their email, so we use username as placeholder.
		email := oldUsername + "@unknown.local"
		projectConfig.Users[newUUID] = email
		projectConfig.Devices[newUUID] = DeviceConfig{
			Email:     email,
			Name:      "migrated-device",
			CreatedAt: time.Now().UTC(),
		}

		migratedUsers = append(migratedUsers, MigratedUser{
			OldUsername: oldUsername,
			NewUUID:     newUUID,
			Email:       email,
			DeviceName:  "migrated-device",
		})
	}

	return migratedUsers, nil
}

// MigrateUserKeys migrates the user's local private keys from legacy formats to the new directory structure.
// Supported migrations:
// 1. From project-name based files: {keysDir}/{projectName}, {keysDir}/{projectName}.pub
// 2. From UUID-based flat files: {keysDir}/{projectUUID}, {keysDir}/{projectUUID}.pub
// To new structure: {keysDir}/{projectUUID}/privkey, {keysDir}/{projectUUID}/pubkey.pub.
func MigrateUserKeys(projectName, projectUUID string) error {
	keysDir := UserKanukaSettings.UserKeysPath

	// Check if already migrated to new directory structure.
	newKeyDir := GetKeyDirPath(projectUUID)
	newPrivateKeyPath := GetPrivateKeyPath(projectUUID)
	newPublicKeyPath := GetPublicKeyPath(projectUUID)

	if _, err := os.Stat(newPrivateKeyPath); err == nil {
		// New key already exists in directory structure, migration complete.
		return nil
	}

	// Try legacy project-name based keys first.
	oldPrivateKeyPath := filepath.Join(keysDir, projectName)
	oldPublicKeyPath := filepath.Join(keysDir, projectName+".pub")

	// Check if UUID-based flat files exist (intermediate migration state).
	uuidPrivateKeyPath := filepath.Join(keysDir, projectUUID)
	uuidPublicKeyPath := filepath.Join(keysDir, projectUUID+".pub")

	var sourcePrivateKey, sourcePublicKey string
	var isUUIDFlatFile bool

	// Determine source paths.
	if _, err := os.Stat(oldPrivateKeyPath); err == nil {
		sourcePrivateKey = oldPrivateKeyPath
		sourcePublicKey = oldPublicKeyPath
	} else if info, err := os.Stat(uuidPrivateKeyPath); err == nil && !info.IsDir() {
		// UUID-based flat file exists (not a directory).
		sourcePrivateKey = uuidPrivateKeyPath
		sourcePublicKey = uuidPublicKeyPath
		isUUIDFlatFile = true
	} else {
		// No legacy keys to migrate.
		return nil
	}

	// For UUID flat files, we need to handle the special case where the file
	// is at the same path as the directory we want to create.
	if isUUIDFlatFile {
		// Move the flat files to temporary locations first.
		tempPrivateKey := sourcePrivateKey + ".migrating"
		tempPublicKey := sourcePublicKey + ".migrating"

		if _, err := os.Stat(sourcePrivateKey); err == nil {
			if err := os.Rename(sourcePrivateKey, tempPrivateKey); err != nil {
				return fmt.Errorf("failed to move private key to temp location: %w", err)
			}
			sourcePrivateKey = tempPrivateKey
		}

		if _, err := os.Stat(sourcePublicKey); err == nil {
			if err := os.Rename(sourcePublicKey, tempPublicKey); err != nil {
				return fmt.Errorf("failed to move public key to temp location: %w", err)
			}
			sourcePublicKey = tempPublicKey
		}
	}

	// Create new key directory.
	if err := os.MkdirAll(newKeyDir, 0700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	// Move private key if it exists.
	if _, err := os.Stat(sourcePrivateKey); err == nil {
		if err := os.Rename(sourcePrivateKey, newPrivateKeyPath); err != nil {
			return fmt.Errorf("failed to migrate private key: %w", err)
		}
	}

	// Move public key if it exists.
	if _, err := os.Stat(sourcePublicKey); err == nil {
		if err := os.Rename(sourcePublicKey, newPublicKeyPath); err != nil {
			return fmt.Errorf("failed to migrate public key: %w", err)
		}
	}

	return nil
}

// UpdateUserConfigWithProjectUUID updates the user's config.toml to use project UUID instead of name.
func UpdateUserConfigWithProjectUUID(projectName, projectUUID string) error {
	userConfig, err := LoadUserConfig()
	if err != nil {
		return fmt.Errorf("failed to load user config: %w", err)
	}

	// Check if project exists by name (old format used project name as key).
	if existingEntry, exists := userConfig.Projects[projectName]; exists {
		// Move to UUID-based key, preserving the device name and setting project name.
		delete(userConfig.Projects, projectName)
		userConfig.Projects[projectUUID] = UserProjectEntry{
			DeviceName:  existingEntry.DeviceName,
			ProjectName: projectName, // Use the old project name as the display name.
		}

		if err := SaveUserConfig(userConfig); err != nil {
			return fmt.Errorf("failed to save user config: %w", err)
		}
	}

	return nil
}
