# Kanuka Identity System Implementation Plan

## Overview

This document provides an actionable implementation plan to resolve namespace clashes in Kanuka. The solution introduces a three-layer identity system:

1. **Project Identity**: UUID-based project identification (prevents project name collisions)
2. **User Identity**: Email-based user identification with UUID fallback (prevents username collisions)
3. **Device Identity**: Device names scoped to users (enables multi-device support)

## Config Files

### User Config: `~/.config/kanuka/config.toml`

```toml
[user]
# User's email address (acts as username)
email = "alice@example.com"

# User's unique identifier (auto-generated, never change this!)
user_uuid = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"

[projects]
# Per-project device name overrides (optional)
# Format: "<project-uuid>" = "<device-name>"
# Example: "550e8400-e29b-41d4-a716-446655440000" = "workstation"
```

### Project Config: `.kanuka/config.toml`

```toml
[project]
# Unique project identifier (auto-generated on init, DO NOT CHANGE)
project_uuid = "550e8400-e29b-41d4-a716-446655440000"

# Optional friendly name (defaults to directory name)
name = "my-awesome-project"

[users]
# Map user UUID to email (for display and --user flag)
"6ba7b810-9dad-11d1-80b4-00c04fd430c8" = "alice@example.com"
"7ba7b810-9dad-11d1-80b4-00c04fd430c9" = "alice@company.com"

[devices]
# Device details (for --device flag and display)
"6ba7b810-9dad-11d1-80b4-00c04fd430c8" = {
    email = "alice@example.com",
    name = "macbook",
    created_at = "2025-01-06T10:00:00Z"
}
"7ba7b810-9dad-11d1-80b4-00c04fd430c9" = {
    email = "alice@example.com",
    name = "workstation",
    created_at = "2025-01-06T11:00:00Z"
}
```

## File Naming Changes

### Before (Current)

```
.kanuka/
  public_keys/
    alice.pub
    bob.pub
  secrets/
    alice.kanuka
    bob.kanuka

~/.local/share/kanuka/keys/
  project_name
  project_name.pub
```

### After (New)

```
.kanuka/
  config.toml
  public_keys/
    6ba7b810-9dad-11d1-80b4-00c04fd430c8.pub
    7ba7b810-9dad-11d1-80b4-00c04fd430c9.pub
  secrets/
    6ba7b810-9dad-11d1-80b4-00c04fd430c8.kanuka
    7ba7b810-9dad-11d1-80b4-00c04fd430c9.kanuka

~/.local/share/kanuka/keys/
  550e8400-e29b-41d4-a716-446655440000
  550e8400-e29b-41d4-a716-446655440000.pub
```

## Command Semantics

### Revoke Command

| Command                                                           | Meaning                                  | Confirmation                      |
| ----------------------------------------------------------------- | ---------------------------------------- | --------------------------------- |
| `kanuka secrets revoke --user alice@example.com`                  | Revoke ALL devices for alice@example.com | Yes if 2+ devices, No if 1 device |
| `kanuka secrets revoke --user alice@example.com --device macbook` | Revoke only alice@example.com's macbook  | No (explicitly specified)         |
| `kanuka secrets revoke --user alice@example.com --yes`            | Revoke ALL devices (skip confirmation)   | Never (for scripts)               |

### Examples

**User leaves team (common):**

```bash
$ kanuka secrets revoke --user alice@example.com
⚠ Warning: alice@example.com has 2 devices:
  - macbook (created: Jan 6, 2025)
  - workstation (created: Jan 6, 2025)

This will revoke ALL devices for this user.
Proceed? [y/N]: y
✓ All devices for alice@example.com have been revoked successfully!
```

**Device compromised (less common):**

```bash
$ kanuka secrets revoke --user alice@example.com --device macbook
✓ Device 'macbook' (alice@example.com) has been revoked successfully!
```

**CI/CD automation:**

```bash
$ kanuka secrets revoke --user alice@example.com --yes
✓ All devices for alice@example.com have been revoked successfully!
```

### Create Command

**Auto-generate device name from hostname:**

```bash
$ kanuka secrets create
Enter your email: alice@example.com
✓ Auto-detected device name: MacBook-Pro
✓ Keys created for alice@example.com (device: MacBook-Pro)
```

**Custom device name:**

```bash
$ kanuka secrets create --device-name "workstation"
Enter your email: alice@example.com
✓ Keys created for alice@example.com (device: workstation)
```

