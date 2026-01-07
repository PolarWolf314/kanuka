# Kanuka Missing Features & Fixes

This document contains actionable tickets for missing features and bugs discovered during manual testing. Each ticket is self-contained and can be picked up independently unless dependencies are noted.

---

## Ticket 1: Create `kanuka config init` Command for User Setup

**Priority:** High  
**Type:** New Feature  
**Estimated Effort:** Medium  
**Dependencies:** None (but Tickets 2, 3, 6 depend on this)  
**Status:** ✅ COMPLETED

### Problem Statement

When a user runs `kanuka secrets init` for the first time, there is no mechanism to collect their user information (email, device name). The command assumes the user config already exists at `~/.config/kanuka/config.toml` with valid values, but first-time users have nothing configured.

Currently, users must manually create their config file or the system silently fails/uses incomplete data.

### Proposed Solution

Create a new `kanuka config init` command that handles interactive user setup:

1. Prompts for email address (required, with validation)
2. Prompts for display name (optional, for future git-log style tracking)
3. Prompts for default device name (with hostname-based default)
4. Generates user UUID if not already present
5. Saves to `~/.config/kanuka/config.toml`

The command should be idempotent - running it again should show current values and allow updates.

### User Flow

```bash
$ kanuka config init
Welcome to Kanuka! Let's set up your identity.

Email address: alice@example.com
Display name (optional) []: Alice Smith
Default device name [MacBook-Pro]: 

✓ User configuration saved to ~/.config/kanuka/config.toml

Your settings:
  Email: alice@example.com
  Name: Alice Smith
  Device: MacBook-Pro
  User ID: 6ba7b810-9dad-11d1-80b4-00c04fd430c8
```

### Acceptance Criteria

- [ ] Create `cmd/config_init.go` with `config init` subcommand
- [ ] Prompt for email with format validation (must contain `@`)
- [ ] Prompt for display name (optional, can be empty)
- [ ] Prompt for default device name with hostname-based default in brackets
- [ ] Generate user UUID if not present in existing config
- [ ] Preserve existing values if config already exists (show as defaults)
- [ ] Save config to `~/.config/kanuka/config.toml`
- [ ] Display summary of saved settings on completion
- [ ] Add `--email`, `--name`, `--device` flags for non-interactive/script usage
- [ ] Add unit tests for config init logic
- [ ] Add integration tests for interactive and non-interactive flows
- [ ] Update command help text with examples

### User Config Structure

```toml
[user]
email = "alice@example.com"
name = "Alice Smith"           # New field, optional
user_uuid = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
default_device_name = "MacBook-Pro"  # New field, optional

[projects]
# project_uuid -> device_name overrides
```

### Technical Notes

- The `name` field is for display purposes only (future git-log style tracking)
- The `default_device_name` is used when creating keys for new projects
- Email remains the primary user identifier
- Must update `internal/configs/config.go` to add new struct fields

### Rationale

Separating user identity setup from project initialization follows the principle of separation of concerns. User configuration is fundamentally different from project configuration - it's personal, machine-specific, and persists across all projects. Having a dedicated command makes the mental model clearer and enables other commands to call it when needed.

---

## Ticket 2: Integrate `config init` into `secrets init` Flow

**Priority:** High  
**Type:** Enhancement  
**Estimated Effort:** Small  
**Dependencies:** Ticket 1 must be completed first  
**Status:** ✅ COMPLETED

### Problem Statement

After Ticket 1 is implemented, `kanuka secrets init` needs to detect when user configuration is missing or incomplete and prompt the user to complete setup before proceeding with project initialization.

### Proposed Solution

Modify `kanuka secrets init` to:

1. Check if user config exists and has required fields (email, user_uuid)
2. If missing/incomplete, either:
   - Run `config init` flow inline, OR
   - Print error with instructions to run `kanuka config init` first
3. Proceed with project initialization only after user config is valid

### User Flow (Missing Config)

```bash
$ kanuka secrets init
⚠ User configuration not found.

Running initial setup...

Email address: alice@example.com
Display name (optional) []: Alice Smith
Default device name [MacBook-Pro]: 

✓ User configuration saved.

Initializing project...
Project name [my-project]: 
✓ Project initialized successfully!
```

### Acceptance Criteria

