# New Features Plan

## Overview

This document outlines new features and improvements for Kanuka's secrets management system. Features are organized by priority based on security impact and user value.

---

## Feature 1: Register Warning + Force Flag

### Priority: 1 (Critical - Trivial Fix)

### Problem

Currently, `kanuka secrets register` silently overwrites existing user keys without warning. This can accidentally lock someone out if the old key was correct.

### Solution

- Detect when a user already has access (public key + `.kanuka` file exist)
- Print a warning and require confirmation
- Add `--force` flag to skip confirmation (for scripting)

### Behavior

```bash
# User already has access
$ kanuka secrets register --user alice@example.com
⚠ Warning: alice@example.com already has access to this project.
  Continuing will replace their existing key.
  If they generated a new keypair, this is expected.
  If not, they may lose access.

Do you want to continue? [y/N]: 

# With --force flag
$ kanuka secrets register --user alice@example.com --force
✓ Updated access for alice@example.com
```

### Implementation

1. In `cmd/secrets_register.go`, before registering:
   - Check if `<uuid>.pub` exists in `.kanuka/public_keys/`
   - Check if `<uuid>.kanuka` exists in `.kanuka/secrets/`
2. If both exist, prompt for confirmation (unless `--force`)
3. Add `--force` flag to command

### Files to Modify

- `cmd/secrets_register.go`

---

## Feature 2: Revoke Security Fix + Full Re-encryption

### Priority: 2 (Critical - Security Gap)

### Problem

Currently, `kanuka secrets revoke` removes the user's key files and rotates the symmetric key, but **does not re-encrypt the secret files**. This is a security gap:

- The revoked user may have copied the encrypted `.kanuka` files
- They still have access to the old symmetric key (it was encrypted for them)
- They can decrypt any files encrypted with the old symmetric key

### Solution

Revoke must perform full re-encryption:

1. Decrypt all `.kanuka` files to get plaintext
2. Remove revoked user's public key and `.kanuka` file
3. Generate NEW symmetric key
4. Re-encrypt symmetric key for all remaining users
5. Re-encrypt all files with new symmetric key

### Behavior

```bash
$ kanuka secrets revoke --user alice@example.com
⠋ Revoking access for alice@example.com...
  Decrypting secrets...
  Removing user keys...
  Generating new encryption key...
  Re-encrypting secrets for 3 remaining users...
✓ Access revoked for alice@example.com
  All secrets have been re-encrypted with a new key.
```

### Implementation

1. Decrypt all `.kanuka` files to memory (not disk)
2. Delete user's public key and `.kanuka` file
3. Generate new symmetric key
4. For each remaining user, encrypt new symmetric key with their public key
5. Re-encrypt all plaintext with new symmetric key
6. Write new `.kanuka` files

### Files to Modify

- `cmd/secrets_revoke.go`
- `internal/secrets/crypto.go` (may need new helpers)

### Security Considerations

- Plaintext is held in memory only, never written to disk during revoke
- Old encrypted files are overwritten atomically
- If revoke fails mid-operation, should rollback cleanly

---

## Feature 3: `kanuka secrets sync`

### Priority: 3 (High - Enables Revoke Fix)

### Purpose

Re-encrypt all secrets with a new symmetric key. Useful for:

- Manual key rotation
- After adding new users (ensure they can decrypt)
- Called internally by `revoke`

### Behavior

```bash
$ kanuka secrets sync
⠋ Syncing secrets...
  Decrypting 5 secret files...
  Generating new encryption key...
  Re-encrypting for 4 users...
  Re-encrypting 5 secret files...
✓ Secrets synced successfully
  New encryption key generated and distributed to all users.
```

### Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Show what would happen without making changes |

### Implementation

1. Find all `.kanuka` files
2. Decrypt all files using current symmetric key
3. Generate new symmetric key
4. Encrypt new symmetric key for each user (using their public keys)
5. Re-encrypt all files with new symmetric key
6. Write new `.kanuka` files and user key files

### Refactoring Opportunity

Extract the core sync logic into `internal/secrets/sync.go`:

```go
func SyncSecrets(privateKey *rsa.PrivateKey) error {
    // 1. Decrypt all files
    // 2. Generate new symmetric key
    // 3. Re-encrypt for all users
    // 4. Re-encrypt all files
}
```

Then `revoke` becomes:
```go
// 1. Delete user's keys
// 2. Call SyncSecrets()
```

### Files to Create/Modify

- `cmd/secrets_sync.go` (new)
- `internal/secrets/sync.go` (new)
- `cmd/secrets_revoke.go` (refactor to use sync)

---

## Feature 4: `kanuka secrets access`