### List Devices Command

```bash
$ kanuka secrets list-devices
Devices in this project:
  alice@example.com
    - macbook (UUID: 6ba7b810...) - created: Jan 6, 2025
    - workstation (UUID: 7ba7b810...) - created: Jan 6, 2025
  bob@company.com
    - laptop (UUID: 8ba7b810...) - created: Jan 5, 2025
```

---

# Implementation Phases

## Phase 1: TOML Configuration Support

### Milestone 1.1: Add TOML dependency

**Tasks:**

- [x] Add `github.com/BurntSushi/toml` to `go.mod`
- [x] Create `internal/configs/toml.go` for TOML parsing
- [x] Write unit tests for TOML parsing

**Rationale:** TOML is human-readable and well-supported in Go. Needed for all configuration files.

---

### Milestone 1.2: User Config Structure

**Tasks:**

- [x] Create `UserConfig` struct in `internal/configs/config.go`
- [x] Implement `LoadUserConfig()` function
- [x] Implement `SaveUserConfig()` function
- [x] Add auto-generation of `user_uuid` on first run
- [x] Update `InitProjectSettings()` to use user config

**UserConfig struct:**

```go
type UserConfig struct {
    User struct {
        Email string
        UUID  string
    }
    Projects map[string]string // project_uuid -> device_name
}

type User struct {
    Email string
    UUID  string
}
```

**Rationale:** User config persists email and UUID across sessions. Auto-generates UUID on first run.

---

### Milestone 1.3: Project Config Structure

**Tasks:**

- [x] Create `ProjectConfig` struct in `internal/configs/config.go`
- [x] Implement `LoadProjectConfig()` function
- [x] Implement `SaveProjectConfig()` function
- [x] Implement `GenerateProjectUUID()` function

**ProjectConfig struct:**

```go
type ProjectConfig struct {
    Project struct {
        UUID      string
        Name      string
    }
    Users   map[string]string // user_uuid -> email
    Devices map[string]Device
}

type Project struct {
    UUID string
    Name string
}

type Device struct {
    Email     string
    Name      string
    CreatedAt time.Time
}
```

**Rationale:** Project config maps UUIDs to emails and stores device metadata.

---

## Phase 2: UUID-Based File Naming

### Milestone 2.1: Update Key Storage

**Tasks:**

- [x] Modify `CreateAndSaveRSAKeyPair()` in `internal/secrets/keys.go` to use project UUID
- [x] Update private key path to `~/.local/share/kanuka/keys/<project_uuid>`
- [x] Update public key path to `~/.local/share/kanuka/keys/<project_uuid>.pub`

**Before:**

```go
privateKeyPath := filepath.Join(keysDir, projectName)
publicKeyPath := privateKeyPath + ".pub"
```

**After:**

```go
projectUUID := configs.ProjectKanukaSettings.ProjectUUID
privateKeyPath := filepath.Join(keysDir, projectUUID)
publicKeyPath := privateKeyPath + ".pub"
```

**Rationale:** Project UUID prevents collisions between projects with same name.

---

### Milestone 2.2: Update Public Key Storage

**Tasks:**

- [x] Modify `CopyUserPublicKeyToProject()` in `internal/secrets/keys.go`
- [x] Update destination path to use user UUID
- [x] Update `SavePublicKeyToFile()` to use user UUID

**Before:**

```go
destKeyPath := filepath.Join(projectPublicKeyPath, username+".pub")
```

**After:**

```go
userUUID := configs.UserKanukaSettings.UserUUID
destKeyPath := filepath.Join(projectPublicKeyPath, userUUID+".pub")
```

**Rationale:** User UUID prevents collisions between users with same email/name.

---

### Milestone 2.3: Update Encrypted Symmetric Key Storage

**Tasks:**

- [x] Modify `SaveKanukaKeyToProject()` in `internal/secrets/keys.go`
- [x] Modify `GetProjectKanukaKey()` in `internal/secrets/keys.go`
- [x] Update paths to use user UUID

**Before:**

```go
destKeyPath := filepath.Join(projectSecretsPath, username+".kanuka")
userKeyFile := filepath.Join(projectSecretsPath, username+".kanuka")
```

**After:**

```go
userUUID := configs.UserKanukaSettings.UserUUID
destKeyPath := filepath.Join(projectSecretsPath, userUUID+".kanuka")
userKeyFile := filepath.Join(projectSecretsPath, userUUID+".kanuka")
```

