package configs

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateUserUUID(t *testing.T) {
	uuid := GenerateUserUUID()
	if uuid == "" {
		t.Fatal("GenerateUserUUID returned empty string")
	}

	if len(uuid) != 36 {
		t.Fatalf("Expected UUID length 36, got %d", len(uuid))
	}
}

func TestGenerateProjectUUID(t *testing.T) {
	uuid := GenerateProjectUUID()
	if uuid == "" {
		t.Fatal("GenerateProjectUUID returned empty string")
	}

	if len(uuid) != 36 {
		t.Fatalf("Expected UUID length 36, got %d", len(uuid))
	}
}

func TestSaveAndLoadUserConfig(t *testing.T) {
	tempDir := t.TempDir()
	oldUserConfigsPath := UserKanukaSettings.UserConfigsPath
	UserKanukaSettings.UserConfigsPath = tempDir
	defer func() {
		UserKanukaSettings.UserConfigsPath = oldUserConfigsPath
	}()

	config := &UserConfig{
		User: User{
			Email: "test@example.com",
			UUID:  "test-uuid-123",
		},
		Projects: map[string]string{
			"project-uuid-1": "device-1",
			"project-uuid-2": "device-2",
		},
	}

	err := SaveUserConfig(config)
	if err != nil {
		t.Fatalf("SaveUserConfig failed: %v", err)
	}

	loadedConfig, err := LoadUserConfig()
	if err != nil {
		t.Fatalf("LoadUserConfig failed: %v", err)
	}

	if loadedConfig.User.Email != config.User.Email {
		t.Errorf("Expected Email %q, got %q", config.User.Email, loadedConfig.User.Email)
	}

	if loadedConfig.User.UUID != config.User.UUID {
		t.Errorf("Expected UUID %q, got %q", config.User.UUID, loadedConfig.User.UUID)
	}

	if len(loadedConfig.Projects) != len(config.Projects) {
		t.Errorf("Expected %d projects, got %d", len(config.Projects), len(loadedConfig.Projects))
	}
}

func TestLoadUserConfigNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	oldUserConfigsPath := UserKanukaSettings.UserConfigsPath
	UserKanukaSettings.UserConfigsPath = tempDir
	defer func() {
		UserKanukaSettings.UserConfigsPath = oldUserConfigsPath
	}()

	config, err := LoadUserConfig()
	if err != nil {
		t.Fatalf("LoadUserConfig failed: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config to not be nil")
	}

	if config.User.UUID != "" {
		t.Errorf("Expected empty UUID, got %q", config.User.UUID)
	}
}

func TestEnsureUserConfigCreatesUUID(t *testing.T) {
	tempDir := t.TempDir()
	oldUserConfigsPath := UserKanukaSettings.UserConfigsPath
	UserKanukaSettings.UserConfigsPath = tempDir
	defer func() {
		UserKanukaSettings.UserConfigsPath = oldUserConfigsPath
	}()

	config, err := EnsureUserConfig()
	if err != nil {
		t.Fatalf("EnsureUserConfig failed: %v", err)
	}

	if config.User.UUID == "" {
		t.Fatal("EnsureUserConfig did not generate UUID")
	}

	loadedConfig, err := LoadUserConfig()
	if err != nil {
		t.Fatalf("LoadUserConfig failed: %v", err)
	}

	if loadedConfig.User.UUID != config.User.UUID {
		t.Errorf("UUID mismatch: expected %q, got %q", config.User.UUID, loadedConfig.User.UUID)
	}
}

func TestSaveAndLoadProjectConfig(t *testing.T) {
	tempDir := t.TempDir()
	oldProjectPath := ProjectKanukaSettings.ProjectPath
	oldProjectPublicKeyPath := ProjectKanukaSettings.ProjectPublicKeyPath
	oldProjectSecretsPath := ProjectKanukaSettings.ProjectSecretsPath
	ProjectKanukaSettings.ProjectPath = tempDir
	ProjectKanukaSettings.ProjectPublicKeyPath = filepath.Join(tempDir, ".kanuka", "public_keys")
	ProjectKanukaSettings.ProjectSecretsPath = filepath.Join(tempDir, ".kanuka", "secrets")
	defer func() {
		ProjectKanukaSettings.ProjectPath = oldProjectPath
		ProjectKanukaSettings.ProjectPublicKeyPath = oldProjectPublicKeyPath
		ProjectKanukaSettings.ProjectSecretsPath = oldProjectSecretsPath
	}()

	if err := os.MkdirAll(filepath.Join(tempDir, ".kanuka"), 0700); err != nil {
		t.Fatalf("Failed to create .kanuka directory: %v", err)
	}

	config := &ProjectConfig{
		Project: Project{
			UUID: "project-uuid-123",
			Name: "test-project",
		},
		Users: map[string]string{
			"user-uuid-1": "alice@example.com",
			"user-uuid-2": "bob@example.com",
		},
		Devices: map[string]DeviceConfig{
			"user-uuid-1": {
				Email:     "alice@example.com",
				Name:      "macbook",
				CreatedAt: time.Date(2025, 1, 6, 10, 0, 0, 0, time.UTC),
			},
		},
	}

	err := SaveProjectConfig(config)
	if err != nil {
		t.Fatalf("SaveProjectConfig failed: %v", err)
	}

	loadedConfig, err := LoadProjectConfig()
	if err != nil {
		t.Fatalf("LoadProjectConfig failed: %v", err)
	}

	if loadedConfig.Project.UUID != config.Project.UUID {
		t.Errorf("Expected UUID %q, got %q", config.Project.UUID, loadedConfig.Project.UUID)
	}

	if loadedConfig.Project.Name != config.Project.Name {
		t.Errorf("Expected Name %q, got %q", config.Project.Name, loadedConfig.Project.Name)
	}

	if len(loadedConfig.Users) != len(config.Users) {
		t.Errorf("Expected %d users, got %d", len(config.Users), len(loadedConfig.Users))
	}

	if len(loadedConfig.Devices) != len(config.Devices) {
		t.Errorf("Expected %d devices, got %d", len(config.Devices), len(loadedConfig.Devices))
	}
}

func TestLoadProjectConfigNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	oldProjectPath := ProjectKanukaSettings.ProjectPath
	oldProjectPublicKeyPath := ProjectKanukaSettings.ProjectPublicKeyPath
	oldProjectSecretsPath := ProjectKanukaSettings.ProjectSecretsPath
	ProjectKanukaSettings.ProjectPath = tempDir
	ProjectKanukaSettings.ProjectPublicKeyPath = filepath.Join(tempDir, ".kanuka", "public_keys")
	ProjectKanukaSettings.ProjectSecretsPath = filepath.Join(tempDir, ".kanuka", "secrets")
	defer func() {
		ProjectKanukaSettings.ProjectPath = oldProjectPath
		ProjectKanukaSettings.ProjectPublicKeyPath = oldProjectPublicKeyPath
		ProjectKanukaSettings.ProjectSecretsPath = oldProjectSecretsPath
	}()

	config, err := LoadProjectConfig()
	if err != nil {
		t.Fatalf("LoadProjectConfig failed: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config to not be nil")
	}

	if config.Project.UUID != "" {
		t.Errorf("Expected empty UUID, got %q", config.Project.UUID)
	}
}