### Priority: 4 (High Value, Low Effort)

### Purpose

List all users who have access to the project's secrets, and show pending/problematic states.

### Behavior

```bash
$ kanuka secrets access
Project: kanuka
Users with access:

  USER                          STATUS      ADDED
  alice@example.com             ✓ active    2024-01-15
  bob@example.com               ✓ active    2024-01-20
  charlie@example.com           ⚠ pending   2024-02-01

Legend:
  ✓ active  - User has public key and encrypted symmetric key
  ⚠ pending - User has public key but no encrypted symmetric key (needs sync)
  ✗ orphan  - User has encrypted symmetric key but no public key (should not happen)

Total: 3 users (2 active, 1 pending)
```

### Status Definitions

| Status | Public Key Exists | `.kanuka` File Exists | Meaning |
|--------|-------------------|----------------------|---------|
| active | Yes | Yes | User can decrypt secrets |
| pending | Yes | No | User registered but sync needed |
| orphan | No | Yes | Inconsistent state, needs cleanup |

### Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON for scripting |

### Implementation

1. List all `.pub` files in `.kanuka/public_keys/`
2. List all `.kanuka` files in `.kanuka/secrets/`
3. Cross-reference to determine status
4. Extract metadata (email, date) from key files or config

### Files to Create

- `cmd/secrets_access.go`

---

## Feature 5: `kanuka secrets status`

### Priority: 5 (High Value, Low Effort)

### Purpose

Show the encryption status of all secret files in the project.

### Behavior

```bash
$ kanuka secrets status
Project: kanuka
Secret files status:

  FILE                    STATUS
  .env                    ✓ encrypted (up to date)
  .env.local              ✓ encrypted (up to date)
  config/.env.production  ⚠ encrypted (stale - plaintext is newer)
  scripts/.env.test       ✗ not encrypted

Summary:
  2 files up to date
  1 file stale (run 'kanuka secrets encrypt' to update)
  1 file not encrypted (run 'kanuka secrets encrypt' to secure)
```

### Status Definitions

| Status | Meaning |
|--------|---------|
| ✓ encrypted (up to date) | `.kanuka` file exists and is newer than plaintext |
| ⚠ encrypted (stale) | `.kanuka` file exists but plaintext is newer (modified after encryption) |
| ✗ not encrypted | Plaintext exists but no `.kanuka` file |
| ◌ encrypted only | `.kanuka` file exists but no plaintext (normal after decrypt cleanup) |

### Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON for scripting |

### Implementation

1. Find all `.env*` files (excluding `.kanuka` directory)
2. Find all `.kanuka` files in secrets directory
3. Compare timestamps to determine staleness
4. Report status for each file

### Files to Create

- `cmd/secrets_status.go`

---

## Feature 6: `kanuka secrets doctor`

### Priority: 6 (Medium Value)

### Purpose

Check project health and detect common issues.

### Behavior

```bash
$ kanuka secrets doctor
Running health checks...

✓ Project configuration valid
✓ User configuration valid
✓ Private key permissions correct (0600)
✓ All public keys have corresponding .kanuka files
✓ All .kanuka files have corresponding public keys
✓ .env files are in .gitignore
⚠ Found 1 unencrypted .env file (run 'kanuka secrets status' for details)
✗ Private key missing for this project

Summary: 6 passed, 1 warning, 1 error

Run 'kanuka secrets create' to generate missing private key.
```

### Checks Performed

| Check | Severity | Description |
|-------|----------|-------------|
| Project config valid | Error | `.kanuka/config.toml` exists and parses |
| User config valid | Error | User config exists and parses |
| Private key exists | Error | User has private key for this project |
| Private key permissions | Warning | Should be 0600 |
| Public key consistency | Error | Every public key has a `.kanuka` file |
| `.kanuka` file consistency | Error | Every `.kanuka` file has a public key |
| Gitignore check | Warning | `.env*` patterns in `.gitignore` |
| Unencrypted files | Warning | `.env` files without encryption |

### Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON for scripting |
| `--fix` | Attempt to fix issues automatically (where possible) |

### Implementation

1. Run each check in sequence
2. Collect results with severity levels
3. Print summary with actionable suggestions
4. Exit with non-zero code if errors found (useful for CI)

### Files to Create

- `cmd/secrets_doctor.go`
- `internal/secrets/doctor.go` (check implementations)

---

## Feature 7: `kanuka secrets rotate`

### Priority: 7 (Medium Value)

### Purpose

Rotate the current user's keypair. Useful for security hygiene or if a key is compromised.

### Behavior

