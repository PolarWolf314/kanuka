package configs

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestEdgeCases contains edge case tests for the config system.
func TestEdgeCases(t *testing.T) {
	t.Run("TwoUsersWithSameEmail", func(t *testing.T) {
		testTwoUsersWithSameEmail(t)
	})

	t.Run("SameUserOnMultipleDevices", func(t *testing.T) {
		testSameUserOnMultipleDevices(t)
	})

	t.Run("DeviceNameCollision", func(t *testing.T) {
		testDeviceNameCollision(t)
	})

	t.Run("MalformedConfigRecovery", func(t *testing.T) {
		testMalformedConfigRecovery(t)
	})

	t.Run("EmptyUserEmail", func(t *testing.T) {
		testEmptyUserEmail(t)
	})

	t.Run("SpecialCharactersInProjectName", func(t *testing.T) {
		testSpecialCharactersInProjectName(t)
	})
}

// testTwoUsersWithSameEmail tests that two users with the same email can have different UUIDs.
func testTwoUsersWithSameEmail(t *testing.T) {
	config := &ProjectConfig{
		Users: map[string]string{
			"uuid-alice-1": "alice@example.com",
			"uuid-alice-2": "alice@example.com", // Same email, different UUID (different device).
			"uuid-bob":     "bob@example.com",
		},
		Devices: map[string]DeviceConfig{
			"uuid-alice-1": {Email: "alice@example.com", Name: "macbook", CreatedAt: time.Now()},
			"uuid-alice-2": {Email: "alice@example.com", Name: "desktop", CreatedAt: time.Now()},
			"uuid-bob":     {Email: "bob@example.com", Name: "laptop", CreatedAt: time.Now()},
		},
	}

	// GetUserUUIDByEmail should return one of the UUIDs.
	uuid, found := config.GetUserUUIDByEmail("alice@example.com")
	if !found {
		t.Fatal("Expected to find user by email")
	}
	if uuid != "uuid-alice-1" && uuid != "uuid-alice-2" {
		t.Errorf("Expected uuid-alice-1 or uuid-alice-2, got %q", uuid)
	}

	// GetAllUserUUIDsByEmail should return both UUIDs.
	uuids := config.GetAllUserUUIDsByEmail("alice@example.com")
	if len(uuids) != 2 {
		t.Fatalf("Expected 2 UUIDs for alice, got %d", len(uuids))
	}

	// Verify both UUIDs are in the result.
	foundAlice1 := false
	foundAlice2 := false
	for _, u := range uuids {
		if u == "uuid-alice-1" {
			foundAlice1 = true
		}
		if u == "uuid-alice-2" {
			foundAlice2 = true
		}
	}
	if !foundAlice1 || !foundAlice2 {
		t.Errorf("Expected both alice UUIDs, got %v", uuids)
	}

	// GetDevicesByEmail should return both devices.
	devices := config.GetDevicesByEmail("alice@example.com")
	if len(devices) != 2 {
		t.Fatalf("Expected 2 devices for alice, got %d", len(devices))
	}

	// Verify we can look up by email+device name.
	uuid, found = config.GetUserUUIDByEmailAndDevice("alice@example.com", "macbook")
	if !found {
		t.Fatal("Expected to find alice's macbook")
	}
	if uuid != "uuid-alice-1" {
		t.Errorf("Expected uuid-alice-1 for macbook, got %q", uuid)
	}

	uuid, found = config.GetUserUUIDByEmailAndDevice("alice@example.com", "desktop")
	if !found {
		t.Fatal("Expected to find alice's desktop")
	}
	if uuid != "uuid-alice-2" {
		t.Errorf("Expected uuid-alice-2 for desktop, got %q", uuid)
	}
}