- [ ] Add check for valid user config at start of `secrets init`
- [ ] If user config missing/incomplete, run `config init` flow inline
- [ ] After user config is valid, proceed with project initialization
- [ ] Ensure `--yes` flag for scripts handles this gracefully (fail with clear error if config missing)
- [ ] Add integration test: `secrets init` with no user config triggers setup
- [ ] Add integration test: `secrets init` with valid config skips setup

### Technical Notes

- Consider extracting the `config init` logic into a reusable function that both commands can call
- The inline flow should be seamless - user shouldn't feel like they're running two commands

### Rationale

First-time user experience is critical. Users shouldn't have to know about `config init` before they can use `secrets init`. The system should guide them through setup naturally while still maintaining clear command separation for advanced users.

---

## Ticket 3: Add Project Name Prompt to `secrets init`

**Priority:** Medium  
**Type:** Enhancement  
**Estimated Effort:** Small  
**Dependencies:** None (can be done in parallel with Tickets 1-2)  
**Status:** ✅ COMPLETED

### Problem Statement

`kanuka secrets init` currently uses the directory name as the project name without asking the user. This is fine as a default, but users should have the opportunity to specify a different name.

### Proposed Solution

Add an interactive prompt for project name with the directory name as the default:

```bash
$ kanuka secrets init
Project name [my-awesome-project]: My Awesome Project
```

Add a `--name` flag for non-interactive/script usage:

```bash
$ kanuka secrets init --name "My Awesome Project"
```

### Acceptance Criteria

- [ ] Add prompt for project name with directory name as default in brackets
- [ ] User can press Enter to accept default
- [ ] User can type custom name to override
- [ ] Add `--name` flag for non-interactive usage
- [ ] Validate project name (non-empty, reasonable characters)
- [ ] Save project name to `.kanuka/config.toml` `[project]` section
- [ ] Add integration test: default name acceptance
- [ ] Add integration test: custom name entry
- [ ] Add integration test: `--name` flag usage
- [ ] Update command help text with `--name` flag documentation

### Technical Notes

- Project name is stored in `.kanuka/config.toml` under `[project].name`
- The project UUID remains the canonical identifier; name is for display
- Consider what characters are valid in project names (spaces? special chars?)

### Rationale

Providing sensible defaults with the ability to override is good UX. The directory name is usually correct, but projects may have display names that differ from their directory names.

---

## Ticket 4: Fix User Config Not Updated on `secrets init`

**Priority:** High  
**Type:** Bug Fix  
**Estimated Effort:** Small  
**Dependencies:** None  
**Status:** ✅ COMPLETED

### Problem Statement

When `kanuka secrets init` creates a key pair for a new project, it does not add an entry to the user's config file (`~/.config/kanuka/config.toml`). This means:

1. The user has no record of which projects they have keys for
2. The `[projects]` section (project UUID → device name mapping) is never populated
3. Future features like `kanuka list` cannot enumerate user's projects

### Current Behavior

```bash
$ kanuka secrets init
✓ Project initialized

$ cat ~/.config/kanuka/config.toml
[user]
email = "alice@example.com"
user_uuid = "abc123"

[projects]
# Empty - no record of the project we just initialized!
```

### Expected Behavior

```bash
$ kanuka secrets init
✓ Project initialized

$ cat ~/.config/kanuka/config.toml
[user]
email = "alice@example.com"
user_uuid = "abc123"

[projects]
"550e8400-e29b-41d4-a716-446655440000" = "MacBook-Pro"
```

### Acceptance Criteria

- [ ] After creating keys in `secrets init`, add entry to user config `[projects]` section
- [ ] Entry should map project UUID to device name used
- [ ] If entry already exists for this project UUID, update it (user may be re-initializing)
- [ ] Add integration test: verify user config updated after `secrets init`
- [ ] Add integration test: verify existing entry is updated on re-init

### Files to Modify

- `cmd/secrets_init.go` - add call to update user config after key creation
- Possibly `internal/configs/config.go` - add helper function if needed

### Rationale

The user config's `[projects]` section exists specifically to track which projects the user has keys for and what device name they use. Not populating it defeats the purpose of this design.

---

## Ticket 5: Fix `config rename-device` to Update User Config When Appropriate

**Priority:** Low  
**Type:** Enhancement  
**Estimated Effort:** Small  
**Dependencies:** None  
**Status:** ✅ COMPLETED

### Problem Statement

`kanuka config rename-device` updates the device name in the project config (`.kanuka/config.toml`) but does not update the user config (`~/.config/kanuka/config.toml`).