```bash
$ kanuka secrets rotate
⠋ Rotating your keypair...
  Generating new keypair...
  Decrypting your symmetric key with old private key...
  Re-encrypting symmetric key with new public key...
  Updating public key in project...
  Saving new private key...
✓ Keypair rotated successfully

Your new public key has been added to the project.
Other users do not need to take any action.
```

### What It Does

1. Generate new RSA keypair
2. Decrypt the symmetric key using old private key
3. Re-encrypt symmetric key with new public key
4. Replace old public key in `.kanuka/public_keys/`
5. Replace old private key in user's key directory
6. Old `.kanuka` file is updated with new encryption

### What It Does NOT Do

- Does not affect other users
- Does not rotate the project symmetric key (use `sync` for that)
- Does not re-encrypt secret files

### Flags

| Flag | Description |
|------|-------------|
| `--force` | Skip confirmation prompt |

### Files to Create

- `cmd/secrets_rotate.go`

---

## Feature 8: `kanuka secrets export` / `kanuka secrets import`

### Priority: 8 (Lower Priority)

### Purpose

Backup and restore encrypted secrets for disaster recovery or migration.

### `export` Behavior

```bash
$ kanuka secrets export
✓ Exported secrets to kanuka-secrets-backup-2024-02-15.tar.gz

Contents:
  - .kanuka/config.toml
  - .kanuka/public_keys/ (4 keys)
  - .kanuka/secrets/ (4 user keys, 5 encrypted files)

Note: This backup contains encrypted data only.
      Private keys are NOT included.
```

### `import` Behavior

```bash
$ kanuka secrets import kanuka-secrets-backup-2024-02-15.tar.gz
⠋ Importing secrets...

Found existing .kanuka directory. How do you want to proceed?
  [m] Merge - Add new files, keep existing
  [r] Replace - Delete existing, use backup
  [c] Cancel

Choice: m

✓ Imported 2 new public keys
✓ Imported 2 new encrypted files
⚠ Skipped 3 files (already exist)
```

### Export Contents

The export archive contains:
- `.kanuka/config.toml` - Project configuration
- `.kanuka/public_keys/*.pub` - All user public keys
- `.kanuka/secrets/*.kanuka` - Encrypted symmetric keys for each user
- All `*.kanuka` files in project - Encrypted secret files

### What Is NOT Exported

- Private keys (never leave the user's machine)
- Plaintext `.env` files (security risk)

### Flags

**export:**
| Flag | Description |
|------|-------------|
| `--output`, `-o` | Output file path (default: auto-generated name) |

**import:**
| Flag | Description |
|------|-------------|
| `--merge` | Merge with existing (default) |
| `--replace` | Replace existing completely |
| `--dry-run` | Show what would be imported |

### Files to Create

- `cmd/secrets_export.go`
- `cmd/secrets_import.go`

---

## Implementation Order

Based on dependencies and priority:

```
Phase 1: Security Fixes
├── 1. Register warning + --force flag
├── 2. Create internal/secrets/sync.go (core sync logic)
├── 3. Implement 'kanuka secrets sync' command
└── 4. Refactor 'kanuka secrets revoke' to use sync

Phase 2: Visibility Commands
├── 5. Implement 'kanuka secrets access'
└── 6. Implement 'kanuka secrets status'

Phase 3: Health & Maintenance
├── 7. Implement 'kanuka secrets doctor'
└── 8. Implement 'kanuka secrets rotate'

Phase 4: Backup & Recovery
├── 9. Implement 'kanuka secrets export'
└── 10. Implement 'kanuka secrets import'
```

---

## Command Summary

After implementation, the full command tree will be:

```
kanuka secrets
├── init              Initialize secrets management for a project
├── create            Create user keypair for a project
├── encrypt           Encrypt .env files
├── decrypt           Decrypt .kanuka files
├── register          Add user access (with overwrite warning)
├── revoke            Remove user access (with full re-encryption)
├── sync              Re-encrypt all secrets with new symmetric key
├── access            List users and their access status
├── status            Show encryption status of secret files
├── doctor            Check project health
├── rotate            Rotate current user's keypair
├── export            Export encrypted secrets to archive
└── import            Import encrypted secrets from archive
```

---

## Testing Strategy

Each feature should include:

1. **Unit tests** for new functions in `internal/`
2. **Integration tests** in `test/integration/<command>/`
3. **Edge case tests**:
   - Empty project (no secrets)
   - Single user
   - Multiple users
   - Missing files
   - Permission errors

---

## Documentation Updates

After implementation, update:

1. `docs/src/content/docs/guides/` - Add guides for new commands
2. `docs/src/content/docs/reference/` - Command reference
3. `README.md` - Feature list
4. `--help` text for each new command