// testSameUserOnMultipleDevices tests managing a user across multiple devices.
func testSameUserOnMultipleDevices(t *testing.T) {
	config := &ProjectConfig{
		Users: map[string]string{
			"uuid-dev-1": "alice@example.com",
			"uuid-dev-2": "alice@example.com",
			"uuid-dev-3": "alice@example.com",
		},
		Devices: map[string]DeviceConfig{
			"uuid-dev-1": {Email: "alice@example.com", Name: "macbook-pro", CreatedAt: time.Now()},
			"uuid-dev-2": {Email: "alice@example.com", Name: "imac", CreatedAt: time.Now()},
			"uuid-dev-3": {Email: "alice@example.com", Name: "mac-mini", CreatedAt: time.Now()},
		},
	}

	// Verify all device names can be retrieved.
	names := config.GetDeviceNamesByEmail("alice@example.com")
	if len(names) != 3 {
		t.Fatalf("Expected 3 device names, got %d", len(names))
	}

	// Verify HasOtherDevicesForEmail works.
	if !config.HasOtherDevicesForEmail("alice@example.com", "uuid-dev-1") {
		t.Fatal("Alice should have other devices besides uuid-dev-1")
	}

	// Remove one device.
	config.RemoveDevice("uuid-dev-1")

	// Verify alice still has other devices.
	if !config.HasOtherDevicesForEmail("alice@example.com", "uuid-dev-2") {
		t.Fatal("Alice should have other devices besides uuid-dev-2")
	}

	// Verify the removed device is gone.
	if _, exists := config.Users["uuid-dev-1"]; exists {
		t.Fatal("uuid-dev-1 should be removed from Users")
	}
	if _, exists := config.Devices["uuid-dev-1"]; exists {
		t.Fatal("uuid-dev-1 should be removed from Devices")
	}

	// Verify remaining device count.
	uuids := config.GetAllUserUUIDsByEmail("alice@example.com")
	if len(uuids) != 2 {
		t.Fatalf("Expected 2 remaining UUIDs, got %d", len(uuids))
	}
}

// testDeviceNameCollision tests handling of device name collisions.
func testDeviceNameCollision(t *testing.T) {
	config := &ProjectConfig{
		Devices: map[string]DeviceConfig{
			"uuid-1": {Email: "alice@example.com", Name: "macbook", CreatedAt: time.Now()},
			"uuid-2": {Email: "bob@example.com", Name: "macbook", CreatedAt: time.Now()}, // Same device name, different user.
		},
	}

	// Same device name for different users should not be a collision.
	if config.IsDeviceNameTakenByEmail("carol@example.com", "macbook") {
		t.Fatal("macbook should not be taken for carol (different user)")
	}

	// Same device name for same user IS a collision.
	if !config.IsDeviceNameTakenByEmail("alice@example.com", "macbook") {
		t.Fatal("macbook should be taken for alice")
	}

	// Different device name for same user is not a collision.
	if config.IsDeviceNameTakenByEmail("alice@example.com", "desktop") {
		t.Fatal("desktop should not be taken for alice")
	}
}

// testMalformedConfigRecovery tests handling of malformed config files.
func testMalformedConfigRecovery(t *testing.T) {
	tempDir := t.TempDir()
	oldProjectPath := ProjectKanukaSettings.ProjectPath
	ProjectKanukaSettings.ProjectPath = tempDir
	defer func() { ProjectKanukaSettings.ProjectPath = oldProjectPath }()

	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if err := os.MkdirAll(kanukaDir, 0700); err != nil {
		t.Fatalf("Failed to create .kanuka directory: %v", err)
	}

	// Write malformed TOML.
	configPath := filepath.Join(kanukaDir, "config.toml")
	malformedContent := `[project
uuid = "broken"
`
	if err := os.WriteFile(configPath, []byte(malformedContent), 0600); err != nil {
		t.Fatalf("Failed to create malformed config: %v", err)
	}

	// Loading should fail with an error.
	_, err := LoadProjectConfig()
	if err == nil {
		t.Fatal("Expected error when loading malformed config")
	}

	// Write valid TOML to recover.
	validContent := `[project]
project_uuid = "recovered-uuid"
name = "test-project"
`
	if err := os.WriteFile(configPath, []byte(validContent), 0600); err != nil {
		t.Fatalf("Failed to create valid config: %v", err)
	}

	// Loading should now succeed.
	config, err := LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load recovered config: %v", err)
	}
	if config.Project.UUID != "recovered-uuid" {
		t.Errorf("Expected UUID 'recovered-uuid', got %q", config.Project.UUID)
	}
}