**Rationale:** User UUID ensures unique encrypted symmetric key files.

---

## Phase 3: Email-Based Identity

### Milestone 3.1: Email Prompt on Create

**Tasks:**

- [x] Add email prompt to `secrets create` command in `cmd/secrets_create.go`
- [x] Validate email format
- [x] Save email to user config
- [x] Update project config with user mapping

**User flow:**

```bash
$ kanuka secrets create
Enter your email: alice@example.com
✓ Auto-detected device name: MacBook-Pro
✓ Keys created for alice@example.com (device: MacBook-Pro)
```

**Rationale:** Email provides naturally unique user identifier.

---

### Milestone 3.2: Update Register Command

**Tasks:**

- [x] Modify `secrets register` command in `cmd/secrets_register.go`
- [x] Update `--user` flag to accept email instead of username
- [x] Add email lookup in project config
- [x] Map email to user UUID internally

**Before:**

```bash
$ kanuka secrets register --user alice
```

**After:**

```bash
$ kanuka secrets register --user alice@example.com
✓ alice@example.com has been granted access
```

**Rationale:** Email is user-facing identifier, UUID is internal.

---

### Milestone 3.3: Update Revoke Command

**Tasks:**

- [x] Modify `secrets revoke` command in `cmd/secrets_revoke.go`
- [x] Update `--user` flag to accept email
- [x] Add `--device` flag
- [x] Add `--yes` flag
- [x] Implement email → UUID lookup
- [x] Implement smart confirmation logic

**Confirmation rules:**

- Always confirm if user has 2+ devices
- Skip confirmation if user has 1 device (unambiguous)
- Never confirm if `--yes` flag is present
- Never confirm if `--device` flag is present (explicit)

**Rationale:** Provides clear, safe user experience while supporting automation.

---

### Milestone 3.4: Update All Other Commands

**Tasks:**

- [x] Update `secrets encrypt` command to use email in output
- [x] Update `secrets decrypt` command to use email in output
- [x] Update all log messages to use email instead of username

**Before:**

```bash
✓ Files for alice have been revoked successfully!
```

**After:**

```bash
✓ Files for alice@example.com have been revoked successfully!
```

**Rationale:** Consistent user-facing language throughout.

---

## Phase 4: Device Identity Layer

### Milestone 4.1: Device Name Generation

**Tasks:**

- [x] Implement `GetHostname()` function in `internal/utils/system.go`
- [x] Implement `GenerateDeviceName()` function
- [x] Add device name to project config on create

**Device name generation logic:**

1. Get system hostname
2. Sanitize (remove special chars, convert spaces to hyphens)
3. Check for conflicts with existing devices for this user
4. If conflict, append "-2", "-3", etc.

**Rationale:** Auto-generates meaningful device names while allowing customization.

---

### Milestone 4.2: Device Name Management

**Tasks:**

- [x] Add `--device-name` flag to `secrets create` command
- [x] Implement device name uniqueness validation (per user)
- [x] Save device metadata to project config

**User flow:**

```bash
$ kanuka secrets create --device-name "workstation"
Enter your email: alice@example.com
✓ Keys created for alice@example.com (device: workstation)
```

**Rationale:** Users can customize device names for clarity.

---

### Milestone 4.3: List Devices Command (MOVED TO PHASE 10)

**NOTE:** This milestone has been moved to Phase 10 as part of the `kanuka config` command structure. Device listing is now implemented under `kanuka config list-devices` instead of `kanuka secrets list-devices`.

**New location:** Phase 10, Milestone 10.4

**Rationale:** Device listing is a configuration operation, not a secrets operation. Moving to `kanuka config` provides better CLI organization. A deprecated `secrets list-devices` command will be kept for backward compatibility.

---

### Milestone 4.4: Rename Device Command (MOVED TO PHASE 10)

**NOTE:** This milestone has been moved to Phase 10 as part of the `kanuka config` command structure. Device renaming is now implemented under `kanuka config rename-device` instead of `kanuka secrets rename-device`.

**New location:** Phase 10, Milestone 10.3

**Rationale:** Device name management is a configuration operation, not a secrets operation. Moving to `kanuka config` provides better CLI organization and separation of concerns.

---

## Phase 5: Revoke Command Refinement

### Milestone 5.1: Revoke All Devices

**Tasks:**

