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
		Projects: map[string]UserProjectEntry{
			"project-uuid-1": {DeviceName: "device-1", ProjectName: "Project 1"},
			"project-uuid-2": {DeviceName: "device-2", ProjectName: "Project 2"},
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

func TestGetUserUUIDByEmail(t *testing.T) {
	config := &ProjectConfig{
		Users: map[string]string{
			"uuid-1": "alice@example.com",
			"uuid-2": "bob@example.com",
			"uuid-3": "alice@example.com", // Same email, different device.
		},
	}

	t.Run("FindsExistingEmail", func(t *testing.T) {
		uuid, found := config.GetUserUUIDByEmail("alice@example.com")
		if !found {
			t.Fatal("Expected to find user")
		}
		if uuid != "uuid-1" && uuid != "uuid-3" {
			t.Errorf("Expected uuid-1 or uuid-3, got %q", uuid)
		}
	})

	t.Run("NotFoundForNonExistentEmail", func(t *testing.T) {
		uuid, found := config.GetUserUUIDByEmail("unknown@example.com")
		if found {
			t.Fatal("Expected not to find user")
		}
		if uuid != "" {
			t.Errorf("Expected empty UUID, got %q", uuid)
		}
	})
}

func TestGetAllUserUUIDsByEmail(t *testing.T) {
	config := &ProjectConfig{
		Users: map[string]string{
			"uuid-1": "alice@example.com",
			"uuid-2": "bob@example.com",
			"uuid-3": "alice@example.com",
		},
	}

	t.Run("FindsAllMatchingUUIDs", func(t *testing.T) {
		uuids := config.GetAllUserUUIDsByEmail("alice@example.com")
		if len(uuids) != 2 {
			t.Fatalf("Expected 2 UUIDs, got %d", len(uuids))
		}
	})

	t.Run("ReturnsEmptyForNonExistentEmail", func(t *testing.T) {
		uuids := config.GetAllUserUUIDsByEmail("unknown@example.com")
		if len(uuids) != 0 {
			t.Errorf("Expected 0 UUIDs, got %d", len(uuids))
		}
	})

	t.Run("FindsSingleUUID", func(t *testing.T) {
		uuids := config.GetAllUserUUIDsByEmail("bob@example.com")
		if len(uuids) != 1 {
			t.Fatalf("Expected 1 UUID, got %d", len(uuids))
		}
		if uuids[0] != "uuid-2" {
			t.Errorf("Expected uuid-2, got %q", uuids[0])
		}
	})
}

func TestGetDevicesByEmail(t *testing.T) {
	config := &ProjectConfig{
		Devices: map[string]DeviceConfig{
			"uuid-1": {Email: "alice@example.com", Name: "macbook"},
			"uuid-2": {Email: "bob@example.com", Name: "laptop"},
			"uuid-3": {Email: "alice@example.com", Name: "desktop"},
		},
	}

	t.Run("FindsAllDevicesForEmail", func(t *testing.T) {
		devices := config.GetDevicesByEmail("alice@example.com")
		if len(devices) != 2 {
			t.Fatalf("Expected 2 devices, got %d", len(devices))
		}
	})

	t.Run("ReturnsEmptyForNonExistentEmail", func(t *testing.T) {
		devices := config.GetDevicesByEmail("unknown@example.com")
		if len(devices) != 0 {
			t.Errorf("Expected 0 devices, got %d", len(devices))
		}
	})
}

func TestGetUserUUIDByEmailAndDevice(t *testing.T) {
	config := &ProjectConfig{
		Devices: map[string]DeviceConfig{
			"uuid-1": {Email: "alice@example.com", Name: "macbook"},
			"uuid-2": {Email: "alice@example.com", Name: "desktop"},
			"uuid-3": {Email: "bob@example.com", Name: "laptop"},
		},
	}

	t.Run("FindsCorrectDevice", func(t *testing.T) {
		uuid, found := config.GetUserUUIDByEmailAndDevice("alice@example.com", "desktop")
		if !found {
			t.Fatal("Expected to find device")
		}
		if uuid != "uuid-2" {
			t.Errorf("Expected uuid-2, got %q", uuid)
		}
	})

	t.Run("NotFoundForWrongDevice", func(t *testing.T) {
		uuid, found := config.GetUserUUIDByEmailAndDevice("alice@example.com", "laptop")
		if found {
			t.Fatal("Expected not to find device")
		}
		if uuid != "" {
			t.Errorf("Expected empty UUID, got %q", uuid)
		}
	})

	t.Run("NotFoundForWrongEmail", func(t *testing.T) {
		uuid, found := config.GetUserUUIDByEmailAndDevice("unknown@example.com", "macbook")
		if found {
			t.Fatal("Expected not to find device")
		}
		if uuid != "" {
			t.Errorf("Expected empty UUID, got %q", uuid)
		}
	})
}

func TestGetDeviceNamesByEmail(t *testing.T) {
	config := &ProjectConfig{
		Devices: map[string]DeviceConfig{
			"uuid-1": {Email: "alice@example.com", Name: "macbook"},
			"uuid-2": {Email: "alice@example.com", Name: "desktop"},
			"uuid-3": {Email: "bob@example.com", Name: "laptop"},
		},
	}

	t.Run("ReturnsAllDeviceNames", func(t *testing.T) {
		names := config.GetDeviceNamesByEmail("alice@example.com")
		if len(names) != 2 {
			t.Fatalf("Expected 2 names, got %d", len(names))
		}
	})

	t.Run("ReturnsEmptyForNonExistentEmail", func(t *testing.T) {
		names := config.GetDeviceNamesByEmail("unknown@example.com")
		if len(names) != 0 {
			t.Errorf("Expected 0 names, got %d", len(names))
		}
	})
}

func TestIsDeviceNameTakenByEmail(t *testing.T) {
	config := &ProjectConfig{
		Devices: map[string]DeviceConfig{
			"uuid-1": {Email: "alice@example.com", Name: "macbook"},
			"uuid-2": {Email: "bob@example.com", Name: "macbook"},
		},
	}

	t.Run("ReturnsTrueWhenTaken", func(t *testing.T) {
		if !config.IsDeviceNameTakenByEmail("alice@example.com", "macbook") {
			t.Fatal("Expected device name to be taken")
		}
	})

	t.Run("ReturnsFalseWhenNotTaken", func(t *testing.T) {
		if config.IsDeviceNameTakenByEmail("alice@example.com", "desktop") {
			t.Fatal("Expected device name to not be taken")
		}
	})

	t.Run("ReturnsFalseForDifferentEmail", func(t *testing.T) {
		// macbook is taken by bob, but not by carol.
		if config.IsDeviceNameTakenByEmail("carol@example.com", "macbook") {
			t.Fatal("Expected device name to not be taken for different email")
		}
	})
}

func TestRemoveDevice(t *testing.T) {
	config := &ProjectConfig{
		Users: map[string]string{
			"uuid-1": "alice@example.com",
			"uuid-2": "bob@example.com",
		},
		Devices: map[string]DeviceConfig{
			"uuid-1": {Email: "alice@example.com", Name: "macbook"},
			"uuid-2": {Email: "bob@example.com", Name: "laptop"},
		},
	}

	config.RemoveDevice("uuid-1")

	if _, exists := config.Users["uuid-1"]; exists {
		t.Error("Expected uuid-1 to be removed from Users")
	}
	if _, exists := config.Devices["uuid-1"]; exists {
		t.Error("Expected uuid-1 to be removed from Devices")
	}

	// uuid-2 should still exist.
	if _, exists := config.Users["uuid-2"]; !exists {
		t.Error("Expected uuid-2 to still exist in Users")
	}
	if _, exists := config.Devices["uuid-2"]; !exists {
		t.Error("Expected uuid-2 to still exist in Devices")
	}
}

func TestRemoveDevicesByEmail(t *testing.T) {
	config := &ProjectConfig{
		Users: map[string]string{
			"uuid-1": "alice@example.com",
			"uuid-2": "alice@example.com",
			"uuid-3": "bob@example.com",
		},
		Devices: map[string]DeviceConfig{
			"uuid-1": {Email: "alice@example.com", Name: "macbook"},
			"uuid-2": {Email: "alice@example.com", Name: "desktop"},
			"uuid-3": {Email: "bob@example.com", Name: "laptop"},
		},
	}

	removedUUIDs := config.RemoveDevicesByEmail("alice@example.com")

	if len(removedUUIDs) != 2 {
		t.Fatalf("Expected 2 removed UUIDs, got %d", len(removedUUIDs))
	}

	if len(config.Users) != 1 {
		t.Errorf("Expected 1 user remaining, got %d", len(config.Users))
	}

	if len(config.Devices) != 1 {
		t.Errorf("Expected 1 device remaining, got %d", len(config.Devices))
	}

	if _, exists := config.Users["uuid-3"]; !exists {
		t.Error("Expected uuid-3 to still exist")
	}
}

func TestHasOtherDevicesForEmail(t *testing.T) {
	config := &ProjectConfig{
		Devices: map[string]DeviceConfig{
			"uuid-1": {Email: "alice@example.com", Name: "macbook"},
			"uuid-2": {Email: "alice@example.com", Name: "desktop"},
			"uuid-3": {Email: "bob@example.com", Name: "laptop"},
		},
	}

	t.Run("ReturnsTrueWhenOtherDevicesExist", func(t *testing.T) {
		if !config.HasOtherDevicesForEmail("alice@example.com", "uuid-1") {
			t.Fatal("Expected other devices to exist")
		}
	})

	t.Run("ReturnsFalseWhenNoOtherDevices", func(t *testing.T) {
		if config.HasOtherDevicesForEmail("bob@example.com", "uuid-3") {
			t.Fatal("Expected no other devices")
		}
	})

	t.Run("ReturnsFalseForNonExistentEmail", func(t *testing.T) {
		if config.HasOtherDevicesForEmail("unknown@example.com", "") {
			t.Fatal("Expected no other devices for non-existent email")
		}
	})
}