### Design Consideration

There are two perspectives here:

1. **Project config** stores the canonical device name visible to the team
2. **User config** stores the user's personal preference for their device name

When a project admin renames *someone else's* device, it should NOT update that user's local preferences (they may not even be on the same machine).

When a user renames *their own* device, it SHOULD update their user config to keep them in sync.

### Proposed Solution

In `config rename-device`, check if the device being renamed belongs to the current user (compare user UUIDs). If yes, also update the user config's `[projects]` section.

### Acceptance Criteria

- [ ] Detect if device being renamed belongs to current user (same user UUID)
- [ ] If current user's device, update `[projects]` entry in user config
- [ ] If different user's device, only update project config (current behavior)
- [ ] Add integration test: rename own device updates both configs
- [ ] Add integration test: rename other user's device only updates project config

### Technical Notes

- Compare the device's user UUID (from project config) with current user's UUID (from user config)
- This requires loading both configs during the rename operation

### Rationale

Keeping user config in sync when renaming your own device prevents confusion. But we shouldn't modify other users' preferences - that would be overstepping.

---

## Ticket 6: Add `kanuka config show` Command

**Priority:** Medium  
**Type:** New Feature  
**Estimated Effort:** Small  
**Dependencies:** Ticket 1 (for complete user config structure)  
**Status:** ✅ COMPLETED

### Problem Statement

Users have no easy way to view their current configuration. They must manually read TOML files to see what values are set. This is poor UX and makes debugging difficult.

### Proposed Solution

Create a `kanuka config show` command that displays:

1. User config values (default)
2. Project config values (with `--project` flag, when in project directory)

### User Flow

```bash
$ kanuka config show
User Configuration (~/.config/kanuka/config.toml):
  Email:        alice@example.com
  Name:         Alice Smith
  User ID:      6ba7b810-9dad-11d1-80b4-00c04fd430c8
  Default Device: MacBook-Pro

Projects:
  550e8400... → workstation (my-awesome-project)
  7ba7b810... → laptop (another-project)
```

```bash
$ kanuka config show --project
Project Configuration (.kanuka/config.toml):
  Project ID:   550e8400-e29b-41d4-a716-446655440000
  Project Name: my-awesome-project

Users:
  alice@example.com (6ba7b810...)
    - workstation (created: Jan 6, 2025)
    - laptop (created: Jan 7, 2025)
  bob@company.com (8ba7b810...)
    - macbook (created: Jan 5, 2025)
```

### Acceptance Criteria

- [ ] Create `cmd/config_show.go` with `config show` subcommand
- [ ] Display user config by default (email, name, user ID, default device)
- [ ] List all projects in user config with device names
- [ ] Add `--project` flag to show project config instead
- [ ] `--project` requires being in a project directory (show error otherwise)
- [ ] Display project ID, name, all users, and their devices
- [ ] Add `--json` flag for machine-readable output
- [ ] Add integration tests for user config display
- [ ] Add integration tests for project config display
- [ ] Add integration tests for `--json` output format
- [ ] Handle missing config gracefully (show helpful message)

### Technical Notes

- For the projects list in user config, we only have UUID → device name mapping
- To show project names, we'd need to look up each project's config (may not be accessible if not in that directory)
- Consider showing "unknown" for project names we can't resolve, or just showing UUIDs

### Rationale

Visibility into configuration is essential for debugging and understanding system state. Users should never need to manually parse TOML files.

---

## Ticket 7: Restructure Key Storage Directory Layout

**Priority:** Low  
**Type:** Improvement  
**Estimated Effort:** Medium  
**Dependencies:** None  
**Status:** ✅ COMPLETED

### Problem Statement

Current key storage structure is flat:

```
~/.local/share/kanuka/keys/
  550e8400-e29b-41d4-a716-446655440000      # private key
  550e8400-e29b-41d4-a716-446655440000.pub  # public key
```

This has limitations:
- Cannot add metadata without changing filenames
- Harder to extend with additional files per project
- Cluttered listing when user has many projects

### Proposed Solution

Restructure to use directories per project:

```
~/.local/share/kanuka/keys/
  550e8400-e29b-41d4-a716-446655440000/
    privkey
    pubkey.pub
    metadata.toml
```

The `metadata.toml` file would contain:

