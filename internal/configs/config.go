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

// GetUserUUIDByEmail looks up a user UUID by their email in the project config.
// Returns the UUID and true if found, empty string and false if not found.
func (pc *ProjectConfig) GetUserUUIDByEmail(email string) (string, bool) {
	for uuid, userEmail := range pc.Users {
		if userEmail == email {
			return uuid, true
		}
	}
	return "", false
}

// GetAllUserUUIDsByEmail returns all user UUIDs that match the given email.
// This handles the case where the same email might have multiple devices (UUIDs).
func (pc *ProjectConfig) GetAllUserUUIDsByEmail(email string) []string {
	var uuids []string
	for uuid, userEmail := range pc.Users {
		if userEmail == email {
			uuids = append(uuids, uuid)
		}
	}
	return uuids
}

// GetDevicesByEmail returns all devices for a given email address.
func (pc *ProjectConfig) GetDevicesByEmail(email string) map[string]DeviceConfig {
	devices := make(map[string]DeviceConfig)
	for uuid, device := range pc.Devices {
		if device.Email == email {
			devices[uuid] = device
		}
	}
	return devices
}

// GetUserUUIDByEmailAndDevice looks up a user UUID by email and device name.
// Returns the UUID and true if found, empty string and false if not found.
func (pc *ProjectConfig) GetUserUUIDByEmailAndDevice(email, deviceName string) (string, bool) {
	for uuid, device := range pc.Devices {
		if device.Email == email && device.Name == deviceName {
			return uuid, true
		}
	}
	return "", false
}