- [x] Implement "revoke all devices" logic in `cmd/secrets_revoke.go`
- [x] Lookup all devices for user email
- [x] Delete all public keys and encrypted symmetric keys
- [x] Update project config (remove from [devices] section)
- [x] Rotate symmetric key for remaining users
- [x] Add confirmation prompt

**Confirmation prompt:**

```bash
$ kanuka secrets revoke --user alice@example.com
⚠ Warning: alice@example.com has 2 devices:
  - macbook (created: Jan 6, 2025)
  - workstation (created: Jan 6, 2025)

This will revoke ALL devices for this user.
Proceed? [y/N]:
```

**Rationale:** Team departure scenario (most common).

---

### Milestone 5.2: Revoke Single Device

**Tasks:**

- [x] Implement "revoke single device" logic
- [x] Require both `--user` and `--device` flags
- [x] Validate device exists and belongs to user
- [x] Delete specific public key and encrypted symmetric key
- [x] Update project config (remove from [devices] section)
- [x] Keep user in [users] section (still has other devices)
- [x] Rotate symmetric key for remaining devices
- [x] No confirmation (explicitly specified)

**User flow:**

```bash
$ kanuka secrets revoke --user alice@example.com --device macbook
✓ Device 'macbook' (alice@example.com) has been revoked successfully!
```

**Rationale:** Compromised device scenario (less common, but critical).

---

### Milestone 5.3: Auto-Confirm for Single Device

**Tasks:**

- [x] Implement logic to skip confirmation if user has only 1 device
- [x] Same UX whether user has 1 or N devices

**User flow:**

```bash
$ kanuka secrets revoke --user alice@example.com
✓ Device 'macbook' (alice@example.com) has been revoked successfully!
# No confirmation - unambiguous
```

**Rationale:** Reduces friction for single-device users.

---

### Milestone 5.4: Non-Interactive Mode

**Tasks:**

- [x] Add `--yes` flag to `secrets revoke` command
- [x] Skip all confirmation prompts when flag is present
- [x] Document in command help

**User flow:**

```bash
$ kanuka secrets revoke --user alice@example.com --yes
✓ All devices for alice@example.com have been revoked successfully!
```

**Rationale:** Enables CI/CD automation.

---

## Phase 6: Project Initialization

### Milestone 6.1: Generate Project UUID

**Tasks:**

- [x] Implement project UUID generation in `secrets init` command
- [x] Create `.kanuka/config.toml` on init
- [x] Save project UUID to config
- [x] Save project name (optional)

**Project config on init:**

```toml
[project]
project_uuid = "550e8400-e29b-41d4-a716-446655440000"
name = "my-awesome-project"

[users]
[devices]
```

**Rationale:** Unique project identifier prevents collisions.

---

### Milestone 6.2: First Device Registration

**Tasks:**

- [x] Create device entry for first user in project config
- [x] Auto-generate device name from hostname
- [x] Save device metadata

**Project config after first create:**

```toml
[project]
project_uuid = "550e8400-e29b-41d4-a716-446655440000"

[users]
"6ba7b810-9dad-11d1-80b4-00c04fd430c8" = "alice@example.com"

[devices]
"6ba7b810-9dad-11d1-80b4-00c04fd430c8" = {
    email = "alice@example.com",
    name = "MacBook-Pro",
    created_at = "2025-01-06T10:00:00Z"
}
```

**Rationale:** Complete device tracking from the start.

---

## Phase 7: Migration Path

### Milestone 7.1: Detect Legacy Projects

**Tasks:**

- [x] Implement `IsLegacyProject()` function
- [x] Check for absence of `.kanuka/config.toml`
- [x] Check for old-style file naming (username-based)
- [x] Add deprecation warning on first run

**Detection logic:**

```go
func IsLegacyProject(projectPath string) bool {
    configPath := filepath.Join(projectPath, ".kanuka", "config.toml")
    if _, err := os.Stat(configPath); err == nil {
        return false
    }

    publicKeysDir := filepath.Join(projectPath, ".kanuka", "public_keys")
    entries, _ := os.ReadDir(publicKeysDir)

    for _, entry := range entries {
        if strings.Contains(entry.Name(), ".pub") {
            return true
        }
    }

    return false
}
```

**Rationale:** Graceful migration without breaking existing projects.

---

### Milestone 7.2: Generate Project UUID

**Tasks:**

- [x] Implement `MigrateProjectUUID()` function
- [x] Generate UUID for legacy project
- [x] Create `.kanuka/config.toml`
- [x] Update project settings to use UUID

