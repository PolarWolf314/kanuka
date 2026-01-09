# New Features Implementation Guide

This document contains detailed implementation tickets for Kanuka's new features. Each ticket is self-contained with full context, allowing any developer to pick it up and implement it.

---

## Table of Contents

1. [KAN-001: Register Warning + Force Flag](#kan-001-register-warning--force-flag)
2. [KAN-002: Core Sync Logic](#kan-002-core-sync-logic)
3. [KAN-003: Sync Command](#kan-003-sync-command)
4. [KAN-004: Revoke Security Fix](#kan-004-revoke-security-fix)
5. [KAN-005: Access Command](#kan-005-access-command)
6. [KAN-006: Clean Command](#kan-006-clean-command)
7. [KAN-007: Status Command](#kan-007-status-command)
8. [KAN-008: Doctor Command](#kan-008-doctor-command)
9. [KAN-009: Rotate Command](#kan-009-rotate-command)
10. [KAN-010: Export Command](#kan-010-export-command)
11. [KAN-011: Import Command](#kan-011-import-command)

---

## Implementation Order & Dependencies

```
KAN-001 (Register Warning)     ─── No dependencies, implement first
     │
KAN-002 (Core Sync Logic)      ─── No dependencies, implement early
     │
     ├── KAN-003 (Sync Command) ─── Depends on KAN-002
     │
     └── KAN-004 (Revoke Fix)   ─── Depends on KAN-002, KAN-003
     
KAN-005 (Access Command)       ─── No dependencies
KAN-006 (Clean Command)        ─── Depends on KAN-005 (follows from access output)
KAN-007 (Status Command)       ─── No dependencies

KAN-008 (Doctor Command)       ─── Soft dependency on KAN-005, KAN-006, KAN-007 (can reuse logic)
KAN-009 (Rotate Command)       ─── No dependencies

KAN-010 (Export Command)       ─── No dependencies
KAN-011 (Import Command)       ─── Depends on KAN-010 (shared archive format)
```

---

## KAN-001: Register Warning + Force Flag

### Summary

Add a warning when `kanuka secrets register` would overwrite an existing user's keys, with a `--force` flag to skip confirmation.

### Priority

**Critical** - Trivial fix, immediate UX improvement

### Context

Currently, `kanuka secrets register` silently overwrites existing user keys without any warning. This is problematic because:

1. A user might accidentally run register twice, replacing a valid key
2. If the old key was correct and the new one is wrong, the user loses access
3. There's no indication that anything was replaced

The command should detect when a user already has access and warn before proceeding.

### Current Behavior

```bash
$ kanuka secrets register --user alice@example.com
✓ Registered alice@example.com  # Silently overwrites if exists
```

### New Behavior

```bash
# User already has access - interactive mode
$ kanuka secrets register --user alice@example.com
⚠ Warning: alice@example.com already has access to this project.
  Continuing will replace their existing key.
  If they generated a new keypair, this is expected.
  If not, they may lose access.

Do you want to continue? [y/N]: y
✓ Updated access for alice@example.com

# User already has access - with --force flag
$ kanuka secrets register --user alice@example.com --force
✓ Updated access for alice@example.com

# User does not have access - no change in behavior
$ kanuka secrets register --user newuser@example.com
✓ Registered newuser@example.com
```

### Acceptance Criteria

- [x] When registering a user who already has access (public key AND .kanuka file exist), display a warning
- [x] Prompt for confirmation before proceeding (unless `--force` is provided)
- [x] Add `--force` flag to skip the confirmation prompt
- [x] If user declines confirmation, exit without making changes
- [x] If user confirms or uses `--force`, proceed with registration (overwrite)
- [x] Output message should say "Updated access" instead of "Registered" when overwriting
- [x] Existing behavior unchanged for new users

### Technical Details

#### Files to Modify

| File | Changes |
|------|---------|
| `cmd/secrets_register.go` | Add `--force` flag, add existence check, add confirmation prompt |

#### Implementation Steps

1. **Add the `--force` flag:**
   ```go
   var registerForce bool
   
   func init() {
       registerCmd.Flags().BoolVar(&registerForce, "force", false, "skip confirmation when updating existing user")
   }
   ```

2. **Check if user already has access:**
   ```go
   // After resolving the user's UUID
   publicKeyPath := filepath.Join(configs.ProjectKanukaSettings.ProjectPublicKeyPath, userUUID+".pub")
   kanukaFilePath := filepath.Join(configs.ProjectKanukaSettings.ProjectSecretsPath, userUUID+".kanuka")
   
   publicKeyExists := fileExists(publicKeyPath)
   kanukaFileExists := fileExists(kanukaFilePath)
   
   userAlreadyHasAccess := publicKeyExists && kanukaFileExists
   ```

3. **Prompt for confirmation if needed:**
   ```go
   if userAlreadyHasAccess && !registerForce {
       fmt.Printf("⚠ Warning: %s already has access to this project.\n", userEmail)
       fmt.Println("  Continuing will replace their existing key.")
       fmt.Println("  If they generated a new keypair, this is expected.")
       fmt.Println("  If not, they may lose access.")
       fmt.Println()
       
       if !confirmAction("Do you want to continue?") {
           fmt.Println("Aborted.")
           return nil
       }
   }
   ```

4. **Update output message:**
   ```go
   if userAlreadyHasAccess {
       fmt.Printf("✓ Updated access for %s\n", userEmail)
   } else {
       fmt.Printf("✓ Registered %s\n", userEmail)
   }
   ```

#### Helper Function

You may need to add a confirmation helper (or reuse existing):

```go
func confirmAction(prompt string) bool {
    reader := bufio.NewReader(os.Stdin)
    fmt.Printf("%s [y/N]: ", prompt)
    response, _ := reader.ReadString('\n')
    response = strings.TrimSpace(strings.ToLower(response))
    return response == "y" || response == "yes"
}
```

### Testing Requirements

#### Integration Tests

Create `test/integration/register/register_overwrite_test.go`:

| Test Case | Description |
|-----------|-------------|
| `TestRegisterOverwrite_WarnsWhenUserExists` | Register existing user without --force, verify warning is shown |
| `TestRegisterOverwrite_ForceSkipsWarning` | Register existing user with --force, verify no prompt |
| `TestRegisterOverwrite_NewUserNoWarning` | Register new user, verify no warning |
| `TestRegisterOverwrite_AbortOnDecline` | Simulate declining confirmation, verify no changes made |

#### Test Considerations

- The interactive prompt makes testing tricky; consider:
  - Adding a `--yes` flag for non-interactive mode (like `init` has)
  - Or mocking stdin in tests (see `CaptureOutputWithStdin` helper)

### Definition of Done

- [x] `--force` flag added to register command
- [x] Warning displayed when user already has access
- [x] Confirmation prompt works correctly
- [x] "Updated access" message shown for overwrites
- [x] Integration tests pass
- [x] `golangci-lint run` passes
- [x] Help text updated for `--force` flag

---

## KAN-002: Core Sync Logic

### Summary

Create the core sync functionality in `internal/secrets/sync.go` that re-encrypts all secrets with a new symmetric key. This will be used by both the `sync` command and the `revoke` command.

### Priority

**Critical** - Required for KAN-003 and KAN-004

### Context

The sync operation is fundamental to several features:

1. **Revoke** needs to re-encrypt everything after removing a user (security requirement)
2. **Sync command** allows manual key rotation
3. **Future features** may also need this capability

By extracting this into a reusable module, we avoid code duplication and ensure consistent behavior.

### What Sync Does

1. Finds all encrypted `.kanuka` files in the project (the actual secrets, not user key files)
2. Decrypts them using the current symmetric key
3. Generates a new symmetric key
4. Re-encrypts the symmetric key for each user who has access
5. Re-encrypts all secret files with the new symmetric key
6. Writes the new encrypted files

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    internal/secrets/sync.go                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  SyncSecrets(privateKey *rsa.PrivateKey, opts SyncOptions)      │
│      │                                                           │
│      ├── 1. Find all .kanuka secret files                       │
│      ├── 2. Get current symmetric key (decrypt user's .kanuka)  │
│      ├── 3. Decrypt all secret files to memory                  │
│      ├── 4. Generate new symmetric key                          │
│      ├── 5. Get all user public keys                            │
│      ├── 6. Encrypt new symmetric key for each user             │
│      ├── 7. Re-encrypt all secret files                         │
│      └── 8. Write everything to disk                            │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Acceptance Criteria

- [x] `SyncSecrets()` function implemented in `internal/secrets/sync.go`
- [x] Function accepts options struct for flexibility (dry-run, excluded users, etc.)
- [x] All operations happen in memory; disk writes only at the end
- [x] If any step fails, no partial writes occur (atomic operation)
- [x] Verbose/debug logging throughout
- [x] Returns detailed result struct for caller to report

### Technical Details

#### Files to Create

| File | Purpose |
|------|---------|
| `internal/secrets/sync.go` | Core sync logic |

#### Data Structures

```go
// SyncOptions configures the sync operation.
type SyncOptions struct {
    // DryRun if true, simulates the operation without writing files.
    DryRun bool
    
    // ExcludeUsers is a list of user UUIDs to exclude from re-encryption.
    // Used by revoke to exclude the user being removed.
    ExcludeUsers []string
    
    // Verbose enables detailed logging.
    Verbose bool
}

// SyncResult contains the results of a sync operation.
type SyncResult struct {
    // FilesProcessed is the number of secret files re-encrypted.
    FilesProcessed int
    
    // UsersProcessed is the number of users who received the new key.
    UsersProcessed int
    
    // NewSymmetricKeyGenerated indicates if a new key was created.
    NewSymmetricKeyGenerated bool
    
    // Errors contains any non-fatal errors encountered.
    Errors []error
}
```

#### Function Signature

```go
// SyncSecrets re-encrypts all secrets with a new symmetric key.
// The privateKey is used to decrypt the current symmetric key.
// Returns a SyncResult with details of the operation.
func SyncSecrets(privateKey *rsa.PrivateKey, opts SyncOptions) (*SyncResult, error) {
    // Implementation
}
```

#### Implementation Steps

1. **Load project and user configuration:**
   ```go
   projectConfig, err := configs.LoadProjectConfig()
   userConfig, err := configs.LoadUserConfig()
   userUUID := userConfig.User.UUID
   ```

2. **Get current symmetric key:**
   ```go
   encryptedSymKey, err := GetProjectKanukaKey(userUUID)
   symKey, err := DecryptWithPrivateKey(encryptedSymKey, privateKey)
   ```

3. **Find all secret files:**
   ```go
   // Find .kanuka files in project (excluding .kanuka/secrets/ which has user keys)
   secretFiles, err := FindEnvOrKanukaFiles(projectPath, []string{}, true)
   ```

4. **Decrypt all files to memory:**
   ```go
   type decryptedFile struct {
       originalPath string
       plaintext    []byte
   }
   
   var decryptedFiles []decryptedFile
   for _, path := range secretFiles {
       ciphertext, _ := os.ReadFile(path)
       plaintext, _ := DecryptWithSymmetricKey(ciphertext, symKey)
       decryptedFiles = append(decryptedFiles, decryptedFile{path, plaintext})
   }
   ```

5. **Generate new symmetric key:**
   ```go
   newSymKey := make([]byte, 32)
   rand.Read(newSymKey)
   ```

6. **Get all user public keys (excluding specified users):**
   ```go
   publicKeysDir := configs.ProjectKanukaSettings.ProjectPublicKeyPath
   entries, _ := os.ReadDir(publicKeysDir)
   
   var userPublicKeys []struct {
       UUID      string
       PublicKey *rsa.PublicKey
   }
   
   for _, entry := range entries {
       uuid := strings.TrimSuffix(entry.Name(), ".pub")
       if contains(opts.ExcludeUsers, uuid) {
           continue
       }
       pubKey, _ := LoadPublicKey(filepath.Join(publicKeysDir, entry.Name()))
       userPublicKeys = append(userPublicKeys, struct{...}{uuid, pubKey})
   }
   ```

7. **Encrypt symmetric key for each user:**
   ```go
   userKanukaFiles := make(map[string][]byte) // uuid -> encrypted sym key
   for _, user := range userPublicKeys {
       encrypted, _ := EncryptWithPublicKey(newSymKey, user.PublicKey)
       userKanukaFiles[user.UUID] = encrypted
   }
   ```

8. **Re-encrypt all secret files:**
   ```go
   reencryptedFiles := make(map[string][]byte) // path -> ciphertext
   for _, df := range decryptedFiles {
       ciphertext, _ := EncryptWithSymmetricKey(df.plaintext, newSymKey)
       reencryptedFiles[df.originalPath] = ciphertext
   }
   ```

9. **Write everything to disk (if not dry-run):**
   ```go
   if !opts.DryRun {
       // Write user .kanuka files
       for uuid, data := range userKanukaFiles {
           path := filepath.Join(secretsDir, uuid+".kanuka")
           os.WriteFile(path, data, 0600)
       }
       
       // Write secret files
       for path, data := range reencryptedFiles {
           os.WriteFile(path, data, 0600)
       }
   }
   ```

#### Error Handling

The function should be atomic - if any step fails after decryption, no files should be written:

```go
// Collect all data first, then write all at once
// If any decryption fails, return error before any writes
// If any write fails, we have a problem (consider backup/restore)
```

#### Security Considerations

- Plaintext is held in memory only
- Use `defer` to zero out sensitive byte slices when done (defense in depth)
- New symmetric key generated using `crypto/rand`

### Testing Requirements

#### Unit Tests

Create `internal/secrets/sync_test.go`:

| Test Case | Description |
|-----------|-------------|
| `TestSyncSecrets_Basic` | Sync with single user, verify files re-encrypted |
| `TestSyncSecrets_MultipleUsers` | Sync with multiple users, all get new key |
| `TestSyncSecrets_ExcludeUser` | Sync excluding one user, verify they don't get new key |
| `TestSyncSecrets_DryRun` | Dry run, verify no files written |
| `TestSyncSecrets_NoSecretFiles` | Project with no encrypted files, should succeed |
| `TestSyncSecrets_DecryptionFailure` | Invalid symmetric key, should fail before any writes |

### Definition of Done

- [ ] `SyncSecrets()` function implemented
- [ ] `SyncOptions` and `SyncResult` structs defined
- [ ] All operations atomic (no partial writes on failure)
- [ ] Unit tests pass
- [ ] `golangci-lint run` passes

---

## KAN-003: Sync Command

### Summary

Implement `kanuka secrets sync` command that exposes the sync functionality to users.

### Priority

**High** - Enables manual key rotation

### Context

The sync command allows users to manually rotate the project's symmetric encryption key. This is useful for:

1. Security hygiene (periodic key rotation)
2. After adding new team members (ensure everyone has access)
3. If you suspect a key may have been compromised

### Dependencies

- **KAN-002** (Core Sync Logic) must be completed first

### Current Behavior

Command does not exist.

### New Behavior

```bash
# Normal sync
$ kanuka secrets sync
⠋ Syncing secrets...
  Decrypting 5 secret files...
  Generating new encryption key...
  Re-encrypting for 4 users...
  Re-encrypting 5 secret files...
✓ Secrets synced successfully
  New encryption key generated and distributed to all users.

# Dry run
$ kanuka secrets sync --dry-run
[dry-run] Would sync secrets:
  - Decrypt 5 secret files
  - Generate new encryption key
  - Re-encrypt for 4 users:
    - alice@example.com
    - bob@example.com
    - charlie@example.com
    - you@example.com
  - Re-encrypt 5 secret files

No changes made.

# No secrets to sync
$ kanuka secrets sync
✓ No encrypted files found. Nothing to sync.
```

### Acceptance Criteria

- [x] `kanuka secrets sync` command implemented
- [x] `--dry-run` flag shows what would happen without making changes
- [x] `--verbose` flag shows detailed progress
- [x] Proper error handling if user doesn't have access
- [x] Works with both PKCS#1 and OpenSSH private key formats
- [x] Spinner shows progress during operation

### Technical Details

#### Files to Create

| File | Purpose |
|------|---------|
| `cmd/secrets_sync.go` | Command implementation |

#### Command Structure

```go
var syncDryRun bool

func init() {
    syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "preview sync without making changes")
}

var syncCmd = &cobra.Command{
    Use:   "sync",
    Short: "Re-encrypt all secrets with a new symmetric key",
    Long: `Re-encrypts all secret files with a newly generated symmetric key.

This command is useful for:
  - Periodic security key rotation
  - After adding new team members
  - If you suspect a key may have been compromised

All users with access will receive the new symmetric key, encrypted
with their public key. The old symmetric key will no longer work.

Use --dry-run to preview what would happen without making changes.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
    },
}
```

#### Implementation Steps

1. **Initialize project settings:**
   ```go
   if err := configs.InitProjectSettings(); err != nil {
       return err
   }
   ```

2. **Load private key:**
   ```go
   projectConfig, _ := configs.LoadProjectConfig()
   projectUUID := projectConfig.Project.UUID
   privateKeyPath := configs.GetPrivateKeyPath(projectUUID)
   privateKey, err := secrets.LoadPrivateKey(privateKeyPath)
   ```

3. **Call sync function:**
   ```go
   opts := secrets.SyncOptions{
       DryRun:  syncDryRun,
       Verbose: verbose,
   }
   
   result, err := secrets.SyncSecrets(privateKey, opts)
   ```

4. **Display results:**
   ```go
   if syncDryRun {
       fmt.Println("[dry-run] Would sync secrets:")
       fmt.Printf("  - Decrypt %d secret files\n", result.FilesProcessed)
       // ... etc
   } else {
       fmt.Println("✓ Secrets synced successfully")
       fmt.Println("  New encryption key generated and distributed to all users.")
   }
   ```

### Testing Requirements

#### Integration Tests

Create `test/integration/sync/sync_test.go`:

| Test Case | Description |
|-----------|-------------|
| `TestSync_Basic` | Sync project, verify all files re-encrypted |
| `TestSync_DryRun` | Dry run, verify no files changed |
| `TestSync_MultipleUsers` | Sync with multiple users, all can decrypt after |
| `TestSync_NoSecrets` | Project with no encrypted files |
| `TestSync_NotInitialized` | Run in non-kanuka project, proper error |

### Definition of Done

- [x] `kanuka secrets sync` command implemented
- [x] `--dry-run` flag works correctly
- [x] Help text is clear and complete
- [ ] Integration tests pass
- [x] `golangci-lint run` passes

---

## KAN-004: Revoke Security Fix

### Summary

Fix the security gap in `kanuka secrets revoke` by performing full re-encryption after removing a user.

### Priority

**Critical** - Security vulnerability

### Context

**This is a security fix.** The current `revoke` command has a critical flaw:

1. User A is revoked from the project
2. Their public key and `.kanuka` file are removed
3. The symmetric key is "rotated" (new key generated, encrypted for remaining users)
4. **BUT the actual secret files are NOT re-encrypted**

This means:
- If User A had previously copied the encrypted `.kanuka` secret files
- And they still have access to the OLD symmetric key (which they did, it was encrypted for them)
- They can decrypt all the secrets that existed at the time of revocation

**The fix:** After revoking a user, we must re-encrypt all secret files with a new symmetric key that the revoked user never had access to.

### Dependencies

- **KAN-002** (Core Sync Logic) must be completed first
- **KAN-003** (Sync Command) should be completed (not required, but recommended)

### Current Behavior

```bash
$ kanuka secrets revoke --user alice@example.com
✓ Revoked access for alice@example.com
# Files in .kanuka/secrets/*.kanuka are NOT re-encrypted
# Alice can still decrypt old files if she has copies
```

### New Behavior

```bash
$ kanuka secrets revoke --user alice@example.com
⠋ Revoking access for alice@example.com...
  Decrypting secrets...
  Removing user keys...
  Generating new encryption key...
  Re-encrypting secrets for 3 remaining users...
  Re-encrypting 5 secret files...
✓ Access revoked for alice@example.com
  All secrets have been re-encrypted with a new key.
```

### Acceptance Criteria

- [ ] After revoke, all secret files are re-encrypted with a new symmetric key
- [ ] Revoked user's old symmetric key cannot decrypt the new files
- [ ] All remaining users can still decrypt with their keys
- [ ] Operation is atomic (if it fails, nothing changes)
- [ ] Verbose output shows progress of re-encryption

### Technical Details

#### Files to Modify

| File | Changes |
|------|---------|
| `cmd/secrets_revoke.go` | Call sync after removing user |

#### Current Implementation (Simplified)

```go
// Current revoke flow:
// 1. Delete user's public key
// 2. Delete user's .kanuka file
// 3. Generate new symmetric key
// 4. Encrypt for remaining users
// 5. MISSING: Re-encrypt secret files
```

#### New Implementation

```go
// New revoke flow:
// 1. Delete user's public key
// 2. Delete user's .kanuka file  
// 3. Call SyncSecrets() which:
//    - Decrypts all secret files
//    - Generates new symmetric key
//    - Encrypts for remaining users (excluding revoked user)
//    - Re-encrypts all secret files
```

#### Implementation Steps

1. **After deleting user's files, call sync:**
   ```go
   // Delete user's public key
   os.Remove(publicKeyPath)
   
   // Delete user's .kanuka file
   os.Remove(kanukaFilePath)
   
   // Re-encrypt everything with new key (excluding revoked user)
   opts := secrets.SyncOptions{
       ExcludeUsers: []string{revokedUserUUID},
       Verbose:      verbose,
   }
   
   result, err := secrets.SyncSecrets(privateKey, opts)
   if err != nil {
       return fmt.Errorf("failed to re-encrypt secrets: %w", err)
   }
   ```

2. **Update output messages:**
   ```go
   finalMessage := color.GreenString("✓") + " Access revoked for " + userEmail + "\n" +
       "  All secrets have been re-encrypted with a new key."
   ```

#### Edge Cases

| Case | Handling |
|------|----------|
| Revoking last user | Should fail with error (can't revoke yourself if you're the only user) |
| No secret files | Should succeed (just delete user keys, no re-encryption needed) |
| Revoked user is current user | Should fail with clear error message |

### Testing Requirements

#### Integration Tests

Update existing tests in `test/integration/revoke/`:

| Test Case | Description |
|-----------|-------------|
| `TestRevoke_ReencryptsFiles` | After revoke, verify old symmetric key can't decrypt new files |
| `TestRevoke_RemainingUsersCanDecrypt` | After revoke, remaining users can still decrypt |
| `TestRevoke_AtomicOnFailure` | If sync fails, user keys should not be deleted |

### Security Verification

To verify the fix works:

1. Set up project with 2 users (Alice, Bob)
2. Encrypt some secrets
3. Copy the encrypted files somewhere
4. Note Alice's symmetric key
5. Revoke Alice
6. Try to decrypt the copied files with Alice's old symmetric key
7. **Should fail** - this proves the fix works

### Definition of Done

- [ ] Revoke calls sync after removing user
- [ ] Secret files are re-encrypted with new key
- [ ] Old symmetric key cannot decrypt new files
- [ ] Remaining users can still decrypt
- [ ] Integration tests pass
- [ ] `golangci-lint run` passes

---

## KAN-005: Access Command

### Summary

Implement `kanuka secrets access` command to list all users who have access to the project's secrets.

### Priority

**High** - High value, low effort

### Context

Users need visibility into who has access to their project's secrets. Currently, you have to manually inspect the `.kanuka/public_keys/` and `.kanuka/secrets/` directories.

The access command provides a clear overview of:
- Who has full access (can decrypt)
- Who is pending (registered but needs sync)
- Any orphaned state (inconsistencies that need cleanup)

### Current Behavior

Command does not exist. Users must manually:
```bash
$ ls .kanuka/public_keys/
alice-uuid.pub  bob-uuid.pub

$ ls .kanuka/secrets/
alice-uuid.kanuka  bob-uuid.kanuka  .env.kanuka
```

### New Behavior

```bash
$ kanuka secrets access
Project: my-project

Users with access:

  UUID                                    EMAIL                     STATUS
  a1b2c3d4-e5f6-7890-abcd-ef1234567890    alice@example.com         ✓ active
  b2c3d4e5-f6a7-8901-bcde-f12345678901    bob@example.com           ✓ active
  c3d4e5f6-a7b8-9012-cdef-123456789012    charlie@example.com       ⚠ pending

Legend:
  ✓ active  - User has public key and encrypted symmetric key
  ⚠ pending - User has public key but no encrypted symmetric key (run 'sync')
  ✗ orphan  - Encrypted symmetric key exists but no public key (inconsistent)

Total: 3 users (2 active, 1 pending)

# JSON output for scripting
$ kanuka secrets access --json
{
  "project": "my-project",
  "users": [
    {"uuid": "a1b2c3d4-...", "email": "alice@example.com", "status": "active"},
    {"uuid": "b2c3d4e5-...", "email": "bob@example.com", "status": "active"},
    {"uuid": "c3d4e5f6-...", "email": "charlie@example.com", "status": "pending"}
  ],
  "summary": {"active": 2, "pending": 1, "orphan": 0}
}
```

### Acceptance Criteria

- [ ] `kanuka secrets access` command implemented
- [ ] Shows all users with their UUID, email (if available), and status
- [ ] Status correctly determined based on file existence
- [ ] `--json` flag outputs machine-readable JSON
- [ ] Summary shows counts by status
- [ ] Works in non-initialized projects (shows helpful error)

### Technical Details

#### Files to Create

| File | Purpose |
|------|---------|
| `cmd/secrets_access.go` | Command implementation |

#### Status Determination Logic

```go
type UserStatus string

const (
    StatusActive  UserStatus = "active"   // Has public key AND .kanuka file
    StatusPending UserStatus = "pending"  // Has public key but NO .kanuka file
    StatusOrphan  UserStatus = "orphan"   // Has .kanuka file but NO public key
)

func determineUserStatus(uuid string) UserStatus {
    publicKeyPath := filepath.Join(publicKeysDir, uuid+".pub")
    kanukaPath := filepath.Join(secretsDir, uuid+".kanuka")
    
    hasPublicKey := fileExists(publicKeyPath)
    hasKanukaFile := fileExists(kanukaPath)
    
    switch {
    case hasPublicKey && hasKanukaFile:
        return StatusActive
    case hasPublicKey && !hasKanukaFile:
        return StatusPending
    case !hasPublicKey && hasKanukaFile:
        return StatusOrphan
    default:
        // Neither exists - not a user (shouldn't happen if we're iterating)
        return ""
    }
}
```

#### Getting User Email from UUID

The email might be stored in:
1. The public key file comment (SSH keys often have email as comment)
2. A metadata file
3. The project config

Check existing code for how emails are associated with UUIDs.

#### Implementation Steps

1. **List all public keys and .kanuka files:**
   ```go
   publicKeyFiles, _ := os.ReadDir(publicKeysDir)
   kanukaFiles, _ := os.ReadDir(secretsDir)
   
   // Build set of all UUIDs
   uuids := make(map[string]bool)
   for _, f := range publicKeyFiles {
       if strings.HasSuffix(f.Name(), ".pub") {
           uuid := strings.TrimSuffix(f.Name(), ".pub")
           uuids[uuid] = true
       }
   }
   for _, f := range kanukaFiles {
       if strings.HasSuffix(f.Name(), ".kanuka") {
           uuid := strings.TrimSuffix(f.Name(), ".kanuka")
           uuids[uuid] = true
       }
   }
   ```

2. **Determine status for each user:**
   ```go
   type UserInfo struct {
       UUID   string
       Email  string
       Status UserStatus
   }
   
   var users []UserInfo
   for uuid := range uuids {
       status := determineUserStatus(uuid)
       email := getEmailForUUID(uuid) // Implement based on how emails are stored
       users = append(users, UserInfo{uuid, email, status})
   }
   ```

3. **Output results:**
   ```go
   if jsonOutput {
       json.NewEncoder(os.Stdout).Encode(result)
   } else {
       // Pretty print table
   }
   ```

### Testing Requirements

#### Integration Tests

Create `test/integration/access/access_test.go`:

| Test Case | Description |
|-----------|-------------|
| `TestAccess_SingleActiveUser` | One user with full access |
| `TestAccess_MultipleUsers` | Multiple users with different statuses |
| `TestAccess_PendingUser` | User with public key but no .kanuka file |
| `TestAccess_OrphanUser` | .kanuka file with no public key |
| `TestAccess_JsonOutput` | Verify JSON output format |
| `TestAccess_NotInitialized` | Run in non-kanuka project |

### Definition of Done

- [ ] `kanuka secrets access` command implemented
- [ ] Status correctly determined for all user states
- [ ] `--json` flag works
- [ ] Output is clear and readable
- [ ] Integration tests pass
- [ ] `golangci-lint run` passes

---

## KAN-006: Clean Command

### Summary

Implement `kanuka secrets clean` command to remove orphaned keys and clean up inconsistent state detected by the `access` command.

### Priority

**High** - Natural follow-up to KAN-005, completes the access workflow

### Context

When users run `kanuka secrets access`, they may discover orphaned state:
- `.kanuka` files without corresponding public keys (orphan status)
- This can happen if someone manually deleted a public key, or if a revoke operation was interrupted

The `clean` command provides a safe way to remove this inconsistent state. It's the suggested action when `access` shows orphaned entries.

### Dependencies

- **KAN-005** (Access Command) should be completed first (clean is the suggested follow-up)

### Current Behavior

Command does not exist. Users must manually:
```bash
$ kanuka secrets access
  ...
  c3d4e5f6-a7b8-9012-cdef-123456789012    unknown                   ✗ orphan
  ...

# User has to manually figure out what to delete
$ rm .kanuka/secrets/c3d4e5f6-a7b8-9012-cdef-123456789012.kanuka
```

### New Behavior

```bash
# After seeing orphaned entries in access output
$ kanuka secrets access
  ...
  c3d4e5f6-a7b8-9012-cdef-123456789012    unknown                   ✗ orphan

Tip: Run 'kanuka secrets clean' to remove orphaned entries.

# Clean command with confirmation
$ kanuka secrets clean
Found 1 orphaned entry:

  UUID                                    FILE
  c3d4e5f6-a7b8-9012-cdef-123456789012    .kanuka/secrets/c3d4e5f6-a7b8-9012-cdef-123456789012.kanuka

This will permanently delete the orphaned files listed above.
These files cannot be recovered.

Do you want to continue? [y/N]: y
✓ Removed 1 orphaned file

# Clean with --force flag (no confirmation)
$ kanuka secrets clean --force
✓ Removed 1 orphaned file

# Dry run
$ kanuka secrets clean --dry-run
[dry-run] Would remove 1 orphaned file:
  .kanuka/secrets/c3d4e5f6-a7b8-9012-cdef-123456789012.kanuka

No changes made.

# No orphans to clean
$ kanuka secrets clean
✓ No orphaned entries found. Nothing to clean.
```

### Acceptance Criteria

- [ ] `kanuka secrets clean` command implemented
- [ ] Finds and removes orphaned `.kanuka` files (those without corresponding public keys)
- [ ] Confirmation prompt before deletion (unless `--force` is provided)
- [ ] `--force` flag to skip confirmation
- [ ] `--dry-run` flag to show what would be deleted without making changes
- [ ] Clear output showing which files will be/were removed
- [ ] Works in non-initialized projects (shows helpful error)

### Technical Details

#### Files to Create

| File | Purpose |
|------|---------|
| `cmd/secrets_clean.go` | Command implementation |

#### Orphan Detection Logic

```go
// An orphan is a .kanuka file in .kanuka/secrets/ that has no corresponding
// public key in .kanuka/public_keys/
func findOrphanedEntries() ([]OrphanEntry, error) {
    secretsDir := configs.ProjectKanukaSettings.ProjectSecretsPath
    publicKeysDir := configs.ProjectKanukaSettings.ProjectPublicKeyPath
    
    var orphans []OrphanEntry
    
    entries, _ := os.ReadDir(secretsDir)
    for _, entry := range entries {
        if !strings.HasSuffix(entry.Name(), ".kanuka") {
            continue
        }
        
        uuid := strings.TrimSuffix(entry.Name(), ".kanuka")
        publicKeyPath := filepath.Join(publicKeysDir, uuid+".pub")
        
        if !fileExists(publicKeyPath) {
            orphans = append(orphans, OrphanEntry{
                UUID:     uuid,
                FilePath: filepath.Join(secretsDir, entry.Name()),
            })
        }
    }
    
    return orphans, nil
}

type OrphanEntry struct {
    UUID     string
    FilePath string
}
```

#### Command Structure

```go
var cleanForce bool
var cleanDryRun bool

func init() {
    cleanCmd.Flags().BoolVar(&cleanForce, "force", false, "skip confirmation prompt")
    cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "show what would be removed without making changes")
}

var cleanCmd = &cobra.Command{
    Use:   "clean",
    Short: "Remove orphaned keys and inconsistent state",
    Long: `Removes orphaned entries detected by 'kanuka secrets access'.

An orphan is a .kanuka file that has no corresponding public key.
This can happen if:
  - A public key was manually deleted
  - A revoke operation was interrupted
  - Files were corrupted or partially restored

Use --dry-run to preview what would be removed.
Use --force to skip the confirmation prompt.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
    },
}
```

#### Implementation Steps

1. **Initialize project settings:**
   ```go
   if err := configs.InitProjectSettings(); err != nil {
       return err
   }
   ```

2. **Find orphaned entries:**
   ```go
   orphans, err := findOrphanedEntries()
   if err != nil {
       return fmt.Errorf("failed to find orphaned entries: %w", err)
   }
   
   if len(orphans) == 0 {
       fmt.Println("✓ No orphaned entries found. Nothing to clean.")
       return nil
   }
   ```

3. **Display orphans:**
   ```go
   if cleanDryRun {
       fmt.Printf("[dry-run] Would remove %d orphaned file(s):\n", len(orphans))
   } else {
       fmt.Printf("Found %d orphaned entry(ies):\n\n", len(orphans))
   }
   
   fmt.Println("  UUID                                    FILE")
   for _, o := range orphans {
       fmt.Printf("  %s    %s\n", o.UUID, o.FilePath)
   }
   ```

4. **Confirm deletion (if not --force or --dry-run):**
   ```go
   if cleanDryRun {
       fmt.Println("\nNo changes made.")
       return nil
   }
   
   if !cleanForce {
       fmt.Println("\nThis will permanently delete the orphaned files listed above.")
       fmt.Println("These files cannot be recovered.")
       fmt.Println()
       
       if !confirmAction("Do you want to continue?") {
           fmt.Println("Aborted.")
           return nil
       }
   }
   ```

5. **Remove orphaned files:**
   ```go
   for _, o := range orphans {
       if err := os.Remove(o.FilePath); err != nil {
           return fmt.Errorf("failed to remove %s: %w", o.FilePath, err)
       }
   }
   
   fmt.Printf("✓ Removed %d orphaned file(s)\n", len(orphans))
   ```

#### Update Access Command Output

When implementing this, also update `KAN-005` to show a tip when orphans are found:

```go
// At the end of access command output, if orphans exist:
if orphanCount > 0 {
    fmt.Println()
    fmt.Println("Tip: Run 'kanuka secrets clean' to remove orphaned entries.")
}
```

### Testing Requirements

#### Integration Tests

Create `test/integration/clean/clean_test.go`:

| Test Case | Description |
|-----------|-------------|
| `TestClean_NoOrphans` | No orphaned entries, shows success message |
| `TestClean_SingleOrphan` | Single orphan removed with confirmation |
| `TestClean_MultipleOrphans` | Multiple orphans removed |
| `TestClean_Force` | --force skips confirmation |
| `TestClean_DryRun` | --dry-run shows files but doesn't delete |
| `TestClean_AbortOnDecline` | User declines, no files deleted |
| `TestClean_NotInitialized` | Run in non-kanuka project, proper error |

#### Test Setup

```go
// Helper to create orphaned state for testing
func createOrphanedEntry(t *testing.T, secretsDir, uuid string) {
    kanukaPath := filepath.Join(secretsDir, uuid+".kanuka")
    // Create .kanuka file without corresponding public key
    err := os.WriteFile(kanukaPath, []byte("dummy-encrypted-key"), 0600)
    require.NoError(t, err)
}
```

### Definition of Done

- [ ] `kanuka secrets clean` command implemented
- [ ] Finds orphaned entries correctly
- [ ] Confirmation prompt works
- [ ] `--force` flag skips confirmation
- [ ] `--dry-run` flag shows preview without changes
- [ ] Help text is clear and complete
- [ ] Integration tests pass
- [ ] `golangci-lint run` passes
- [ ] Access command updated to suggest clean when orphans found

---

## KAN-007: Status Command

### Summary

Implement `kanuka secrets status` command to show the encryption status of all secret files.

### Priority

**High** - High value, low effort

### Context

### Context

Users need to know:
- Which files are encrypted and up to date
- Which files have been modified since encryption (stale)
- Which files are not encrypted (security risk)

This helps users understand their security posture and know when to run `encrypt`.

### Current Behavior

Command does not exist. Users must manually check file modification times:
```bash
$ ls -la .env .env.kanuka
-rw-r--r--  1 user  staff  100 Jan 10 10:00 .env
-rw-------  1 user  staff  200 Jan  9 09:00 .env.kanuka  # Older = stale!
```

### New Behavior

```bash
$ kanuka secrets status
Project: my-project
Secret files status:

  FILE                      STATUS
  .env                      ✓ encrypted (up to date)
  .env.local                ✓ encrypted (up to date)
  config/.env.production    ⚠ stale (plaintext modified after encryption)
  scripts/.env.test         ✗ not encrypted
  .env.backup.kanuka        ◌ encrypted only (no plaintext)

Summary:
  2 files up to date
  1 file stale (run 'kanuka secrets encrypt' to update)
  1 file not encrypted (run 'kanuka secrets encrypt' to secure)
  1 file encrypted only (plaintext removed, this is normal)

# JSON output
$ kanuka secrets status --json
{
  "files": [
    {"path": ".env", "status": "current", "plaintextMtime": "...", "encryptedMtime": "..."},
    ...
  ],
  "summary": {"current": 2, "stale": 1, "unencrypted": 1, "encryptedOnly": 1}
}
```

### Acceptance Criteria

- [ ] `kanuka secrets status` command implemented
- [ ] Correctly identifies all file states (current, stale, unencrypted, encrypted-only)
- [ ] Shows relative paths from project root
- [ ] `--json` flag outputs machine-readable JSON
- [ ] Summary with actionable suggestions
- [ ] Recursively finds files in subdirectories

### Technical Details

#### Files to Create

| File | Purpose |
|------|---------|
| `cmd/secrets_status.go` | Command implementation |

#### Status Determination Logic

```go
type FileStatus string

const (
    StatusCurrent       FileStatus = "current"        // Encrypted file newer than plaintext
    StatusStale         FileStatus = "stale"          // Plaintext newer than encrypted
    StatusUnencrypted   FileStatus = "unencrypted"    // Plaintext exists, no encrypted
    StatusEncryptedOnly FileStatus = "encrypted_only" // Encrypted exists, no plaintext
)

func determineFileStatus(envPath string) FileStatus {
    kanukaPath := envPath + ".kanuka"
    
    envExists := fileExists(envPath)
    kanukaExists := fileExists(kanukaPath)
    
    switch {
    case envExists && kanukaExists:
        envMtime := getModTime(envPath)
        kanukaMtime := getModTime(kanukaPath)
        if kanukaMtime.After(envMtime) {
            return StatusCurrent
        }
        return StatusStale
        
    case envExists && !kanukaExists:
        return StatusUnencrypted
        
    case !envExists && kanukaExists:
        return StatusEncryptedOnly
        
    default:
        // Neither exists - shouldn't happen
        return ""
    }
}
```

#### Implementation Steps

1. **Find all relevant files:**
   ```go
   // Find all .env* files (excluding .kanuka directory)
   envFiles, _ := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, false)
   
   // Find all .kanuka files
   kanukaFiles, _ := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, true)
   ```

2. **Build unified list:**
   ```go
   // Create map of base paths (without .kanuka extension)
   allPaths := make(map[string]bool)
   for _, f := range envFiles {
       allPaths[f] = true
   }
   for _, f := range kanukaFiles {
       basePath := strings.TrimSuffix(f, ".kanuka")
       allPaths[basePath] = true
   }
   ```

3. **Determine status for each:**
   ```go
   type FileInfo struct {
       Path   string
       Status FileStatus
   }
   
   var files []FileInfo
   for path := range allPaths {
       status := determineFileStatus(path)
       files = append(files, FileInfo{path, status})
   }
   ```

4. **Sort and display:**
   ```go
   // Sort by path for consistent output
   sort.Slice(files, func(i, j int) bool {
       return files[i].Path < files[j].Path
   })
   ```

### Testing Requirements

#### Integration Tests

Create `test/integration/status/status_test.go`:

| Test Case | Description |
|-----------|-------------|
| `TestStatus_AllCurrent` | All files encrypted and up to date |
| `TestStatus_StaleFile` | Plaintext modified after encryption |
| `TestStatus_UnencryptedFile` | Plaintext without encryption |
| `TestStatus_EncryptedOnlyFile` | Encrypted without plaintext |
| `TestStatus_MixedStates` | Files in various states |
| `TestStatus_Subdirectories` | Files in nested directories |
| `TestStatus_JsonOutput` | Verify JSON format |
| `TestStatus_NoFiles` | Project with no .env files |

### Definition of Done

- [ ] `kanuka secrets status` command implemented
- [ ] All file states correctly identified
- [ ] Relative paths shown
- [ ] `--json` flag works
- [ ] Summary with suggestions
- [ ] Integration tests pass
- [ ] `golangci-lint run` passes

---

## KAN-008: Doctor Command

### Summary

Implement `kanuka secrets doctor` command to check project health and detect common issues.

### Priority

**Medium** - Nice to have for project hygiene

### Context

Users may have various configuration issues, permission problems, or inconsistent states. The doctor command runs a series of health checks and provides actionable suggestions.

This is similar to `brew doctor`, `npm doctor`, etc.

Note: This command can reuse logic from KAN-005 (Access), KAN-006 (Clean), and KAN-007 (Status) for some checks.

### New Behavior

```bash
$ kanuka secrets doctor
Running health checks...

✓ Project configuration valid
✓ User configuration valid
✓ Private key exists for this project
✓ Private key permissions correct (0600)
✓ All public keys have corresponding .kanuka files
✓ All .kanuka files have corresponding public keys
✓ .env patterns found in .gitignore
⚠ Found 1 unencrypted .env file (run 'kanuka secrets status')
✗ 2 .env files not in .gitignore

Summary: 7 passed, 1 warning, 1 error

Suggestions:
  - Run 'kanuka secrets encrypt' to encrypt unprotected files
  - Add '.env*' to your .gitignore file
```

### Acceptance Criteria

- [ ] `kanuka secrets doctor` command implemented
- [ ] Runs all defined health checks
- [ ] Clear pass/warning/error indicators
- [ ] Actionable suggestions for each issue
- [ ] `--json` flag for scripting
- [ ] Exit code reflects health (0 = healthy, 1 = warnings, 2 = errors)

### Technical Details

#### Files to Create

| File | Purpose |
|------|---------|
| `cmd/secrets_doctor.go` | Command implementation |
| `internal/secrets/doctor.go` | Health check implementations |

#### Health Check Structure

```go
type CheckResult struct {
    Name        string
    Status      CheckStatus // Pass, Warning, Error
    Message     string
    Suggestion  string
}

type CheckStatus int

const (
    CheckPass CheckStatus = iota
    CheckWarning
    CheckError
)

type HealthCheck func() CheckResult
```

#### Checks to Implement

| Check | Severity | Description |
|-------|----------|-------------|
| `checkProjectConfig` | Error | `.kanuka/config.toml` exists and parses |
| `checkUserConfig` | Error | User config exists and parses |
| `checkPrivateKeyExists` | Error | Private key exists for this project |
| `checkPrivateKeyPermissions` | Warning | Private key has 0600 permissions |
| `checkPublicKeyConsistency` | Error | Every public key has a .kanuka file |
| `checkKanukaFileConsistency` | Error | Every .kanuka file has a public key |
| `checkGitignore` | Warning | .env patterns in .gitignore |
| `checkUnencryptedFiles` | Warning | No unencrypted .env files |

#### Implementation Steps

1. **Define all checks:**
   ```go
   var checks = []HealthCheck{
       checkProjectConfig,
       checkUserConfig,
       checkPrivateKeyExists,
       checkPrivateKeyPermissions,
       checkPublicKeyConsistency,
       checkKanukaFileConsistency,
       checkGitignore,
       checkUnencryptedFiles,
   }
   ```

2. **Run all checks:**
   ```go
   var results []CheckResult
   for _, check := range checks {
       result := check()
       results = append(results, result)
   }
   ```

3. **Display results:**
   ```go
   for _, r := range results {
       switch r.Status {
       case CheckPass:
           fmt.Printf("✓ %s\n", r.Name)
       case CheckWarning:
           fmt.Printf("⚠ %s\n", r.Message)
       case CheckError:
           fmt.Printf("✗ %s\n", r.Message)
       }
   }
   ```

4. **Determine exit code:**
   ```go
   hasError := false
   hasWarning := false
   for _, r := range results {
       if r.Status == CheckError {
           hasError = true
       }
       if r.Status == CheckWarning {
           hasWarning = true
       }
   }
   
   if hasError {
       os.Exit(2)
   } else if hasWarning {
       os.Exit(1)
   }
   os.Exit(0)
   ```

### Testing Requirements

#### Integration Tests

Create `test/integration/doctor/doctor_test.go`:

| Test Case | Description |
|-----------|-------------|
| `TestDoctor_HealthyProject` | All checks pass |
| `TestDoctor_MissingPrivateKey` | Private key doesn't exist |
| `TestDoctor_BadPermissions` | Private key has wrong permissions |
| `TestDoctor_InconsistentState` | Public key without .kanuka file |
| `TestDoctor_MissingGitignore` | .env not in .gitignore |
| `TestDoctor_JsonOutput` | Verify JSON format |
| `TestDoctor_ExitCodes` | Verify correct exit codes |

### Definition of Done

- [ ] `kanuka secrets doctor` command implemented
- [ ] All health checks implemented
- [ ] Clear output with suggestions
- [ ] `--json` flag works
- [ ] Exit codes correct
- [ ] Integration tests pass
- [ ] `golangci-lint run` passes

---

## KAN-009: Rotate Command

### Summary

Implement `kanuka secrets rotate` command to rotate the current user's keypair.

### Priority

**Medium** - Security best practice

### Context

Users should be able to rotate their personal keypair without affecting other users. This is useful for:

1. Security hygiene (periodic rotation)
2. If a private key may have been compromised
3. When changing machines

### New Behavior

```bash
$ kanuka secrets rotate
⚠ This will generate a new keypair and replace your current one.
  Your old private key will no longer work for this project.

Do you want to continue? [y/N]: y

⠋ Rotating your keypair...
  Generating new keypair...
  Decrypting symmetric key with old private key...
  Re-encrypting symmetric key with new public key...
  Updating public key in project...
  Saving new private key...
✓ Keypair rotated successfully

Your new public key has been added to the project.
Other users do not need to take any action.

# With --force flag
$ kanuka secrets rotate --force
⠋ Rotating your keypair...
  ...
✓ Keypair rotated successfully
```

### Acceptance Criteria

- [ ] `kanuka secrets rotate` command implemented
- [ ] Generates new RSA keypair
- [ ] Decrypts symmetric key with old private key
- [ ] Re-encrypts symmetric key with new public key
- [ ] Updates public key in project
- [ ] Saves new private key to user's key directory
- [ ] Confirmation prompt (skippable with `--force`)
- [ ] Works with passphrase-protected keys

### Technical Details

#### Files to Create

| File | Purpose |
|------|---------|
| `cmd/secrets_rotate.go` | Command implementation |

#### Implementation Steps

1. **Load current private key:**
   ```go
   projectConfig, _ := configs.LoadProjectConfig()
   projectUUID := projectConfig.Project.UUID
   privateKeyPath := configs.GetPrivateKeyPath(projectUUID)
   oldPrivateKey, err := secrets.LoadPrivateKey(privateKeyPath)
   ```

2. **Get current encrypted symmetric key:**
   ```go
   userConfig, _ := configs.LoadUserConfig()
   userUUID := userConfig.User.UUID
   encryptedSymKey, _ := secrets.GetProjectKanukaKey(userUUID)
   ```

3. **Decrypt symmetric key with old private key:**
   ```go
   symKey, _ := secrets.DecryptWithPrivateKey(encryptedSymKey, oldPrivateKey)
   ```

4. **Generate new keypair:**
   ```go
   newPrivateKey, _ := rsa.GenerateKey(rand.Reader, 4096)
   newPublicKey := &newPrivateKey.PublicKey
   ```

5. **Re-encrypt symmetric key with new public key:**
   ```go
   newEncryptedSymKey, _ := secrets.EncryptWithPublicKey(symKey, newPublicKey)
   ```

6. **Write new files:**
   ```go
   // Update public key in project
   publicKeyPath := filepath.Join(publicKeysDir, userUUID+".pub")
   writePublicKey(publicKeyPath, newPublicKey)
   
   // Update user's .kanuka file
   kanukaPath := filepath.Join(secretsDir, userUUID+".kanuka")
   os.WriteFile(kanukaPath, newEncryptedSymKey, 0600)
   
   // Save new private key
   writePrivateKey(privateKeyPath, newPrivateKey)
   ```

### Testing Requirements

#### Integration Tests

Create `test/integration/rotate/rotate_test.go`:

| Test Case | Description |
|-----------|-------------|
| `TestRotate_Basic` | Rotate keypair, verify new key works |
| `TestRotate_OldKeyNoLongerWorks` | Old private key can't decrypt after rotate |
| `TestRotate_OtherUsersUnaffected` | Other users can still decrypt |
| `TestRotate_Force` | --force skips confirmation |

### Definition of Done

- [ ] `kanuka secrets rotate` command implemented
- [ ] New keypair generated correctly
- [ ] Old key no longer works
- [ ] Other users unaffected
- [ ] Confirmation prompt works
- [ ] Integration tests pass
- [ ] `golangci-lint run` passes

---

## KAN-010: Export Command

### Summary

Implement `kanuka secrets export` command to create a backup archive of encrypted secrets.

### Priority

**Lower** - Useful but not urgent

### Context

Users need to be able to backup their encrypted secrets for:
- Disaster recovery
- Migration to new systems
- Archival purposes

The export only includes encrypted data - private keys and plaintext are never exported.

### New Behavior

```bash
$ kanuka secrets export
✓ Exported secrets to kanuka-secrets-2024-01-15.tar.gz

Archive contents:
  .kanuka/config.toml
  .kanuka/public_keys/ (3 files)
  .kanuka/secrets/ (3 user keys)
  5 encrypted secret files

Note: This archive contains encrypted data only.
      Private keys are NOT included.
      
# Custom output path
$ kanuka secrets export -o /backups/project-secrets.tar.gz
✓ Exported secrets to /backups/project-secrets.tar.gz
```

### Acceptance Criteria

- [ ] `kanuka secrets export` command implemented
- [ ] Creates tar.gz archive with encrypted data
- [ ] Includes: config.toml, public_keys/*, secrets/*.kanuka, all *.kanuka files
- [ ] Does NOT include: private keys, plaintext .env files
- [ ] `-o` / `--output` flag for custom output path
- [ ] Default filename includes date
- [ ] Summary of archive contents shown

### Technical Details

#### Files to Create

| File | Purpose |
|------|---------|
| `cmd/secrets_export.go` | Command implementation |

#### Archive Structure

```
kanuka-secrets-2024-01-15.tar.gz
├── .kanuka/
│   ├── config.toml
│   ├── public_keys/
│   │   ├── user1-uuid.pub
│   │   └── user2-uuid.pub
│   └── secrets/
│       ├── user1-uuid.kanuka
│       └── user2-uuid.kanuka
├── .env.kanuka
├── .env.local.kanuka
└── config/.env.production.kanuka
```

#### Implementation Steps

1. **Collect files to archive:**
   ```go
   var filesToArchive []string
   
   // .kanuka directory contents
   filesToArchive = append(filesToArchive, ".kanuka/config.toml")
   publicKeys, _ := filepath.Glob(".kanuka/public_keys/*.pub")
   filesToArchive = append(filesToArchive, publicKeys...)
   userKanukaFiles, _ := filepath.Glob(".kanuka/secrets/*.kanuka")
   filesToArchive = append(filesToArchive, userKanukaFiles...)
   
   // All .kanuka secret files in project
   secretFiles, _ := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, true)
   filesToArchive = append(filesToArchive, secretFiles...)
   ```

2. **Create tar.gz archive:**
   ```go
   outputPath := fmt.Sprintf("kanuka-secrets-%s.tar.gz", time.Now().Format("2006-01-02"))
   if outputFlag != "" {
       outputPath = outputFlag
   }
   
   file, _ := os.Create(outputPath)
   gzWriter := gzip.NewWriter(file)
   tarWriter := tar.NewWriter(gzWriter)
   
   for _, filePath := range filesToArchive {
       addFileToTar(tarWriter, filePath)
   }
   
   tarWriter.Close()
   gzWriter.Close()
   file.Close()
   ```

### Testing Requirements

#### Integration Tests

Create `test/integration/export/export_test.go`:

| Test Case | Description |
|-----------|-------------|
| `TestExport_Basic` | Export creates valid archive |
| `TestExport_ContainsExpectedFiles` | Archive contains all expected files |
| `TestExport_ExcludesPrivateKey` | Private key not in archive |
| `TestExport_ExcludesPlaintext` | Plaintext .env not in archive |
| `TestExport_CustomOutput` | -o flag works |

### Definition of Done

- [ ] `kanuka secrets export` command implemented
- [ ] Creates valid tar.gz archive
- [ ] Contains all encrypted data
- [ ] Excludes sensitive data
- [ ] `-o` flag works
- [ ] Integration tests pass
- [ ] `golangci-lint run` passes

---

## KAN-011: Import Command

### Summary

Implement `kanuka secrets import` command to restore secrets from an export archive.

### Priority

**Lower** - Pairs with export

### Context

After exporting secrets (KAN-010), users need to be able to import them. This is useful for:
- Restoring from backup
- Setting up a new machine
- Migrating projects

### Dependencies

- **KAN-010** (Export Command) - uses same archive format

### New Behavior

```bash
$ kanuka secrets import kanuka-secrets-2024-01-15.tar.gz
⠋ Importing secrets...

Found existing .kanuka directory. How do you want to proceed?
  [m] Merge - Add new files, keep existing
  [r] Replace - Delete existing, use backup
  [c] Cancel

Choice: m

Importing files:
  ✓ .kanuka/config.toml (skipped - exists)
  ✓ .kanuka/public_keys/user1-uuid.pub (skipped - exists)
  ✓ .kanuka/public_keys/user3-uuid.pub (added)
  ✓ .env.kanuka (skipped - exists)
  ✓ config/.env.production.kanuka (added)

Summary:
  2 files added
  3 files skipped (already exist)

# Non-interactive with flags
$ kanuka secrets import backup.tar.gz --merge
$ kanuka secrets import backup.tar.gz --replace

# Dry run
$ kanuka secrets import backup.tar.gz --dry-run
```

### Acceptance Criteria

- [ ] `kanuka secrets import` command implemented
- [ ] Extracts tar.gz archive
- [ ] `--merge` flag adds new files, keeps existing
- [ ] `--replace` flag deletes existing, uses backup
- [ ] `--dry-run` shows what would happen
- [ ] Interactive prompt if neither merge/replace specified
- [ ] Validates archive structure before importing

### Technical Details

#### Files to Create

| File | Purpose |
|------|---------|
| `cmd/secrets_import.go` | Command implementation |

#### Implementation Steps

1. **Open and validate archive:**
   ```go
   file, _ := os.Open(archivePath)
   gzReader, _ := gzip.NewReader(file)
   tarReader := tar.NewReader(gzReader)
   
   // First pass: validate structure
   var files []string
   for {
       header, err := tarReader.Next()
       if err == io.EOF {
           break
       }
       files = append(files, header.Name)
   }
   
   if !isValidKanukaArchive(files) {
       return fmt.Errorf("invalid archive: missing required files")
   }
   ```

2. **Determine import mode:**
   ```go
   if mergeFlag {
       importMode = MergeMode
   } else if replaceFlag {
       importMode = ReplaceMode
   } else if existingKanukaDir() {
       importMode = promptForMode()
   } else {
       importMode = MergeMode // Nothing to merge with
   }
   ```

3. **Extract files:**
   ```go
   // Reset reader for second pass
   file.Seek(0, 0)
   gzReader, _ = gzip.NewReader(file)
   tarReader = tar.NewReader(gzReader)
   
   for {
       header, err := tarReader.Next()
       if err == io.EOF {
           break
       }
       
       targetPath := header.Name
       exists := fileExists(targetPath)
       
       if exists && importMode == MergeMode {
           fmt.Printf("  ✓ %s (skipped - exists)\n", targetPath)
           continue
       }
       
       extractFile(tarReader, header, targetPath)
       fmt.Printf("  ✓ %s (added)\n", targetPath)
   }
   ```

### Testing Requirements

#### Integration Tests

Create `test/integration/import/import_test.go`:

| Test Case | Description |
|-----------|-------------|
| `TestImport_EmptyProject` | Import into project without existing secrets |
| `TestImport_MergeMode` | Merge adds new, keeps existing |
| `TestImport_ReplaceMode` | Replace deletes existing |
| `TestImport_DryRun` | Dry run doesn't modify files |
| `TestImport_InvalidArchive` | Reject malformed archive |
| `TestImport_RoundTrip` | Export then import produces same result |

### Definition of Done

- [ ] `kanuka secrets import` command implemented
- [ ] Merge and replace modes work correctly
- [ ] `--dry-run` flag works
- [ ] Archive validation works
- [ ] Integration tests pass
- [ ] `golangci-lint run` passes

---

## Appendix: Testing Utilities

### Shared Test Helpers

These helpers may be useful across multiple ticket implementations:

```go
// In test/integration/shared/testing_utils.go

// CreateMultiUserProject sets up a project with multiple users for testing.
func CreateMultiUserProject(t *testing.T, userCount int) (*TestProject, []*TestUser) {
    // Implementation
}

// SimulateUserInput provides input to stdin for interactive prompts.
func SimulateUserInput(input string, fn func() error) (string, error) {
    // Similar to CaptureOutputWithStdin but for interactive input
}

// VerifyFileEncryptedWith checks if a file can be decrypted with a given key.
func VerifyFileEncryptedWith(t *testing.T, filePath string, symKey []byte) bool {
    // Returns true if decryption succeeds
}
```

---

## Appendix: Documentation Updates

After implementing all features, update these documentation files:

| File | Updates Needed |
|------|----------------|
| `docs/src/content/docs/guides/` | Add guides for new commands |
| `docs/src/content/docs/reference/` | Command reference |
| `README.md` | Feature list, command overview |
| CLI `--help` text | Already covered by cobra command definitions |

---

## Appendix: Future Considerations

Features explicitly **not** included but worth considering later:

1. **Audit logging** - Track who accessed/modified secrets
2. **Key escrow** - Backup keys for recovery
3. **Hardware key support** - YubiKey, etc.
4. **Secret versioning** - Track history of secret changes
5. **Team management** - Named groups with access policies