```toml
project_name = "my-awesome-project"
project_path = "/Users/alice/projects/my-awesome-project"  # last known path
created_at = "2025-01-06T10:00:00Z"
```

### Acceptance Criteria

- [ ] Create new directory structure for key storage
- [ ] Update `CreateAndSaveRSAKeyPair()` to use new structure
- [ ] Update all key loading functions to use new paths
- [ ] Create `metadata.toml` when generating keys
- [ ] Update `metadata.toml` when project is accessed (update `project_path`)
- [ ] Update all tests that check key paths
- [ ] Add integration test: verify directory structure after key creation
- [ ] Add integration test: verify metadata.toml contents

### Files to Modify

- `internal/secrets/keys.go` - key creation and loading
- `internal/configs/settings.go` - path construction
- All integration tests that verify key locations

### Benefits

1. **Extensibility**: Easy to add more metadata without path changes
2. **Discoverability**: `ls ~/.local/share/kanuka/keys/` shows project UUIDs clearly
3. **Future features**: Enables `kanuka list` to show all projects with names
4. **Cleaner organization**: Related files grouped together

### Rationale

This is a foundational improvement that makes the system more maintainable and enables future features. The metadata file is particularly valuable for features like listing all projects a user has keys for.

---

## Ticket 8: Improve `secrets register` Output with Full Paths

**Priority:** Low  
**Type:** UX Improvement  
**Estimated Effort:** Small  
**Dependencies:** None

### Problem Statement

`kanuka secrets register` output only shows filenames, not full paths:

```
✓ alice@example.com has been granted access
```

For consistency with `secrets encrypt` (which shows full paths) and for clarity, the output should include the paths of created files.

### Proposed Solution

Update output to show full paths of created files:

```
✓ alice@example.com has been granted access

Files created:
  Public key:     /path/to/project/.kanuka/public_keys/6ba7b810.pub
  Encrypted key:  /path/to/project/.kanuka/secrets/6ba7b810.kanuka
```

### Acceptance Criteria

- [ ] Update `secrets register` success output to include file paths
- [ ] Show public key path
- [ ] Show encrypted symmetric key path
- [ ] Maintain existing success message
- [ ] Update integration tests to verify new output format
- [ ] Ensure verbose mode shows even more detail if appropriate

### Files to Modify

- `cmd/secrets_register.go` - update success output

### Rationale

Consistency across commands improves UX. When encryption shows paths, registration should too. This also helps users understand where files are being created, which is valuable for debugging and learning how Kanuka works.

---

## Ticket 9: Complete Documentation for Config Commands

**Priority:** Medium  
**Type:** Documentation  
**Estimated Effort:** Medium  
**Dependencies:** Tickets 1, 6 (to document final command structure)

### Problem Statement

The `kanuka config` commands lack comprehensive documentation:

1. No complete command reference for all subcommands
2. No conceptual documentation explaining user config vs project config
3. No examples of common workflows

### Proposed Solution

Create/update documentation:

1. **Command Reference**: Document all `kanuka config` subcommands with flags, examples
2. **Concepts Guide**: Explain the two config files and how they interact
3. **Workflow Examples**: Common scenarios with step-by-step instructions

### Acceptance Criteria

#### Command Reference

- [ ] Document `kanuka config init` with all flags and examples
- [ ] Document `kanuka config show` with all flags and examples
- [ ] Document `kanuka config set-device-name` with all flags and examples
- [ ] Document `kanuka config rename-device` with all flags and examples
- [ ] Document `kanuka config list-devices` with all flags and examples

#### Concepts Guide

- [ ] Create `docs/src/content/docs/concepts/configuration.md` (or similar)
- [ ] Explain user config (`~/.config/kanuka/config.toml`)
  - [ ] Purpose and scope
  - [ ] All fields and their meanings
  - [ ] When it's created/updated
- [ ] Explain project config (`.kanuka/config.toml`)
  - [ ] Purpose and scope
  - [ ] All sections (`[project]`, `[users]`, `[devices]`)
  - [ ] When it's created/updated
- [ ] Explain how the two configs interact
- [ ] Explain the identity hierarchy (project → user → device)

#### Workflow Examples

- [ ] First-time setup workflow
- [ ] Adding a new device workflow
- [ ] Renaming your device workflow
- [ ] Viewing your configuration workflow

### Files to Create/Modify