**Rationale:** Enables new file naming format.

---

### Milestone 7.3: Migrate User Files

**Tasks:**

- [x] Implement `MigrateUserFiles()` function
- [x] Scan existing `.kanuka/public_keys/` directory
- [x] Scan existing `.kanuka/secrets/` directory
- [x] For each user, generate user UUID
- [x] Rename files from `<username>.pub` to `<user_uuid>.pub`
- [x] Rename files from `<username>.kanuka` to `<user_uuid>.kanuka`
- [x] Update project config with user mappings

**Migration example:**

```
Before:
.kanuka/public_keys/alice.pub
.kanuka/secrets/alice.kanuka

After:
.kanuka/public_keys/6ba7b810-9dad-11d1-80b4-00c04fd430c8.pub
.kanuka/secrets/6ba7b810-9dad-11d1-80b4-00c04fd430c8.kanuka
.kanuka/config.toml (new)
  [users]
  "6ba7b810-9dad-11d1-80b4-00c04fd430c8" = "alice@example.com"
```

**Rationale:** Preserves existing access while enabling new features.

---

### Milestone 7.4: Migrate User Keys

**Tasks:**

- [x] Implement `MigrateUserKeys()` function
- [x] Scan `~/.local/share/kanuka/keys/` directory
- [x] For each project, generate project UUID
- [x] Rename keys from `<project_name>` to `<project_uuid>`
- [x] Rename keys from `<project_name>.pub` to `<project_uuid>.pub`

**Migration example:**

```
Before:
~/.local/share/kanuka/keys/my-project
~/.local/share/kanuka/keys/my-project.pub

After:
~/.local/share/kanuka/keys/550e8400-e29b-41d4-a716-446655440000
~/.local/share/kanuka/keys/550e8400-e29b-41d4-a716-446655440000.pub
```

**Rationale:** Private keys match new naming scheme.

---

### Milestone 7.5: Auto-Migration on First Run

**Tasks:**

- [x] Add migration check to `InitProjectSettings()`
- [x] Run migration if legacy project detected
- [x] Show migration progress and success message
- [x] Create backup before migration

**User flow:**

```bash
$ kanuka secrets encrypt
⚠ Legacy project detected. Migrating to new format...
✓ Generated project UUID
✓ Migrated user files
✓ Migrated user keys
✓ Migration complete. Running command...
✓ Encryption complete
```

**Rationale:** Transparent migration with minimal user friction.

---

## Phase 8: Testing

### Milestone 8.1: Unit Tests

**Tasks:**

- [x] Test user config generation and parsing
- [x] Test project config generation and parsing
- [x] Test UUID generation
- [x] Test email validation
- [x] Test device name generation
- [x] Test email → UUID lookup
- [x] Test config migration logic

**Test files:**

- `internal/configs/config_test.go`
- `internal/configs/migration_test.go`
- `internal/configs/edge_cases_test.go`
- `internal/utils/system_test.go`

**Rationale:** Ensure core functionality works correctly.

---

### Milestone 8.2: Integration Tests

**Tasks:**

- [x] Test `secrets init` with new config format
- [x] Test `secrets create` with email prompt
- [x] Test `secrets create` with custom device name
- [x] Test `secrets register` with email
- [x] Test `secrets revoke --user` (all devices)
- [x] Test `secrets revoke --user --device` (single device)
- [x] Test `secrets revoke --user --yes` (non-interactive)
- [ ] Test `config list-devices` (moved from secrets)
- [ ] Test `config rename-device` (moved from secrets)
- [ ] Test `config set-device-name` (new command)
- [x] Test migration from legacy format
- [x] Test multi-device scenarios
- [x] Test email collision scenarios

**Test files:**

- `test/integration/config/`
- `test/integration/migration/`

**Rationale:** Verify end-to-end functionality.

---

### Milestone 8.3: Edge Case Tests

**Tasks:**

- [x] Test two users with same email on same project (should use different UUIDs)
- [x] Test same user on multiple devices
- [x] Test device name collision (per user)
- [x] Test project name collision
- [ ] Test invalid email format
- [ ] Test non-existent user
- [ ] Test non-existent device
- [x] Test malformed config files
- [x] Test migration with no existing keys

**Rationale:** Ensure robustness.

---

## Phase 9: Documentation

### Milestone 9.1: Update Command Help

**Tasks:**