// testEmptyUserEmail tests handling of empty user email.
func testEmptyUserEmail(t *testing.T) {
	config := &ProjectConfig{
		Users: map[string]string{
			"uuid-1": "", // Empty email.
			"uuid-2": "bob@example.com",
		},
		Devices: map[string]DeviceConfig{
			"uuid-1": {Email: "", Name: "device1", CreatedAt: time.Now()},
			"uuid-2": {Email: "bob@example.com", Name: "device2", CreatedAt: time.Now()},
		},
	}

	// Looking up empty email should find the user.
	uuid, found := config.GetUserUUIDByEmail("")
	if !found {
		t.Fatal("Expected to find user with empty email")
	}
	if uuid != "uuid-1" {
		t.Errorf("Expected uuid-1, got %q", uuid)
	}

	// GetDevicesByEmail should work with empty email.
	devices := config.GetDevicesByEmail("")
	if len(devices) != 1 {
		t.Fatalf("Expected 1 device with empty email, got %d", len(devices))
	}
}

// testSpecialCharactersInProjectName tests handling of special characters in project names.
func testSpecialCharactersInProjectName(t *testing.T) {
	tempDir := t.TempDir()
	oldProjectPath := ProjectKanukaSettings.ProjectPath
	ProjectKanukaSettings.ProjectPath = tempDir
	defer func() { ProjectKanukaSettings.ProjectPath = oldProjectPath }()

	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if err := os.MkdirAll(kanukaDir, 0700); err != nil {
		t.Fatalf("Failed to create .kanuka directory: %v", err)
	}

	// Test various special characters in project name.
	specialNames := []string{
		"my-project",
		"my_project",
		"my.project",
		"Project with Spaces",
		"project@2024",
		"日本語プロジェクト", // Japanese characters.
	}

	for _, name := range specialNames {
		t.Run(name, func(t *testing.T) {
			config := &ProjectConfig{
				Project: Project{
					UUID: GenerateProjectUUID(),
					Name: name,
				},
				Users:   make(map[string]string),
				Devices: make(map[string]DeviceConfig),
			}

			// Save config.
			if err := SaveProjectConfig(config); err != nil {
				t.Fatalf("Failed to save config with name %q: %v", name, err)
			}

			// Load config.
			loadedConfig, err := LoadProjectConfig()
			if err != nil {
				t.Fatalf("Failed to load config with name %q: %v", name, err)
			}

			// Verify name was preserved.
			if loadedConfig.Project.Name != name {
				t.Errorf("Name not preserved: expected %q, got %q", name, loadedConfig.Project.Name)
			}
		})
	}
}

// TestMigrationEdgeCases tests edge cases in the migration logic.
func TestMigrationEdgeCases(t *testing.T) {
	t.Run("MigrationWithNoExistingKeys", func(t *testing.T) {
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

		// Create a .pub file without corresponding .kanuka file.
		if err := os.WriteFile(filepath.Join(publicKeysDir, "alice.pub"), []byte("key"), 0600); err != nil {
			t.Fatalf("Failed to create alice.pub: %v", err)
		}

		// Migration should still work.
		result, err := MigrateProject(tempDir)
		if err != nil {
			t.Fatalf("MigrateProject failed: %v", err)
		}

		if len(result.MigratedUsers) != 1 {
			t.Fatalf("Expected 1 migrated user, got %d", len(result.MigratedUsers))
		}
	})

	t.Run("MigrationSkipsUUIDFiles", func(t *testing.T) {
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

		// Create one legacy file and one UUID file.
		if err := os.WriteFile(filepath.Join(publicKeysDir, "alice.pub"), []byte("key"), 0600); err != nil {
			t.Fatalf("Failed to create alice.pub: %v", err)
		}
		// This UUID file should be skipped.
		if err := os.WriteFile(filepath.Join(publicKeysDir, "550e8400-e29b-41d4-a716-446655440000.pub"), []byte("key"), 0600); err != nil {
			t.Fatalf("Failed to create UUID.pub: %v", err)
		}

		// Migration should only migrate alice.
		result, err := MigrateProject(tempDir)
		if err != nil {
			t.Fatalf("MigrateProject failed: %v", err)
		}

		if len(result.MigratedUsers) != 1 {
			t.Fatalf("Expected 1 migrated user (alice only), got %d", len(result.MigratedUsers))
		}

		if result.MigratedUsers[0].OldUsername != "alice" {
			t.Errorf("Expected 'alice' to be migrated, got %q", result.MigratedUsers[0].OldUsername)
		}
	})

	t.Run("MigrationHandlesEmptyDirectories", func(t *testing.T) {
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

		// Empty public_keys directory should not be detected as legacy.
		if IsLegacyProject(tempDir) {
			t.Fatal("Empty project should not be detected as legacy")
		}
	})
}
