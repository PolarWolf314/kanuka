package configs

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type UserConfig struct {
	User     User              `toml:"user"`
	Projects map[string]string `toml:"projects"`
}

type User struct {
	Email string `toml:"email"`
	UUID  string `toml:"user_uuid"`
}

type ProjectConfig struct {
	Project Project                 `toml:"project"`
	Users   map[string]string       `toml:"users"`
	Devices map[string]DeviceConfig `toml:"devices"`
}

type Project struct {
	UUID string `toml:"project_uuid"`
	Name string `toml:"name"`
}

type DeviceConfig struct {
	Email     string    `toml:"email"`
	Name      string    `toml:"name"`
	CreatedAt time.Time `toml:"created_at"`
}

var (
	GlobalUserConfig    *UserConfig
	GlobalProjectConfig *ProjectConfig
)

// LoadUserConfig loads the user configuration from the config file.
func LoadUserConfig() (*UserConfig, error) {
	configPath := filepath.Join(UserKanukaSettings.UserConfigsPath, "config.toml")

	config := &UserConfig{
		Projects: make(map[string]string),
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil
	}

	if err := LoadTOML(configPath, config); err != nil {
		return nil, fmt.Errorf("failed to load user config: %w", err)
	}

	return config, nil
}

// SaveUserConfig saves the user configuration to the config file.
func SaveUserConfig(config *UserConfig) error {
	configPath := filepath.Join(UserKanukaSettings.UserConfigsPath, "config.toml")

	if err := SaveTOML(configPath, config); err != nil {
		return fmt.Errorf("failed to save user config: %w", err)
	}

	return nil
}

// GenerateUserUUID generates a new UUID for the user.
func GenerateUserUUID() string {
	return uuid.New().String()
}

// EnsureUserConfig ensures the user configuration exists and has a UUID.
func EnsureUserConfig() (*UserConfig, error) {
	config, err := LoadUserConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load user config: %w", err)
	}

	if config.User.UUID == "" {
		config.User.UUID = GenerateUserUUID()
		if err := SaveUserConfig(config); err != nil {
			return nil, fmt.Errorf("failed to save user config: %w", err)
		}
	}

	return config, nil
}

// LoadProjectConfig loads the project configuration from the config file.
// Note: Caller should ensure InitProjectSettings is called before calling this function.
func LoadProjectConfig() (*ProjectConfig, error) {
	configPath := filepath.Join(ProjectKanukaSettings.ProjectPath, ".kanuka", "config.toml")

	config := &ProjectConfig{
		Users:   make(map[string]string),
		Devices: make(map[string]DeviceConfig),
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil
	}

	if err := LoadTOML(configPath, config); err != nil {
		return nil, fmt.Errorf("failed to load project config: %w", err)
	}

	return config, nil
}

// SaveProjectConfig saves the project configuration to the config file.
// Note: Caller should ensure InitProjectSettings is called before calling this function.
func SaveProjectConfig(config *ProjectConfig) error {
	configPath := filepath.Join(ProjectKanukaSettings.ProjectPath, ".kanuka", "config.toml")

	if err := SaveTOML(configPath, config); err != nil {
		return fmt.Errorf("failed to save project config: %w", err)
	}

	return nil
}

// GenerateProjectUUID generates a new UUID for the project.
func GenerateProjectUUID() string {
	return uuid.New().String()
}
