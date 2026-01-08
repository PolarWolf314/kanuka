# Dry-Run Implementation Tickets

This document contains actionable implementation tickets for adding `--dry-run` support to Kanuka CLI commands. Each ticket is self-contained with full context, acceptance criteria, and implementation steps.

---

## Ticket Overview

| Ticket | Command | Priority | Effort | Dependencies |
|--------|---------|----------|--------|--------------|
| [DRY-001](#dry-001-add---dry-run-flag-to-secrets-revoke) | `secrets revoke` | HIGH | 2-3 hours | None |
| [DRY-002](#dry-002-add---dry-run-flag-to-secrets-encrypt) | `secrets encrypt` | MEDIUM | 1-2 hours | None |
| [DRY-003](#dry-003-add---dry-run-flag-to-secrets-decrypt) | `secrets decrypt` | MEDIUM | 2-3 hours | None |
| [DRY-004](#dry-004-add---dry-run-flag-to-secrets-register) | `secrets register` | LOW | 2-3 hours | None |

**Recommended implementation order:** DRY-001 → DRY-002 → DRY-003 → DRY-004

---

## DRY-001: Add `--dry-run` flag to `secrets revoke`

### Summary

Add a `--dry-run` flag to `kanuka secrets revoke` that shows what would be deleted and which users would be affected, without making any changes.

### Priority

**HIGH** - This is the most destructive command in Kanuka. It deletes files, updates config, and rotates the symmetric key for all remaining users.

### Context & Rationale

The `secrets revoke` command:
1. Deletes the target user's `.pub` file from `.kanuka/public_keys/`
2. Deletes the target user's `.kanuka` file from `.kanuka/secrets/`
3. Removes the user from `.kanuka/config.toml`
4. **Rotates the symmetric key** for all remaining users (re-encrypts everyone's `.kanuka` files)

The key rotation step is particularly impactful - it modifies files for users who aren't even involved in the revocation. Users should be able to preview this impact before executing.

The command already has a `--yes` flag to skip confirmation prompts, indicating users want control over execution. A `--dry-run` flag complements this by allowing preview without any confirmation bypass.

### Current Behavior

```bash
$ kanuka secrets revoke --user alice@example.com
✓ Access for alice@example.com has been revoked successfully!
→ Revoked: a1b2c3d4.pub, a1b2c3d4.kanuka
→ Symmetric key has been rotated for remaining users
⚠ Warning: alice@example.com may still have access to old secrets from their local git history.
→ If necessary, rotate your actual secret values after this revocation.
```

### Expected Behavior with `--dry-run`

```bash
$ kanuka secrets revoke --user alice@example.com --dry-run
[dry-run] Would revoke access for alice@example.com

Files that would be deleted:
  - .kanuka/public_keys/a1b2c3d4-5678-90ab-cdef-1234567890ab.pub
  - .kanuka/secrets/a1b2c3d4-5678-90ab-cdef-1234567890ab.kanuka

Config changes:
  - Remove user a1b2c3d4-5678-90ab-cdef-1234567890ab from project
  - Remove device "macbook-pro" from devices

Post-revocation actions:
  - Symmetric key would be rotated for 3 remaining user(s)

⚠ Warning: After revocation, alice@example.com may still have access to old secrets from git history.

No changes made. Run without --dry-run to execute.
```

### Acceptance Criteria

- [x] `--dry-run` flag is available on `secrets revoke` command
- [x] When `--dry-run` is set:
  - [x] No files are deleted
  - [x] No config files are modified
  - [x] No key rotation occurs
  - [x] Output clearly shows files that would be deleted
  - [x] Output shows config changes that would occur
  - [x] Output shows how many users would have keys rotated
  - [x] Output ends with "No changes made" message
- [x] All validation still runs (invalid email, user not found, etc. still produce errors)
- [x] `--dry-run` works with all revocation methods:
  - [x] `--user alice@example.com`
  - [x] `--user alice@example.com --device macbook`
  - [x] `--file .kanuka/secrets/uuid.kanuka`
- [x] `--dry-run` combined with `--yes` still shows preview (doesn't skip output)
- [x] Tests added for dry-run behavior

**Status: COMPLETED** ✓

### Implementation Steps

#### Step 1: Add the flag

In `cmd/secrets_revoke.go`:

```go
var (
    revokeUserEmail string
    revokeFilePath  string
    revokeDevice    string
    revokeYes       bool
    revokeDryRun    bool  // Add this
)

func init() {
    // ... existing flags ...
    revokeCmd.Flags().BoolVar(&revokeDryRun, "dry-run", false, "preview revocation without making changes")
}

func resetRevokeCommandState() {
    // ... existing resets ...
    revokeDryRun = false  // Add this
}
```

#### Step 2: Update the Long description

Add dry-run to the command's Long description:

```go
Long: `Revokes a user's access to the project's encrypted secrets.

... existing description ...

Use --dry-run to preview what would be revoked without making changes.

Examples:
  ... existing examples ...

  # Preview revocation without making changes
  kanuka secrets revoke --user alice@example.com --dry-run`,
```

#### Step 3: Modify `revokeFiles` function

The `revokeFiles` function in `cmd/secrets_revoke.go` needs to check for dry-run mode and print a preview instead of executing:

```go
func revokeFiles(spinner *spinner.Spinner, ctx *revokeContext) error {
    if len(ctx.Files) == 0 {
        return nil
    }

    // If dry-run, print preview and exit early
    if revokeDryRun {
        return printRevokeDryRun(spinner, ctx)
    }

    // ... rest of existing implementation ...
}
```

#### Step 4: Add dry-run output function

Create a new function to handle dry-run output:

```go
func printRevokeDryRun(spinner *spinner.Spinner, ctx *revokeContext) error {
    spinner.Stop()

    fmt.Println(color.YellowString("[dry-run]") + " Would revoke access for " + color.CyanString(ctx.DisplayName))
    fmt.Println()

    // List files that would be deleted
    fmt.Println("Files that would be deleted:")
    for _, file := range ctx.Files {
        fmt.Println("  - " + color.RedString(file.Path))
    }
    fmt.Println()

    // Show config changes
    fmt.Println("Config changes:")
    for _, uuid := range ctx.UUIDsRevoked {
        fmt.Println("  - Remove user " + color.YellowString(uuid) + " from project")
    }
    fmt.Println()

    // Show key rotation impact
    allUsers, err := secrets.GetAllUsersInProject()
    if err == nil && len(allUsers) > len(ctx.UUIDsRevoked) {
        remainingCount := len(allUsers) - len(ctx.UUIDsRevoked)
        fmt.Println("Post-revocation actions:")
        fmt.Printf("  - Symmetric key would be rotated for %d remaining user(s)\n", remainingCount)
        fmt.Println()
    }

    // Warning about git history
    fmt.Println(color.YellowString("⚠") + " Warning: After revocation, " + ctx.DisplayName + " may still have access to old secrets from git history.")
    fmt.Println()

    fmt.Println(color.CyanString("No changes made.") + " Run without --dry-run to execute.")

    spinner.FinalMSG = ""  // Clear spinner message since we printed our own output
    return nil
}
```

#### Step 5: Add tests

Create tests in `test/integration/revoke/revoke_dry_run_test.go`:

```go
func TestRevokeCommand_DryRun(t *testing.T) {
    // Test: --dry-run does not delete files
    // Test: --dry-run does not modify config
    // Test: --dry-run shows correct file list
    // Test: --dry-run works with --user flag
    // Test: --dry-run works with --device flag
    // Test: --dry-run works with --file flag
    // Test: --dry-run with invalid user still shows error
    // Test: --dry-run combined with --yes still shows preview
}
```

### Files to Modify

- `cmd/secrets_revoke.go` - Add flag, modify logic, add dry-run output
- `test/integration/revoke/` - Add dry-run tests

### Testing Checklist

```bash
# Manual testing commands
kanuka secrets revoke --user test@example.com --dry-run
kanuka secrets revoke --user test@example.com --device laptop --dry-run
kanuka secrets revoke --file .kanuka/secrets/uuid.kanuka --dry-run
kanuka secrets revoke --user nonexistent@example.com --dry-run  # Should error
kanuka secrets revoke --user test@example.com --dry-run --yes  # Should still show preview

# Verify no files changed
git status  # Should show no changes after dry-run

# Run automated tests
go test -v ./test/integration/revoke/...
```

---

## DRY-002: Add `--dry-run` flag to `secrets encrypt`

### Summary

Add a `--dry-run` flag to `kanuka secrets encrypt` that shows which `.env` files would be encrypted without actually creating any `.kanuka` files.

### Priority

**MEDIUM** - Useful for previewing file discovery, especially in new projects or CI/CD pipelines.

### Context & Rationale

The `secrets encrypt` command:
1. Discovers all `.env` files in the project (recursively, excluding `.kanuka/` directory)
2. Encrypts each file using the project's symmetric key
3. Creates corresponding `.env.kanuka` files

Users may want to preview which files will be encrypted before committing, especially:
- When running encrypt for the first time in a project
- To verify file discovery is finding the expected files
- In CI/CD pipelines for validation without side effects

### Current Behavior

```bash
$ kanuka secrets encrypt
✓ Environment files encrypted successfully!
The following files were created: .env.kanuka, src/config/.env.local.kanuka
→ You can now safely commit all .kanuka files to version control
```

### Expected Behavior with `--dry-run`

```bash
$ kanuka secrets encrypt --dry-run
[dry-run] Would encrypt 3 environment file(s)

Files that would be created:
  .env                    → .env.kanuka
  src/config/.env.local   → src/config/.env.local.kanuka
  tests/.env.test         → tests/.env.test.kanuka

No changes made. Run without --dry-run to execute.
```

### Acceptance Criteria

- [ ] `--dry-run` flag is available on `secrets encrypt` command
- [ ] When `--dry-run` is set:
  - [ ] No `.kanuka` files are created or modified
  - [ ] Output shows source → destination file mapping
  - [ ] Output shows total count of files that would be encrypted
  - [ ] Output ends with "No changes made" message
- [ ] All validation still runs (no access, no project init, etc. still produce errors)
- [ ] Symmetric key decryption is still validated (user must have access)
- [ ] Tests added for dry-run behavior

### Implementation Steps

#### Step 1: Add the flag

In `cmd/secrets_encrypt.go`:

```go
var (
    encryptDryRun bool
)

func init() {
    encryptCmd.Flags().BoolVar(&encryptDryRun, "dry-run", false, "preview encryption without making changes")
}

// Add reset function for testing
func resetEncryptCommandState() {
    encryptDryRun = false
}
```

#### Step 2: Update the command description

Add dry-run example to the command.

#### Step 3: Add dry-run check before encryption

In the `RunE` function, after file discovery and key validation but before `secrets.EncryptFiles()`:

```go
// After symKey is obtained and before EncryptFiles()

if encryptDryRun {
    spinner.Stop()
    printEncryptDryRun(listOfEnvFiles)
    return nil
}

Logger.Infof("Encrypting %d files", len(listOfEnvFiles))
if err := secrets.EncryptFiles(symKey, listOfEnvFiles, verbose); err != nil {
    // ... existing error handling ...
}
```

#### Step 4: Add dry-run output function

```go
func printEncryptDryRun(envFiles []string) {
    fmt.Println(color.YellowString("[dry-run]") + fmt.Sprintf(" Would encrypt %d environment file(s)", len(envFiles)))
    fmt.Println()

    fmt.Println("Files that would be created:")
    for _, envFile := range envFiles {
        kanukaFile := envFile + ".kanuka"
        fmt.Printf("  %s → %s\n", color.CyanString(envFile), color.GreenString(kanukaFile))
    }
    fmt.Println()

    fmt.Println(color.CyanString("No changes made.") + " Run without --dry-run to execute.")
}
```

#### Step 5: Add tests

Create tests in `test/integration/encrypt/encrypt_dry_run_test.go`.

### Files to Modify

- `cmd/secrets_encrypt.go` - Add flag and dry-run logic
- `test/integration/encrypt/` - Add dry-run tests

### Testing Checklist

```bash
# Manual testing
kanuka secrets encrypt --dry-run

# Verify no files created
git status  # Should show no new .kanuka files

# Run automated tests
go test -v ./test/integration/encrypt/...
```

---

## DRY-003: Add `--dry-run` flag to `secrets decrypt`

### Summary

Add a `--dry-run` flag to `kanuka secrets decrypt` that shows which `.kanuka` files would be decrypted and warns about any existing `.env` files that would be overwritten.

### Priority

**MEDIUM** - More complex than encrypt because it adds overwrite detection, which provides additional safety value.

### Context & Rationale

The `secrets decrypt` command:
1. Discovers all `.kanuka` files in the project
2. Decrypts each file using the user's private key
3. Creates corresponding `.env` files (overwriting if they exist)

The key concern here is **overwriting existing `.env` files**. If a user has made local modifications to their `.env` file, running decrypt will silently overwrite those changes. A dry-run with overwrite detection helps users avoid accidental data loss.

### Current Behavior

```bash
$ kanuka secrets decrypt
✓ Environment files decrypted successfully!
The following files were created: .env, src/config/.env.local
→ Your environment files are now ready to use
```

### Expected Behavior with `--dry-run`

```bash
$ kanuka secrets decrypt --dry-run
[dry-run] Would decrypt 3 encrypted file(s)

Files that would be created:
  .env.kanuka                    → .env (exists - would be overwritten)
  src/config/.env.local.kanuka   → src/config/.env.local (new file)
  tests/.env.test.kanuka         → tests/.env.test (new file)

⚠ Warning: 1 existing file would be overwritten.

No changes made. Run without --dry-run to execute.
```

### Acceptance Criteria

- [ ] `--dry-run` flag is available on `secrets decrypt` command
- [ ] When `--dry-run` is set:
  - [ ] No `.env` files are created or modified
  - [ ] Output shows source → destination file mapping
  - [ ] Output indicates which destination files already exist
  - [ ] Output shows warning if any files would be overwritten
  - [ ] Output shows total count of files
  - [ ] Output ends with "No changes made" message
- [ ] All validation still runs (no access, etc. still produce errors)
- [ ] Symmetric key decryption is still validated
- [ ] Tests added for dry-run behavior

### Implementation Steps

#### Step 1: Add the flag

In `cmd/secrets_decrypt.go`:

```go
var (
    decryptDryRun bool
)

func init() {
    decryptCmd.Flags().BoolVar(&decryptDryRun, "dry-run", false, "preview decryption without making changes")
}

func resetDecryptCommandState() {
    decryptDryRun = false
}
```

#### Step 2: Add dry-run check and overwrite detection

```go
// After symKey is obtained and before DecryptFiles()

if decryptDryRun {
    spinner.Stop()
    printDecryptDryRun(listOfKanukaFiles)
    return nil
}
```

#### Step 3: Add dry-run output function with overwrite detection

```go
func printDecryptDryRun(kanukaFiles []string) {
    fmt.Println(color.YellowString("[dry-run]") + fmt.Sprintf(" Would decrypt %d encrypted file(s)", len(kanukaFiles)))
    fmt.Println()

    fmt.Println("Files that would be created:")
    
    overwriteCount := 0
    for _, kanukaFile := range kanukaFiles {
        // Remove .kanuka extension to get target .env file
        envFile := strings.TrimSuffix(kanukaFile, ".kanuka")
        
        // Check if target file exists
        status := color.GreenString("new file")
        if _, err := os.Stat(envFile); err == nil {
            status = color.YellowString("exists - would be overwritten")
            overwriteCount++
        }
        
        fmt.Printf("  %s → %s (%s)\n", color.CyanString(kanukaFile), envFile, status)
    }
    fmt.Println()

    if overwriteCount > 0 {
        fmt.Printf(color.YellowString("⚠")+" Warning: %d existing file(s) would be overwritten.\n", overwriteCount)
        fmt.Println()
    }

    fmt.Println(color.CyanString("No changes made.") + " Run without --dry-run to execute.")
}
```

#### Step 4: Add tests

Create tests covering:
- Dry-run with no existing files
- Dry-run with some existing files (overwrite warning)
- Dry-run with all existing files

### Files to Modify

- `cmd/secrets_decrypt.go` - Add flag and dry-run logic
- `test/integration/decrypt/` - Add dry-run tests

### Testing Checklist

```bash
# Manual testing - with no existing .env files
rm -f .env
kanuka secrets decrypt --dry-run

# Manual testing - with existing .env file
touch .env
kanuka secrets decrypt --dry-run  # Should show overwrite warning

# Verify no files changed
git status

# Run automated tests
go test -v ./test/integration/decrypt/...
```

---

## DRY-004: Add `--dry-run` flag to `secrets register`

### Summary

Add a `--dry-run` flag to `kanuka secrets register` that shows what files would be created when registering a user.

### Priority

**LOW** - This is a non-destructive operation (only creates files) and the current output already clearly shows what was created. Implement only if users request this feature.

### Context & Rationale

The `secrets register` command:
1. Loads the target user's public key
2. Decrypts the symmetric key using the current user's private key
3. Encrypts the symmetric key with the target user's public key
4. Saves the encrypted key as a `.kanuka` file for the target user

This is a non-destructive operation - it only creates new files. The current output already clearly shows what was created. However, for consistency with other commands and CI/CD validation, a dry-run option could be useful.

### Current Behavior

```bash
$ kanuka secrets register --user alice@example.com
✓ alice@example.com has been granted access successfully!

Files created:
  Public key:    .kanuka/public_keys/a1b2c3d4.pub
  Encrypted key: .kanuka/secrets/a1b2c3d4.kanuka

→ They now have access to decrypt the repository's secrets
```

### Expected Behavior with `--dry-run`

```bash
$ kanuka secrets register --user alice@example.com --dry-run
[dry-run] Would register alice@example.com

Files that would be created:
  - .kanuka/secrets/a1b2c3d4-5678-90ab-cdef-1234567890ab.kanuka

Prerequisites verified:
  ✓ User exists in project config
  ✓ Public key found at .kanuka/public_keys/a1b2c3d4.pub
  ✓ Current user has access to decrypt symmetric key

No changes made. Run without --dry-run to execute.
```

### Acceptance Criteria

- [ ] `--dry-run` flag is available on `secrets register` command
- [ ] When `--dry-run` is set:
  - [ ] No `.kanuka` files are created
  - [ ] No public keys are copied (for `--file` or `--pubkey` modes)
  - [ ] Output shows files that would be created
  - [ ] Output confirms prerequisites were verified
  - [ ] Output ends with "No changes made" message
- [ ] All validation still runs
- [ ] Works with all registration methods:
  - [ ] `--user email@example.com`
  - [ ] `--file path/to/key.pub`
  - [ ] `--pubkey "ssh-rsa ..." --user email@example.com`
- [ ] Tests added for dry-run behavior

### Implementation Steps

#### Step 1: Add the flag

In `cmd/secrets_register.go`:

```go
var (
    registerUserEmail string
    customFilePath    string
    publicKeyText     string
    registerDryRun    bool  // Add this
)

func init() {
    // ... existing flags ...
    RegisterCmd.Flags().BoolVar(&registerDryRun, "dry-run", false, "preview registration without making changes")
}

func resetRegisterCommandState() {
    // ... existing resets ...
    registerDryRun = false
}
```

#### Step 2: Modify each registration handler

Each of the three registration paths needs to be updated:
- `handleUserRegistration()`
- `handleCustomFileRegistration()`
- `handlePubkeyTextRegistration()`

For each, add a dry-run check after validation but before file creation.

#### Step 3: Add dry-run output function

```go
func printRegisterDryRun(spinner *spinner.Spinner, displayName, targetUUID, pubKeyPath, kanukaPath string) {
    spinner.Stop()

    fmt.Println(color.YellowString("[dry-run]") + " Would register " + color.CyanString(displayName))
    fmt.Println()

    fmt.Println("Files that would be created:")
    fmt.Println("  - " + color.GreenString(kanukaPath))
    fmt.Println()

    fmt.Println("Prerequisites verified:")
    fmt.Println("  " + color.GreenString("✓") + " User exists in project config")
    fmt.Println("  " + color.GreenString("✓") + " Public key found at " + pubKeyPath)
    fmt.Println("  " + color.GreenString("✓") + " Current user has access to decrypt symmetric key")
    fmt.Println()

    fmt.Println(color.CyanString("No changes made.") + " Run without --dry-run to execute.")
}
```

#### Step 4: Add tests

Create tests in `test/integration/register/register_dry_run_test.go`.

### Files to Modify

- `cmd/secrets_register.go` - Add flag and dry-run logic to all three handlers
- `test/integration/register/` - Add dry-run tests

### Testing Checklist

```bash
# Manual testing
kanuka secrets register --user test@example.com --dry-run
kanuka secrets register --file ./key.pub --dry-run
kanuka secrets register --pubkey "ssh-rsa ..." --user test@example.com --dry-run

# Verify no files created
git status

# Run automated tests
go test -v ./test/integration/register/...
```

---

## Appendix: Shared Patterns

### Output Format Standard

All dry-run implementations should use consistent formatting:

```go
// Header
fmt.Println(color.YellowString("[dry-run]") + " Would <action> <target>")
fmt.Println()

// File changes section
fmt.Println("Files that would be <created/deleted/modified>:")
fmt.Println("  - " + color.GreenString(path))  // Created
fmt.Println("  - " + color.RedString(path))    // Deleted
fmt.Println("  - " + color.YellowString(path)) // Modified
fmt.Println()

// Warnings (if applicable)
fmt.Println(color.YellowString("⚠") + " Warning: <warning message>")
fmt.Println()

// Footer
fmt.Println(color.CyanString("No changes made.") + " Run without --dry-run to execute.")
```

### Testing Pattern

Each dry-run test should verify:

```go
func TestCommand_DryRun(t *testing.T) {
    // Setup: Create necessary files/state
    
    // Execute: Run command with --dry-run
    
    // Assert: No files were created/modified/deleted
    // Assert: Output contains expected preview information
    // Assert: Command exits successfully
}

func TestCommand_DryRun_InvalidInput(t *testing.T) {
    // Setup: Invalid state
    
    // Execute: Run command with --dry-run
    
    // Assert: Appropriate error is shown
    // Assert: Validation still runs even in dry-run mode
}
```

### Flag Reset Pattern

All commands with dry-run flags need reset functions for testing:

```go
func resetCommandState() {
    // Reset all package-level flag variables
    dryRun = false
    // ... other flags ...
}
```

These reset functions are called in test setup to ensure clean state between tests.