- [ ] Update `secrets create` help text
- [ ] Update `secrets register` help text
- [ ] Update `secrets revoke` help text
- [ ] Add `config list-devices` help text
- [ ] Add `config rename-device` help text
- [ ] Add `config set-device-name` help text
- [ ] Add deprecation warning to `secrets list-devices`

**Example:**

```bash
$ kanuka secrets revoke --help
Revoke access to the secret store

Usage:
  kanuka secrets revoke --user <email> [--device <name>] [--yes]

Flags:
  -u, --user string      User email to revoke (required)
  -d, --device string   Device name to revoke (optional, requires --user)
  -y, --yes              Skip confirmation prompts (for automation)

Examples:
  # Revoke all devices for a user (with confirmation)
  kanuka secrets revoke --user alice@example.com

  # Revoke specific device for a user
  kanuka secrets revoke --user alice@example.com --device macbook

  # Revoke all devices without confirmation (for CI/CD)
  kanuka secrets revoke --user alice@example.com --yes
```

**Rationale:** Clear documentation for all users.

---

### Milestone 9.3: User FAQ

**Tasks:**

- [ ] Add FAQ entry for "Why do I need to provide my email?"
- [ ] Add FAQ entry for "What if I have multiple devices?"
- [ ] Add FAQ entry for "How do I revoke a compromised device?"
- [ ] Add FAQ entry for "Can I have multiple emails?"

**Rationale:** Address common questions proactively.

---

## Phase 10: Config Command Structure

### Milestone 10.1: Config Command Infrastructure

**Tasks:**

- [ ] Create `cmd/config.go` with top-level `ConfigCmd`
- [ ] Add `ConfigCmd` to root command in `main.go`
- [ ] Set up persistent flags (verbose, debug) matching `SecretsCmd`
- [ ] Add `GetConfigCmd()` and `ResetConfigState()` helper functions for testing

**Implementation:**

```go
var ConfigCmd = &cobra.Command{
    Use:   "config",
    Short: "Manage Kānuka configuration",
    Long:  `Provides commands for managing user and project configuration settings.`,
    PersistentPreRun: func(cmd *cobra.Command, args []string) {
        // Reuse logger from secrets command
        Logger = logger.Logger{
            Verbose: verbose,
            Debug:   debug,
        }
        Logger.Debugf("Initializing config command with verbose=%t, debug=%t", verbose, debug)
    },
}
```

**User flow:**

```bash
$ kanuka config --help
Manage Kānuka configuration

Usage:
  kanuka config [command]

Available Commands:
  set-device-name    Set your device name for a project
  rename-device      Rename a device in the project
  list-devices       List all devices in the project
```

**Rationale:** Separates configuration management from secrets operations, providing a cleaner CLI structure.

---

### Milestone 10.2: Set Device Name Command (User Config)

**Tasks:**

- [ ] Create `cmd/config_set_device_name.go`
- [ ] Add `--device-name` flag for the new device name
- [ ] Add optional `--project-uuid` flag (defaults to current project)
- [ ] Validate device name format (alphanumeric, hyphens, underscores)
- [ ] Save to user config's `[projects]` section
- [ ] Add confirmation if device name already exists for project

**User flow (with project UUID):**

```bash
$ kanuka config set-device-name --project-uuid 550e84... --device-name "workstation"
✓ Device name for project 550e84... set to 'workstation'
```

**User flow (in project directory):**

```bash
$ kanuka config set-device-name "workstation"
✓ Device name for project my-awesome-project set to 'workstation'
```

**User config update:**

```toml
[projects]
"550e8400-e29b-41d4-a716-446655440000" = "workstation"
```

**Rationale:** Allows users to set their preferred device name per project, stored in their local user config.

---

### Milestone 10.3: Rename Device Command (Project Config)

**Tasks:**