- `docs/src/content/docs/concepts/configuration.md` (new)
- `docs/src/content/docs/reference/commands/config.md` (new or update existing)
- `docs/src/content/docs/guides/config.md` (update existing)

### Rationale

Good documentation is essential for user adoption. The config system is a core part of Kanuka's identity model, and users need to understand how it works to use Kanuka effectively.

---

## Ticket 10: Document Non-Deterministic Encryption Behavior

**Priority:** Low  
**Type:** Documentation  
**Estimated Effort:** Small  
**Dependencies:** None

### Problem Statement

When running `kanuka secrets encrypt` multiple times on the same `.env` file with no changes, the output differs each time. This creates git diffs even when secrets haven't changed.

Users may perceive this as a bug, but it's actually a **security feature**: AES-GCM encryption uses a random nonce/IV for each encryption operation. Encrypting the same plaintext twice produces different ciphertext, which prevents attackers from detecting when the same secret is reused.

### Proposed Solution

Document this behavior clearly:

1. Add explanation to the encryption guide
2. Add FAQ entry
3. Consider adding a note in the CLI output

### Acceptance Criteria

#### Documentation

- [ ] Add section to encryption guide explaining non-deterministic output
- [ ] Explain why this is a security feature (nonce/IV randomization)
- [ ] Explain that git diffs are expected even with unchanged secrets
- [ ] Recommend committing encrypted files after each `encrypt` run

#### FAQ

- [ ] Add FAQ entry: "Why do encrypted files change even when my secrets haven't?"
- [ ] Explain the security rationale
- [ ] Provide guidance on git workflow

#### Optional: CLI Output

- [ ] Consider adding a note after encryption: "Note: Encrypted output is non-deterministic for security. Git diffs are expected."
- [ ] This could be shown only on first use or with `--verbose`

### Documentation Content

```markdown
## Non-Deterministic Encryption

You may notice that running `kanuka secrets encrypt` produces different 
output each time, even when your `.env` file hasn't changed. This is 
expected behavior and a security feature.

### Why This Happens

Kanuka uses AES-GCM encryption, which requires a unique nonce (number 
used once) for each encryption operation. This nonce is randomly 
generated, so encrypting the same plaintext twice produces different 
ciphertext.

### Why This Matters for Security

If encryption were deterministic, an attacker could:
- Detect when the same secret is reused across files
- Build a dictionary of encrypted values to guess plaintext
- Identify patterns in your secrets

Random nonces prevent these attacks.

### Git Workflow

Since encrypted files change on each run, you'll see git diffs even when 
secrets haven't changed. This is normal. We recommend:

1. Run `encrypt` when you actually change secrets
2. Commit the encrypted files immediately after
3. Don't re-run `encrypt` unnecessarily
```

### Rationale

Users encountering unexpected behavior should find clear explanations. Documenting this as a security feature (rather than letting users think it's a bug) builds trust and demonstrates that Kanuka follows cryptographic best practices.

---

## Summary

| Ticket | Title | Priority | Type | Effort |
|--------|-------|----------|------|--------|
| 1 | Create `kanuka config init` Command | High | Feature | Medium |
| 2 | Integrate `config init` into `secrets init` | High | Enhancement | Small |
| 3 | Add Project Name Prompt to `secrets init` | Medium | Enhancement | Small |
| 4 | Fix User Config Not Updated on `secrets init` | High | Bug Fix | Small |
| 5 | Fix `config rename-device` User Config Update | Low | Enhancement | Small |
| 6 | Add `kanuka config show` Command | Medium | Feature | Small |
| 7 | Restructure Key Storage Directory Layout | Low | Improvement | Medium |
| 8 | Improve `secrets register` Output with Paths | Low | UX | Small |
| 9 | Complete Documentation for Config Commands | Medium | Docs | Medium |
| 10 | Document Non-Deterministic Encryption | Low | Docs | Small |

### Recommended Order

1. **Ticket 1** - Foundation for user setup
2. **Ticket 4** - Bug fix, can be done in parallel
3. **Ticket 2** - Depends on Ticket 1
4. **Ticket 3** - Can be done in parallel with 1-2
5. **Ticket 6** - Depends on Ticket 1 for complete structure
6. **Ticket 5** - Low priority, do after core features
7. **Ticket 7** - Low priority, architectural improvement
8. **Ticket 8** - Low priority, UX polish
9. **Ticket 9** - Do after features are finalized
10. **Ticket 10** - Can be done anytime
