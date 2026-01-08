package configs

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/PolarWolf314/kanuka/internal/utils"
)

type UserSettings struct {
	UserKeysPath    string
	UserConfigsPath string
	Username        string
}

type ProjectSettings struct {
	ProjectUUID          string
	ProjectName          string
	ProjectPath          string
	ProjectPublicKeyPath string
	ProjectSecretsPath   string
}

var (
	UserKanukaSettings    *UserSettings
	ProjectKanukaSettings *ProjectSettings
)

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("error getting home directory: %s", err)
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("error getting config directory: %s", err)
	}

	dataDir := os.Getenv("XDG_DATA_HOME")

	if dataDir == "" {
		dataDir = filepath.Join(homeDir, ".local", "share")
	}

	username, err := utils.GetUsername()
	if err != nil {
		log.Fatalf("error getting username: %s", err)
	}

	// This is independent of what repo you are in, so it is ok to init here
	UserKanukaSettings = &UserSettings{
		UserKeysPath:    filepath.Join(dataDir, "kanuka", "keys"),
		UserConfigsPath: filepath.Join(configDir, "kanuka"),
		Username:        username,
	}
	ProjectKanukaSettings = &ProjectSettings{
		ProjectName:          "",
		ProjectPath:          "",
		ProjectPublicKeyPath: "",
		ProjectSecretsPath:   "",
	}
}

func InitProjectSettings() error {
	projectName, err := utils.GetProjectName()
	if err != nil {
		return fmt.Errorf("error getting project name: %w", err)
	}

	projectPath, err := utils.FindProjectKanukaRoot()
	if err != nil {
		return fmt.Errorf("error getting project root: %w", err)
	}

	// Check for legacy project and migrate if needed.
	if IsLegacyProject(projectPath) {
		result, err := MigrateProject(projectPath)
		if err != nil {
			return fmt.Errorf("failed to migrate legacy project: %w", err)
		}

		// Migrate user's local keys.
		if err := MigrateUserKeys(projectName, result.ProjectUUID); err != nil {
			return fmt.Errorf("failed to migrate user keys: %w", err)
		}

		// Update user config with project UUID.
		if err := UpdateUserConfigWithProjectUUID(projectName, result.ProjectUUID); err != nil {
			return fmt.Errorf("failed to update user config: %w", err)
		}
	}

	ProjectKanukaSettings = &ProjectSettings{
		ProjectName:          projectName,
		ProjectPath:          projectPath,
		ProjectPublicKeyPath: filepath.Join(projectPath, ".kanuka", "public_keys"),
		ProjectSecretsPath:   filepath.Join(projectPath, ".kanuka", "secrets"),
	}

	userConfig, err := LoadUserConfig()
	if err != nil {
		return fmt.Errorf("error loading user config: %w", err)
	}
	GlobalUserConfig = userConfig

	return nil
}