- [ ] Create `cmd/config_rename_device.go`
- [ ] Add `--user` flag (required, accepts email)
- [ ] Add `--new-name` flag (required)
- [ ] Add optional `--old-name` flag (if user has 1 device, auto-infer)
- [ ] Look up user UUID from project config
- [ ] Validate device exists and belongs to user
- [ ] Validate new device name is unique for this user
- [ ] Update project config's `[devices]` section
- [ ] Rotate symmetric key and re-encrypt (device name change doesn't affect access)

**User flow (single device):**

```bash
$ kanuka config rename-device --user alice@example.com "personal-macbook"
✓ Device 'macbook' renamed to 'personal-macbook' for alice@example.com
```

**User flow (multiple devices, explicit old name):**

```bash
$ kanuka config rename-device --user alice@example.com --old-name macbook "personal-macbook"
✓ Device 'macbook' renamed to 'personal-macbook' for alice@example.com
```

**Project config update:**

```toml
[devices]
"6ba7b810-9dad-11d1-80b4-00c04fd430c8" = {
    email = "alice@example.com",
    name = "personal-macbook",  # changed from "macbook"
    created_at = "2025-01-06T10:00:00Z"
}
```

**Rationale:** Enables project-wide device name changes, useful for when a device is repurposed or renamed.

---

### Milestone 10.4: Move List Devices to Config

**Tasks:**

- [ ] Create `cmd/config_list_devices.go` (move from `secrets list-devices`)
- [ ] Keep existing functionality (read from project config)
- [ ] Maintain `--user` flag for filtering
- [ ] Update all documentation references from `secrets list-devices` to `config list-devices`
- [ ] Add deprecation warning to `secrets list-devices` (keep for backward compatibility)

**User flow:**

```bash
$ kanuka config list-devices
Devices in this project:
  alice@example.com
    - personal-macbook (UUID: 6ba7b810...) - created: Jan 6, 2025
    - workstation (UUID: 7ba7b810...) - created: Jan 6, 2025
  bob@company.com
    - laptop (UUID: 8ba7b810...) - created: Jan 5, 2025
```

**Deprecation warning:**

```bash
$ kanuka secrets list-devices
⚠ Warning: 'secrets list-devices' is deprecated and will be removed in a future version.
Use 'kanuka config list-devices' instead.

Devices in this project:
  ...
```

**Rationale:** Moves device listing to config command where it logically belongs, while maintaining backward compatibility.

---

### Milestone 10.5: Update Documentation

**Tasks:**

- [ ] Create `docs/src/content/docs/guides/config.md` with config command overview
- [ ] Update `docs/src/content/docs/configuration/configuration.mdx` with config commands
- [ ] Add guide for setting device names
- [ ] Add guide for renaming devices
- [ ] Update all references from `secrets rename-device` to `config rename-device`
- [ ] Update all references from `secrets list-devices` to `config list-devices`
- [ ] Update main README with config command examples

**Example documentation:**

````markdown
## Setting Device Names

You can set your preferred device name for a project using:

```bash
kanuka config set-device-name "workstation"
```
````

This sets your device name in your local user config. Other users will still see your device name in the project config, which they can update with the rename-device command.

## Renaming Devices

To rename a device in the project (requires project access):

```bash
# Rename your only device
kanuka config rename-device --user alice@example.com "personal-macbook"

# Rename a specific device (if user has multiple)
kanuka config rename-device --user alice@example.com --old-name macbook "personal-macbook"
```

This updates the project config and is visible to all team members.

````

**Rationale:** Ensures users understand the difference between user config preferences and project config settings.

---

### Milestone 10.6: Integration Tests for Config Commands

**Tasks:**
- [ ] Test `config set-device-name` with project UUID
- [ ] Test `config set-device-name` in project directory
- [ ] Test `config set-device-name` validation (invalid names)
- [ ] Test `config rename-device` (single device, auto-infer old name)
- [ ] Test `config rename-device` (multiple devices, explicit old name)
- [ ] Test `config rename-device` validation (non-existent user, wrong device ownership)
- [ ] Test `config rename-device` name uniqueness (per user)
- [ ] Test `config list-devices` with and without `--user` filter
- [ ] Test deprecation warning for `secrets list-devices`
- [ ] Test that device name changes don't break encryption/decryption

**Test files:**
- `test/integration/config_set_device_name_test.go`
- `test/integration/config_rename_device_test.go`
- `test/integration/config_list_devices_test.go`

**Rationale:** Ensures config commands work correctly across various scenarios.

---

### Milestone 10.7: Update Rollout Plan

**Tasks:**
- [ ] Update Phase 4 (Device Identity Layer) to remove Milestone 4.4 (Rename Device Command)
- [ ] Update Phase 4 to remove `secrets rename-device` from list-devices command
- [ ] Update Rollout Plan to include Phase 10
- [ ] Update Success Criteria to include config command structure

**Updated Rollout Plan:**
```markdown
### Week 1: Configuration Foundation
- Complete Phases 1-2 (TOML, UUIDs)
- No breaking changes yet

### Week 2: Email Identity
- Complete Phase 3 (Email-based identity)
- Update all commands

### Week 3: Device Layer
- Complete Phase 4-5 (Device identity, revoke refinement)

### Week 4: Init & Migration
- Complete Phase 6-7 (Init, migration)

### Week 5: Testing & Docs
- Complete Phase 8-9 (Testing, documentation)

### Week 6: Config Commands
- Complete Phase 10 (Config command structure)
````

**Updated Success Criteria:**

```markdown
### Functional Requirements

- [ ] Users can set custom email
- [ ] Projects have unique UUIDs
- [ ] Files named with UUIDs (no collisions)
- [ ] Users can manage multiple devices
- [ ] Revoke works correctly (all devices, one device)
- [ ] Legacy projects migrate automatically
- [ ] Users can set device name preferences (config set-device-name)
- [ ] Project maintainers can rename devices (config rename-device)
- [ ] Device names are managed via `kanuka config` commands
```

**Rationale:** Ensures implementation plan is updated to reflect new command structure.

---

## Dependencies

### Go Packages

```go
require (
    github.com/BurntSushi/toml v1.3.2
    github.com/google/uuid v1.5.0
)
```

### File Structure Changes

```
cmd/
  config.go                    // New: Config command infrastructure
  config_set_device_name.go    // New: Set device name command
  config_rename_device.go      // New: Rename device command
  config_list_devices.go       // New: List devices command
internal/
  configs/
    config.go          // Update with new structs
    toml.go             // New: TOML parsing
  utils/
    system.go           // Update: Add GetHostname()
    uuid.go             // New: UUID generation
```

---

## Rollout Plan

### Week 1: Configuration Foundation

- Complete Phases 1-2 (TOML, UUIDs)
- No breaking changes yet

### Week 2: Email Identity

- Complete Phase 3 (Email-based identity)
- Update all commands

### Week 3: Device Layer

- Complete Phase 4-5 (Device identity, revoke refinement)
  - Note: `secrets list-devices` and `secrets rename-device` moved to Phase 10

### Week 4: Init & Migration

- Complete Phase 6-7 (Init, migration)

### Week 5: Testing & Docs

- Complete Phase 8-9 (Testing, documentation)

### Week 6: Config Commands

- Complete Phase 10 (Config command structure)
  - Implement `kanuka config` top-level command
  - Implement `config set-device-name` (user preferences)
  - Implement `config rename-device` (project-wide changes)
  - Move `config list-devices` from secrets
  - Add deprecation warnings for moved commands

---

## Success Criteria

### Functional Requirements

- [ ] Users can set custom email
- [ ] Projects have unique UUIDs
- [ ] Files named with UUIDs (no collisions)
- [ ] Users can manage multiple devices
- [ ] Revoke works correctly (all devices, one device)
- [ ] Legacy projects migrate automatically
- [ ] Users can set device name preferences (config set-device-name)
- [ ] Project maintainers can rename devices (config rename-device)
- [ ] Device names are managed via `kanuka config` commands

### Non-Functional Requirements

- [ ] All existing tests pass
- [ ] New tests cover edge cases
- [ ] Documentation is clear and complete
- [ ] Migration is transparent to users
- [ ] No breaking changes for existing workflows

---

## Risks and Mitigations

### Risk: Migration Failures

**Mitigation:**

- Create backups before migration
- Implement rollback capability
- Add verbose logging during migration
- Test migration thoroughly in staging

### Risk: User Confusion

**Mitigation:**

- Clear deprecation warnings
- Comprehensive migration guide
- FAQ for common questions
- Detailed command help

### Risk: Breaking Existing Workflows

**Mitigation:**

- Support both old and new formats temporarily
- Gradual migration (not forced)
- Clear upgrade instructions
- Monitor for issues post-release

---

## Notes

### Device Names Are Per-User

Device names only need to be unique **per user**, not globally. Alice's "macbook" is different from Bob's "macbook".

### Always Require `--user` with `--device`

The `--device` flag REQUIRES the `--user` flag. This prevents ambiguity when multiple users have the same device name.

### Smart Confirmation Logic

- Always confirm if revoking 2+ devices
- Skip confirmation if revoking 1 device
- Never confirm if using `--device` (explicit)
- Never confirm if using `--yes` (automation)

### Email is User Identifier

Email addresses are the user-facing identifier. UUIDs are internal only. Users never see UUIDs in normal operations.
