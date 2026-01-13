# Kanuka Acceptance Test Findings

This document summarizes all errors and issues found during manual acceptance testing of the Kanuka CLI tool.

---

## Executive Summary

During manual acceptance testing of Kanuka v1.2.1, 18 distinct errors were identified across multiple test scenarios. These range from UI/UX issues to serious functional bugs that affect data integrity and user experience.

**Error Severity Breakdown:**
- **Critical:** 5 (data integrity, security, file system state issues)
- **High:** 4 (functional bugs affecting core workflows)
- **Medium:** 6 (UX issues, error handling)
- **Low:** 3 (cosmetic issues, minor UX problems)

---

## Critical Issues

### ERR-001: Encrypt/Decrypt/Access/Status Hang When Not in Project

**Test Case:** TEST-002
**Command(s):**
```bash
kanuka secrets encrypt
kanuka secrets decrypt
kanuka secrets access
kanuka secrets status
```

**Issue:**
When running secrets commands (encrypt, decrypt, access, status) outside of a Kanuka-initialized project, the commands hang indefinitely or show incorrect behavior:
- `encrypt` causes spinner to hang infinitely
- `decrypt` causes spinner to hang infinitely
- `access` says "no user found" (correct) but displays "test-project" (incorrect)
- `status` hangs with no spinner

**Root Cause Analysis:**
Looking at the code in `cmd/secrets_encrypt.go:74-86` and `cmd/secrets_decrypt.go:74-87`:

```go
if projectPath == "" {
    finalMessage := ui.Error.Sprint("✗") + " Kānuka has not been initialized\n" +
        ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " instead"
    spinner.FinalMSG = finalMessage
    return nil
}
```

The code correctly detects the missing project but sets `spinner.FinalMSG` and returns `nil`. The spinner cleanup happens via `defer cleanup()` which calls `spinner.Stop()`. However, the spinner's `FinalMSG` is set before the cleanup runs, and in some cases the spinner may not display the final message properly, leading to a hang.

For `access` (cmd/secrets_access.go:82-90):
```go
if projectPath == "" {
    if accessJSONOutput {
        fmt.Println(`{"error": "Kanuka has not been initialized"}`)
        return nil
    }
    fmt.Println(ui.Error.Sprint("✗") + " Kanuka has not been initialized")
    fmt.Println(ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first")
    return nil
}
```

The "test-project" display appears to be from the test environment. Looking at grep results, "test-project" is used extensively in test files. This suggests the `access` command might be displaying fallback test data when no project exists.

For `status` (cmd/secrets_status.go:87-95):
```go
if projectPath == "" {
    if statusJSONOutput {
        fmt.Println(`{"error": "Kanuka has not been initialized"}`)
        return nil
    }
    fmt.Println(ui.Error.Sprint("✗") + " Kanuka has not been initialized")
    fmt.Println(ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first")
    return nil
}
```

The issue with `status` hanging may be related to how the spinner is managed when no project exists.

**Why This Is an Error:**
1. **Poor UX:** Users expect clear error messages, not indefinite hangs
2. **Misleading output:** "test-project" being displayed when no project exists is confusing
3. **Inconsistent behavior:** Different commands behave differently in the same error condition

**Reproduction Steps:**
1. Navigate to a directory without a `.kanuka` folder
2. Run any of: `kanuka secrets encrypt`, `kanuka secrets decrypt`, `kanuka secrets status`
3. Observe the spinner hanging or incorrect output

**Expected Behavior:**
All commands should immediately display:
```
✗ Kānuka has not been initialized
→ Run 'kanuka secrets init' first
```
And exit with a non-zero status code.

---

### ERR-002: Create Command Fails Without Clear Error When Not in Project

**Test Case:** TEST-003
**Command:**
```bash
kanuka secrets create
```

**Issue:**
When running `kanuka secrets create` outside of a Kanuka project directory, the command fails with a cryptic Go error instead of a user-friendly error message.

**Root Cause Analysis:**
Looking at `cmd/secrets_create.go:78-95`:

```go
Logger.Debugf("Initializing project settings")
if err := configs.InitProjectSettings(); err != nil {
    return Logger.ErrorfAndReturn("failed to init project settings: %v", err)
}
projectPath := configs.ProjectKanukaSettings.ProjectPath
Logger.Debugf("Project path: %s", projectPath)

if projectPath == "" {
    finalMessage := ui.Error.Sprint("✗") + " Kānuka has not been initialized\n" +
        ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " instead"
    spinner.FinalMSG = finalMessage
    return nil
}
```

The code checks if `projectPath == ""` after calling `configs.InitProjectSettings()`. Looking at `internal/configs/settings.go:67-110`:

```go
func InitProjectSettings() error {
    projectName, err := utils.GetProjectName()
    if err != nil {
        return fmt.Errorf("error getting project name: %w", err)
    }
    projectPath, err := utils.FindProjectKanukaRoot()
    if err != nil {
        return fmt.Errorf("error getting project root: %w", err)
    }
    // ... checks for legacy project, migrates, etc.
    ProjectKanukaSettings = &ProjectSettings{
        ProjectName:          projectName,
        ProjectPath:          projectPath,
        ProjectPublicKeyPath: filepath.Join(projectPath, ".kanuka", "public_keys"),
        ProjectSecretsPath:   filepath.Join(projectPath, ".kanuka", "secrets"),
    }
    // ...
}
```

When there's no `.kanuka` directory, `utils.FindProjectKanukaRoot()` returns empty string. The `InitProjectSettings()` function returns `nil` (success) even when no project exists, and just sets `ProjectPath` to empty string. The subsequent check `if projectPath == ""` should catch this.

However, the error in the test output shows:
```
Error: Failed to copy public key to project: failed to write key to project: open /Users/aaron/.kanuka/public_keys/ab26c005-c609-4e34-9277-6fd811700ad9.pub: no such file or directory
```

This suggests the keypair was created but the copy failed because the `.kanuka` directory doesn't exist. Looking at `cmd/secrets_create.go:202-217`:

```go
Logger.Debugf("Creating and saving RSA key pair")
if err := secrets.CreateAndSaveRSAKeyPair(verbose); err != nil {
    return Logger.ErrorfAndReturn("Failed to generate and save RSA key pair: %v", err)
}
Logger.Infof("RSA key pair created successfully")

Logger.Debugf("Copying user public key to project")
destPath, err := secrets.CopyUserPublicKeyToProject()
_ = destPath // explicitly ignore destPath for now
if err != nil {
    return Logger.ErrorfAndReturn("Failed to copy public key to project: %v", err)
}
```

The `CreateAndSaveRSAKeyPair` function creates keys in the user's local storage (`~/.local/share/kanuka/keys/`), but the subsequent `CopyUserPublicKeyToProject()` tries to copy to the project's `.kanuka/public_keys/` directory which doesn't exist.

**Why This Is an Error:**
1. **Confusing error message:** The error is about a missing directory (`/Users/aaron/.kanuka/public_keys/`) which doesn't explain the root cause
2. **Poor UX:** The command generates keypair before verifying the project exists
3. **Inconsistent:** Other commands check for project existence before performing operations

**Reproduction Steps:**
1. Navigate to a directory without a `.kanuka` folder
2. Run `kanuka secrets create` (with or without verbose flag)
3. Observe the error message about missing directory

**Expected Behavior:**
```
✗ Kānuka has not been initialized
→ Run 'kanuka secrets init' first to create a project
```

---

### ERR-003: .kanuka Folder Created Too Early During Init

**Test Case:** TEST-030
**Command:**
```bash
kanuka secrets init
# (then cancel during prompt)
kanuka secrets init
```

**Issue:**
If user cancels the init prompt (e.g., with Ctrl+C or by not providing input), the `.kanuka` folder is already created. Subsequent init attempts fail with "already initialized" error, even though the init was incomplete.

**Root Cause Analysis:**
Looking at `cmd/secrets_init.go:100-104`:

```go
Logger.Debugf("Ensuring kanuka settings and creating .kanuka folders")
if err := secrets.EnsureKanukaSettings(); err != nil {
    return Logger.ErrorfAndReturn("Failed to create .kanuka folders: %v", err)
}
Logger.Infof("Kanuka settings and folders created successfully")
```

The `.kanuka` folder is created early in the init process, before the interactive prompts complete. Later at line 44-54:

```go
Logger.Debugf("Checking if project kanuka settings already exist")
kanukaExists, err := secrets.DoesProjectKanukaSettingsExist()
if err != nil {
    return Logger.ErrorfAndReturn("Failed to check if project kanuka settings exists: %v", err)
}
if kanukaExists {
    finalMessage := ui.Error.Sprint("✗") + " Kānuka has already been initialized\n" +
        ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets create") + " instead"
    spinner.FinalMSG = finalMessage
    return nil
}
```

This check happens, but since `.kanuka` folder was already created, `DoesProjectKanukaSettingsExist()` returns `true`, blocking re-init.

**Why This Is an Error:**
1. **Data inconsistency:** Partial `.kanuka` directory structure left behind
2. **Poor recovery:** User must manually delete `.kanuka` folder to retry init
3. **Bad UX:** Canceling a prompt should not leave persistent state changes

**Reproduction Steps:**
1. Navigate to an empty directory
2. Run `kanuka secrets init`
3. When prompted for project name, cancel with Ctrl+C
4. Run `kanuka secrets init` again
5. Observe "already initialized" error

**Expected Behavior:**
If init is cancelled:
1. Any created files should be cleaned up
2. Subsequent init should succeed
3. Error message should explain the situation (e.g., "Incomplete init detected. Run `rm -rf .kanuka` to clean up and try again")

---

### ERR-004: Glob Pattern Matching Encrypts All Files Instead of Matching Pattern

**Test Case:** TEST-042
**Command:**
```bash
kanuka secrets encrypt "services/*/.env"
```

**Issue:**
When providing a specific glob pattern like `"services/*/.env"`, the command encrypts ALL `.env` files in the project instead of only matching the glob pattern.

**Root Cause Analysis:**
Looking at `cmd/secrets_encrypt.go:92-111`:

```go
var listOfEnvFiles []string
if len(args) > 0 {
    // Use user-provided file patterns.
    Logger.Debugf("User provided %d file pattern(s)", len(args))
    resolved, err := secrets.ResolveFiles(args, projectPath, true)
    if err != nil {
        Logger.Errorf("Failed to resolve file patterns: %v", err)
        finalMessage := ui.Error.Sprint("✗") + " " + err.Error()
        spinner.FinalMSG = finalMessage
        return nil
    }
    listOfEnvFiles = resolved
} else {
    // Default: find all .env files.
    Logger.Debugf("Searching for .env files in project path")
    found, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, false)
    if err != nil {
        return Logger.ErrorfAndReturn("Failed to find environment files: %v", err)
    }
    listOfEnvFiles = found
}
```

The code calls `secrets.ResolveFiles(args, projectPath, true)` when arguments are provided. Looking at `internal/secrets/files.go:12-43`:

```go
func resolvePattern(pattern string, projectPath string, forEncryption bool) ([]string, error) {
    // Convert relative patterns to absolute paths based on project path.
    absPattern := pattern
    if !filepath.IsAbs(pattern) {
        absPattern = filepath.Join(projectPath, pattern)
    }

    // Check if it's a directory.
    info, err := os.Stat(absPattern)
    if err == nil && info.IsDir() {
        return findFilesInDir(absPattern, forEncryption)
    }

    // Check if it contains glob characters.
    if strings.ContainsAny(pattern, "*?[") {
        return expandGlob(pattern, projectPath, forEncryption)
    }
    // ...
}
```

And the glob expansion in `internal/secrets/files.go:79-114`:

```go
func expandGlob(pattern string, projectPath string, forEncryption bool) ([]string, error) {
    // Use doublestar for ** support.
    // We need to use fsys version with os.DirFS for proper ** handling.
    absPattern := pattern
    if !filepath.IsAbs(pattern) {
        absPattern = filepath.Join(projectPath, pattern)
    }

    matches, err := doublestar.FilepathGlob(absPattern)
    if err != nil {
        return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
    }

    // Filter to only include appropriate file types.
    var filtered []string
    for _, m := range matches {
        // Skip directories.
        info, err := os.Stat(m)
        if err != nil || info.IsDir() {
            continue
        }

        // Skip files inside .kanuka directory.
        if isInKanukaDir(m) {
            continue
        }

        if forEncryption && isEnvFile(m) {
            filtered = append(filtered, m)
        } else if !forEncryption && isKanukaFile(m) {
            filtered = append(filtered, m)
        }
    }

    return filtered, nil
}
```

The issue may be related to how the pattern is resolved or how the glob library matches files. If `"services/*/.env"` is being joined with the project path and then passed to glob, the glob might be matching more files than expected.

However, looking more closely at the test output in ACCEPTANCE_TESTING.md:
```bash
kanuka secrets encrypt "services/*/.env"
✓ Environment files encrypted successfully!
The following files were created:
    - /Users/aaron/Developer/testing/acceptance_testing/.env.kanuka
    - /Users/aaron/Developer/testing/acceptance_testing/.env.local.kanuka
    - /Users/aaron/Developer/testing/acceptance_testing/config/.env.production.kanuka
    - /Users/aaron/Developer/testing/acceptance_testing/services/api/.env.kanuka
    - /Users/aaron/Developer/testing/acceptance_testing/services/web/.env.kanuka
```

It shows ALL files being encrypted, not just those in `services/*`. This suggests that either:
1. The glob pattern is being ignored entirely
2. The glob is matching too broadly
3. There's a bug in how the pattern is being processed

**Why This Is an Error:**
1. **Functionality broken:** Cannot selectively encrypt files as documented
2. **Unexpected behavior:** Encrypts more files than user requested
3. **Security concern:** User may inadvertently encrypt files they didn't intend to

**Reproduction Steps:**
1. Set up a project with multiple `.env` files in different directories
2. Run `kanuka secrets encrypt "services/*/.env"` (with quotes to prevent shell expansion)
3. Observe that ALL `.env` files are encrypted, not just those in `services/`

**Expected Behavior:**
Only `.env` files matching the pattern `services/*/.env` should be encrypted:
```
✓ Environment files encrypted successfully!
The following files were created:
    - /path/to/project/services/api/.env.kanuka
    - /path/to/project/services/web/.env.kanuka
```

---

### ERR-005: Decrypt Command Ignores File Path Arguments

**Test Case:** TEST-046
**Command:**
```bash
kanuka secrets decrypt .env.kanuka
```

**Issue:**
When providing a specific file path to decrypt, the command ignores the argument and decrypts ALL `.kanuka` files in the project.

**Root Cause Analysis:**
Looking at `cmd/secrets_decrypt.go:92-111`:

```go
var listOfKanukaFiles []string
if len(args) > 0 {
    // Use user-provided file patterns.
    Logger.Debugf("User provided %d file pattern(s)", len(args))
    resolved, err := secrets.ResolveFiles(args, projectPath, false)
    if err != nil {
        Logger.Errorf("Failed to resolve file patterns: %v", err)
        finalMessage := ui.Error.Sprint("✗") + " " + err.Error()
        spinner.FinalMSG = finalMessage
        return nil
    }
    listOfKanukaFiles = resolved
} else {
    // Default: find all .kanuka files.
    Logger.Debugf("Searching for .kanuka files in project path")
    found, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, true)
    if err != nil {
        return Logger.ErrorfAndReturn("Failed to find environment files: %v", err)
    }
    listOfKanukaFiles = found
}
```

The code structure is the same as for encrypt. When `len(args) > 0`, it should use `ResolveFiles()`. However, there may be an issue with how `ResolveFiles()` handles specific file paths vs patterns.

Looking at `internal/secrets/files.go:45-77` (the `resolvePattern` function):

```go
// Check if it's a directory.
info, err := os.Stat(absPattern)
if err == nil && info.IsDir() {
    return findFilesInDir(absPattern, forEncryption)
}

// Check if it contains glob characters.
if strings.ContainsAny(pattern, "*?[") {
    return expandGlob(pattern, projectPath, forEncryption)
}

// Treat as literal file path.
if _, err := os.Stat(absPattern); os.IsNotExist(err) {
    return nil, fmt.Errorf("file not found: %s", pattern)
}

// Validate that file matches the expected type.
if forEncryption && !isEnvFile(absPattern) {
    return nil, fmt.Errorf("file is not a .env file: %s", pattern)
}
if !forEncryption && !isKanukaFile(absPattern) {
    return nil, fmt.Errorf("file is not a .kanuka file: %s", pattern)
}

return []string{absPattern}, nil
```

Wait - this code checks if `forEncryption` and if it's not a `.env` file, it returns an error. But for decrypt, `forEncryption` is `false`. So it should check if it's a `.kanuka` file.

Actually, looking at the test output again:
```bash
kanuka secrets decrypt .env.kanuka
Warning: Decrypted .env files contain sensitive data - ensure they're in your .gitignore
✓ Environment files decrypted successfully!
The following files were created:
    - /Users/aaron/Developer/testing/acceptance_testing/.env
    - /Users/aaron/Developer/testing/acceptance_testing/.env.local
    - /Users/aaron/Developer/testing/acceptance_testing/config/.env.production
    - /Users/aaron/Developer/testing/acceptance_testing/services/api/.env
    - /Users/aaron/Developer/testing/acceptance_testing/services/web/.env
```

All files are being decrypted, not just `.env.kanuka`. This is the same bug pattern as ERR-004 - the glob pattern/file argument is being ignored.

**Why This Is an Error:**
1. **Command doesn't work as documented:** The help text shows specific file decryption is supported
2. **Inconsistent with encrypt:** Same issue affects both commands
3. **Poor UX:** User expects only specific files to be decrypted

**Reproduction Steps:**
1. Have multiple `.kanuka` files in project
2. Run `kanuka secrets decrypt .env.kanuka`
3. Observe that ALL `.kanuka` files are decrypted

**Expected Behavior:**
Only `.env.kanuka` should be decrypted to `.env`:
```
✓ Environment files decrypted successfully!
The following files were created:
    - /path/to/project/.env
→ Your environment files are now ready to use
```

---

### ERR-006: Register Shows Public Key as "Created" When It Already Existed

**Test Case:** TEST-064
**Command:**
```bash
kanuka secrets register --user newuser@example.com
```

**Issue:**
When registering a user who already has a public key, the success message lists the public key path under "Files created", even though it already existed.

**Root Cause Analysis:**
Looking at `cmd/secrets_register.go:527-564` (handleUserRegistration function):

```go
// Compute path for output
targetKanukaFilePath := filepath.Join(projectSecretsPath, targetUserUUID+".kanuka")

// Check if user already has access (both public key AND .kanuka file exist)
userAlreadyHasAccess := fileExists(targetPubkeyPath) && fileExists(targetKanukaFilePath)
Logger.Debugf("User already has access: %t (pubkey: %s, kanuka: %s)", userAlreadyHasAccess, targetPubkeyPath, targetKanukaFilePath)

// If user already has access and not forced, prompt for confirmation
if userAlreadyHasAccess && !registerForce && !registerDryRun {
    if !confirmRegisterOverwrite(spinner, registerUserEmail) {
        spinner.FinalMSG = ui.Warning.Sprint("⚠") + " Registration cancelled.\n"
        return nil
    }
}
// ...
finalMessage := ui.Success.Sprint("✓") + " " + ui.Highlight.Sprint(registerUserEmail) + " " + successVerb + " successfully!\n\n" +
    filesLabel + ":\n" +
    "  Public key:    " + ui.Path.Sprint(targetPubkeyPath) + "\n" +
    "  Encrypted key: " + ui.Path.Sprint(targetKanukaFilePath) + "\n\n" +
    ui.Info.Sprint("→") + " They now have access to decrypt the repository's secrets"
spinner.FinalMSG = finalMessage
return nil
```

The code uses the same message format regardless of whether files were created or updated. When `userAlreadyHasAccess` is true (the public key already existed), the message should say "Files updated" (which it does at line 577), but still lists the public key path. This is misleading because the public key wasn't "created" - it already existed.

The logic at lines 347-354:
```go
// Use different message for update vs new registration
var successVerb, filesLabel string
if userAlreadyHasAccess {
    successVerb = "access has been updated"
    filesLabel = "Files updated"
} else {
    successVerb = "has been granted access"
    filesLabel = "Files created"
}
```

Correctly changes the label to "Files updated", but the file list still shows both files as if they were created.

**Why This Is an Error:**
1. **Misleading message:** "Files updated" would be more appropriate for the `.kanuka` file, but the public key wasn't touched
2. **User confusion:** User might think both files were created when only one was
3. **Inaccuracy:** Success message doesn't accurately reflect what happened

**Reproduction Steps:**
1. User has already run `kanuka secrets create` (public key exists)
2. Admin runs `kanuka secrets register --user newuser@example.com`
3. Observe the success message showing both public key and encrypted key under "Files created" or "Files updated"

**Expected Behavior:**
```
✓ newuser@example.com has been granted access successfully!

Files created:
  Encrypted key: /path/to/project/.kanuka/secrets/<uuid>.kanuka
```

(The public key should not be listed as it already existed)

---

### ERR-007: Register with --file Has Multiple Issues

**Test Case:** TEST-067
**Command:**
```bash
kanuka secrets register --file /path/to/pubkey.pub
```

**Issue:**
Using `--file` to register a user from a public key file has several problems:
1. The encrypted key file is named after the filename base instead of using a UUID
2. No public key is written to the project's `public_keys/` directory
3. Breaks project config because it creates a user entry with no UUID or email

**Root Cause Analysis:**
Looking at `cmd/secrets_register.go:592-751` (handleCustomFileRegistration function):

```go
// The target user UUID is the filename without .pub extension
targetUserUUID := strings.TrimSuffix(filepath.Base(customFilePath), ".pub")

// Try to find email for display purposes
targetEmail := projectConfig.Users[targetUserUUID]
displayName := targetEmail
if displayName == "" {
    displayName = targetUserUUID
}

// Compute path for output
targetKanukaFilePath := filepath.Join(projectSecretsPath, targetUserUUID+".kanuka")
// ...
finalMessage := ui.Success.Sprint("✓") + " " + ui.Highlight.Sprint(displayName) + " " + successVerb + " successfully!\n\n" +
    filesLabel + ":\n" +
    "  Public key:    " + ui.Path.Sprint(customFilePath) + " (provided)\n" +
    "  Encrypted key: " + ui.Path.Sprint(targetKanukaFilePath) + "\n\n" +
    ui.Info.Sprint("→") + " They now have access to decrypt the repository's secrets"
spinner.FinalMSG = finalMessage
return nil
```

The key issues:
1. **Line 678**: `targetUserUUID = strings.TrimSuffix(filepath.Base(customFilePath), ".pub")` - This extracts the filename base (e.g., "pubkey" if the file is `/path/to/pubkey.pub`). This is NOT a UUID.
2. **Line 744-747**: The final message shows `customFilePath` as the public key path, but no public key is actually copied to the project. The `handleCustomFileRegistration` function doesn't call `secrets.SavePublicKeyToFile()` - it only encrypts and saves the `.kanuka` file.
3. **No config update**: Unlike `handleUserRegistration` and `handlePubkeyTextRegistration`, this function doesn't update `projectConfig.Users` or `projectConfig.Devices` maps.

**Why This Is an Error:**
1. **Inconsistent naming:** `.kanuka` files should always be UUID-based for consistency
2. **Incomplete registration:** Public key not copied to project directory
3. **Config corruption:** Project config has no record of this user
4. **Breaks access list:** `kanuka secrets access` won't show this user properly
5. **Breaks revoke:** Cannot revoke by user email since user not in config

**Reproduction Steps:**
1. Create a public key file with a non-UUID filename (e.g., `pubkey.pub`)
2. Run `kanuka secrets register --file /path/to/pubkey.pub`
3. Observe:
   - Encrypted key is named `pubkey.kanuka` instead of `<uuid>.kanuka`
   - No public key in `.kanuka/public_keys/`
   - Project config not updated

**Expected Behavior:**
There are several possible correct behaviors:

**Option A:** Require the file to be UUID-named:
```
✗ Public key file must be named <uuid>.pub
→ Rename your public key file to use UUID, or use --user and --pubkey flags instead
```

**Option B:** Generate a UUID for the custom key:
- Generate a new UUID
- Copy the public key to `.kanuka/public_keys/<uuid>.pub`
- Create `.kanuka/secrets/<uuid>.kanuka`
- Update project config with the UUID and optionally email

**Option C:** Reject custom file registration entirely:
```
✗ Cannot register from file path
→ Use --user <email> with --pubkey <key-content> to register custom keys
```

---

## High Priority Issues

### ERR-008: Access Command Shows "test-project" When Not in Project

**Test Case:** TEST-002
**Command:**
```bash
kanuka secrets access
```

**Issue:**
When running `kanuka secrets access` outside of a Kanuka-initialized project, it displays "test-project" as the project name.

**Root Cause Analysis:**
Looking at `cmd/secrets_access.go:92-99`:

```go
projectName := projectConfig.Project.Name
if projectName == "" {
    projectName = configs.ProjectKanukaSettings.ProjectName
}
Logger.Debugf("Project name: %s", projectName)
```

The issue is that when no project config exists, `configs.LoadProjectConfig()` likely returns a default/empty config, and then `configs.ProjectKanukaSettings.ProjectName` is used. Looking at the test output and the grep results, "test-project" appears to be a fallback/test value.

Actually, looking more closely at `configs.LoadProjectConfig()` and related functions might reveal where "test-project" comes from. It could be from test fixtures or from an empty config having a default name.

**Why This Is an Error:**
1. **Confusing output:** Shows a specific project name when no project exists
2. **Misleading:** User might think they're in a project called "test-project"
3. **Inconsistent:** Other commands show "not initialized" message

**Reproduction Steps:**
1. Navigate to a directory without `.kanuka`
2. Run `kanuka secrets access`
3. Observe "test-project" being displayed

**Expected Behavior:**
```
✗ Kanuka has not been initialized
→ Run 'kanuka secrets init' first
```

---

### ERR-009: Set-Device-Name Doesn't Update Project Config

**Test Case:** TEST-018
**Command:**
```bash
kanuka config set-device-name new-device-name
```

**Issue:**
When setting a device name with `kanuka config set-device-name`, the user config is updated but the project config is not updated to reflect the new device name.

**Root Cause Analysis:**
Looking at `cmd/config_set_device_name.go:98-148`:

```go
// Set the device name, preserving existing project name if available.
if hasExisting && existingEntry.ProjectName != "" {
    projectName = existingEntry.ProjectName
}
userConfig.Projects[projectUUID] = configs.UserProjectEntry{
    DeviceName:  deviceName,
    ProjectName: projectName,
}
ConfigLogger.Debugf("Setting device name for project %s to %s", projectUUID, deviceName)

if err := configs.SaveUserConfig(userConfig); err != nil {
    return ConfigLogger.ErrorfAndReturn("Failed to save user config: %v", err)
}

ConfigLogger.Infof("Device name set successfully")
// ... final message built
```

The function only updates `userConfig.Projects[projectUUID]` but never updates `projectConfig.Devices[projectUUID]`. Looking at `internal/configs/config.go` for the structure of these configs:

```go
type UserProjectEntry struct {
    DeviceName  string
    ProjectName string
}

type DeviceConfig struct {
    Email     string
    Name      string
    CreatedAt time.Time
}

type ProjectConfig struct {
    Project   Project
    Users     map[string]string        // UUID -> email
    Devices   map[string]DeviceConfig  // UUID -> DeviceConfig
}
```

When `set-device-name` is called, it updates `UserProjectEntry` (which stores `DeviceName` and `ProjectName` in the user's config) but does not update the `DeviceConfig` in the project config. This means:
- `kanuka secrets access` will still show the old device name
- `kanuka config list-devices` will show the new device name (because it reads from user config)
- Inconsistent state between user and project configs

**Why This Is an Error:**
1. **Inconsistent state:** User config and project config disagree on device name
2. **Misleading:** Running `kanuka secrets access` still shows old device name
3. **Confusing:** Different commands show different device names

**Reproduction Steps:**
1. In an initialized project, run `kanuka config set-device-name new-name`
2. Run `kanuka secrets access` and observe old device name
3. Run `kanuka config list-devices` and observe new device name

**Expected Behavior:**
Both user config and project config should be updated with the new device name:
```bash
✓ Device name set to new-name for project my-project
```

After this, `kanuka secrets access` should show the updated device name.

---

### ERR-010: Invalid Archive Import Creates Blank Config

**Test Case:** TEST-127
**Command:**
```bash
kanuka secrets import incomplete-backup.tar.gz
```

**Issue:**
When importing an archive that's missing the `config.toml` file, the import command succeeds and creates a blank config file instead of failing with a clear error.

**Root Cause Analysis:**
Looking at `cmd/secrets_import.go:31-57` (validateArchiveStructure function):

```go
func validateArchiveStructure(files []string) error {
    hasConfig := false
    hasContent := false

    for _, f := range files {
        if f == ".kanuka/config.toml" {
            hasConfig = true
        }
        // Check for any content in public_keys, secrets, or .kanuka files.
        if strings.HasPrefix(f, ".kanuka/public_keys/") ||
            strings.HasPrefix(f, ".kanuka/secrets/") ||
            strings.HasSuffix(f, ".kanuka") {
            hasContent = true
        }
    }

    if !hasConfig {
        return fmt.Errorf("archive missing .kanuka/config.toml")
    }

    if !hasContent {
        return fmt.Errorf("archive contains no encrypted content (public_keys, secrets, or .kanuka files)")
    }

    return nil
}
```

The validation correctly checks for `config.toml`. However, the actual extraction happens in `performImport` which doesn't re-validate. Looking at `cmd/secrets_import.go:284-394` (performImport function):

```go
// Extract file.
if err := extractFile(tarReader, targetPath, header.Mode); err != nil {
    return nil, fmt.Errorf("failed to extract %s: %w", header.Name, err)
}
```

The `extractFile` function extracts all files from the archive. If `config.toml` is missing from the archive, it won't be extracted. But the command continues because validation passed before checking the actual content.

Wait, looking more closely at the validation logic - it checks for `.kanuka/config.toml` in the `files` list (which comes from `listArchiveContents`). If the archive doesn't have this file, `hasConfig` is `false` and the function should return an error at line 248-250:

```go
if !hasConfig {
    return fmt.Errorf("archive missing .kanuka/config.toml")
}
```

But the test notes say "Fail. The import succeeded, and created a blank config file." This suggests the validation passed (meaning the archive DID have a `config.toml` file, but it was empty or invalid).

Actually, looking at the extraction code at lines 404-418:

```go
// Extract file.
if err := extractFile(tarReader, targetPath, header.Mode); err != nil {
    return nil, fmt.Errorf("failed to extract %s: %w", header.Name, err)
}
```

If the archive's `config.toml` is empty or has invalid TOML, it will still be extracted. Then at line 387-391:

```go
// Re-initialize project settings after import.
if !dryRun {
    if err := configs.InitProjectSettings(); err != nil {
        Logger.Debugf("Warning: failed to reinitialize project settings: %v", err)
    }
}
```

The `InitProjectSettings()` will load the (possibly invalid) config and continue. If the config is empty/invalid, it might just use defaults or fail silently.

**Why This Is an Error:**
1. **Data integrity:** Invalid/corrupt config imported
2. **Silent failure:** Command succeeds but creates broken state
3. **Poor error handling:** Invalid config should be caught and reported

**Reproduction Steps:**
1. Create a backup archive without `config.toml` or with an empty one
2. Import the archive with `kanuka secrets import backup.tar.gz`
3. Observe success message with invalid config created

**Expected Behavior:**
```
✗ Invalid archive: missing .kanuka/config.toml
→ Ensure your backup was created with 'kanuka secrets export'
```

---

### ERR-011: Encrypt Without Access Shows Go Error Instead of Friendly Message

**Test Case:** TEST-045
**Command:**
```bash
rm .kanuka/secrets/*.kanuka
kanuka secrets encrypt
```

**Issue:**
When a user doesn't have access (no `.kanuka` encrypted key file), the encrypt command shows a Go error message instead of a user-friendly error.

**Root Cause Analysis:**
Looking at `cmd/secrets_encrypt.go:141-149`:

```go
Logger.Debugf("Getting project kanuka key for user: %s", userUUID)
encryptedSymKey, err := secrets.GetProjectKanukaKey(userUUID)
if err != nil {
    Logger.Errorf("Failed to obtain kanuka key for user %s: %v", userUUID, err)
    finalMessage := ui.Error.Sprint("✗") + " Failed to get your " +
        ui.Path.Sprint(".kanuka") + " file. Are you sure you have access?\n" +
        ui.Error.Sprint("Error: ") + err.Error()
    spinner.FinalMSG = finalMessage
    return nil
}
```

The code correctly provides a user-friendly error ("Failed to get your .kanuka file. Are you sure you have access?") but then appends the raw Go error (`err.Error()`). This results in:

```
✗ Failed to get your .kanuka file. Are you sure you have access?
Error: failed to get user's project encrypted symmetric key: stat /Users/aaron/Developer/testing/acceptance_testing/.kanuka/secrets/beafe009-1cc0-44e3-83e2-2071304c5144.kanuka: no such file or directory
```

The raw Go error includes the full error chain and technical details that are confusing to end users.

**Why This Is an Error:**
1. **Poor UX:** Technical Go error shown to users
2. **Redundant information:** User-friendly message already explains the issue
3. **Inconsistent:** Some commands have better error handling

**Reproduction Steps:**
1. Remove your `.kanuka/secrets/<uuid>.kanuka` file
2. Run `kanuka secrets encrypt`
3. Observe the technical Go error appended to the friendly message

**Expected Behavior:**
```
✗ Failed to get your .kanuka file. Are you sure you have access?

→ You don't have access to this project. Ask someone with access to run:
   kanuka secrets register --user <your-email>
```

---

### ERR-012: Register Without Access Shows Go RSA Error

**Test Case:** TEST-070
**Command:**
```bash
rm .kanuka/secrets/<your-uuid>.kanuka
kanuka secrets register --user alice@example.com
```

**Issue:**
When trying to register a user when the current user doesn't have access (no `.kanuka` file), the command shows a raw Go RSA decryption error instead of a user-friendly error.

**Root Cause Analysis:**
Looking at `cmd/secrets_register.go:512-523` (handleUserRegistration):

```go
encryptedSymKey, err := secrets.GetProjectKanukaKey(currentUserUUID)
if err != nil {
    Logger.Errorf("Failed to get kanuka key for current user %s: %v", currentUserUUID, err)
    finalMessage := ui.Error.Sprint("✗") + " Couldn't get your Kānuka key from " + ui.Path.Sprint(kanukaKeyPath) + "\n\n" +
        "Are you sure you have access?\n\n" +
        ui.Error.Sprint("Error: ") + err.Error()
    spinner.FinalMSG = finalMessage
    return nil
}

// Decrypt symmetric key with current user's private key
Logger.Debugf("Decrypting symmetric key with current user's private key")
_, err = secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
if err != nil {
    Logger.Errorf("Failed to decrypt symmetric key: %v", err)
    finalMessage := ui.Error.Sprint("✗") + " Failed to decrypt your Kānuka key using your private key: \n" +
        "    Kānuka key path: " + ui.Path.Sprint(kanukaKeyPath) + "\n" +
        "    Private key path: " + ui.Path.Sprint(privateKeyPath) + "\n\n" +
        "Are you sure you have access?\n\n" +
        ui.Error.Sprint("Error: ") + err.Error()
    spinner.FinalMSG = finalMessage
    return nil
}
```

Same issue as ERR-011 - the code provides a user-friendly message but then appends the raw Go error. When the file doesn't exist, `GetProjectKanukaKey` returns an os error, and then when trying to decrypt with a non-existent file, the RSA decryption fails.

**Why This Is an Error:**
1. **Poor UX:** Raw Go "crypto/rsa: decryption error" shown to users
2. **Confusing:** Technical error doesn't help user understand the problem
3. **Inconsistent:** Some places handle errors better than others

**Reproduction Steps:**
1. Remove your `.kanuka/secrets/<your-uuid>.kanuka` file
2. Run `kanuka secrets register --user alice@example.com`
3. Observe the RSA decryption error message

**Expected Behavior:**
```
✗ Couldn't get your Kānuka key from /path/to/project/.kanuka/secrets/<uuid>.kanuka

Are you sure you have access?

→ You don't have access to this project. Run 'kanuka secrets create' to generate your keys
```

---

## Medium Priority Issues

### ERR-013: Revoke --device Without --user Shows Wrong Error Message

**Test Case:** TEST-087
**Command:**
```bash
kanuka secrets revoke --device device1
```

**Issue:**
When using `--device` without `--user`, the error message is incorrect - it says "Either --user or --file flag is required" instead of the more specific "--device requires --user flag".

**Root Cause Analysis:**
Looking at `cmd/secrets_revoke.go:35-47`:

```go
if revokeUserEmail == "" && revokeFilePath == "" {
    finalMessage := ui.Error.Sprint("✗") + " Either " + ui.Flag.Sprint("--user") + " or " + ui.Flag.Sprint("--file") + " flag is required.\n" +
        "Run " + ui.Code.Sprint("kanuka secrets revoke --help") + " to see the available commands.\n"
    spinner.FinalMSG = finalMessage
    return nil
}

// ...

// --device requires --user
if revokeDevice != "" && revokeUserEmail == "" {
    finalMessage := ui.Error.Sprint("✗") + " The " + ui.Flag.Sprint("--device") + " flag requires " + ui.Flag.Sprint("--user") + " flag.\n" +
        "Run " + ui.Code.Sprint("kanuka secrets revoke --help") + " to see the available commands.\n"
    spinner.FinalMSG = finalMessage
    return nil
}
```

The issue is the order of checks. The first check (lines 35-40) is checked first and catches the case where `--device` is used without `--user`. Only the second check (lines 49-55) would give the specific error message about `--device` requiring `--user`, but the first check returns early.

**Why This Is an Error:**
1. **Incorrect error message:** Doesn't explain that `--device` requires `--user`
2. **Helpful tip missing:** User doesn't know which flag they forgot
3. **Inconsistent with docs:** Help text says `--device` requires `--user`

**Reproduction Steps:**
1. Run `kanuka secrets revoke --device device1`
2. Observe error: "Either --user or --file flag is required"

**Expected Behavior:**
```
✗ The --device flag requires the --user flag.
Run 'kanuka secrets revoke --help' to see the available commands.
```

---

### ERR-014: Invalid Archive Import Shows Go Error

**Test Case:** TEST-126
**Command:**
```bash
echo "not a tar" > fake.tar.gz
kanuka secrets import fake.tar.gz
```

**Issue:**
When importing an invalid archive, the command shows a Go error message about gzip header instead of a user-friendly error.

**Root Cause Analysis:**
Looking at `cmd/secrets_import.go:105-114`:

```go
// Validate archive structure.
Logger.Debugf("Validating archive structure")
archiveFiles, err := listArchiveContents(archivePath)
if err != nil {
    return Logger.ErrorfAndReturn("failed to read archive: %v", err)
}
```

The error from `listArchiveContents` is returned via `Logger.ErrorfAndReturn`, which wraps the Go error. Looking at `cmd/secrets_import.go:200-229` (listArchiveContents):

```go
gzReader, err := gzip.NewReader(file)
if err != nil {
    return nil, fmt.Errorf("failed to create gzip reader: %w", err)
}
```

The Go library returns `gzip: invalid header` error, which is wrapped and shown to the user.

**Why This Is an Error:**
1. **Poor UX:** Technical "gzip: invalid header" error shown to users
2. **Unclear:** User doesn't know what's wrong with their archive
3. **Inconsistent:** Other commands have better error messages

**Reproduction Steps:**
1. Create an invalid gzip file (e.g., `echo "not a tar" > fake.tar.gz`)
2. Run `kanuka secrets import fake.tar.gz`
3. Observe the gzip error message

**Expected Behavior:**
```
✗ Invalid archive file: fake.tar.gz

→ The file is not a valid gzip archive. Ensure it was created with:
   kanuka secrets export
```

---

### ERR-015: Import Both Flags Shows Go Error

**Test Case:** TEST-128
**Command:**
```bash
kanuka secrets import backup.tar.gz --merge --replace
```

**Issue:**
When using both `--merge` and `--replace` flags, the command shows a Go error message instead of a user-friendly error.

**Root Cause Analysis:**
Looking at `cmd/secrets_import.go:92-95`:

```go
// Validate flags - can't use both merge and replace.
if importMergeFlag && importReplaceFlag {
    return Logger.ErrorfAndReturn("cannot use both --merge and --replace flags")
}
```

The error is returned via `Logger.ErrorfAndReturn`, which wraps the Go error. The string "cannot use both --merge and --replace flags" is a user-created message, but the way it's returned includes Go error formatting.

Looking at the test output:
```
✗ cannot use both --merge and --replace flags
Error: cannot use both --merge and --replace flags
```

The error is shown twice - once with the `✗` prefix and once as the raw Go error.

**Why This Is an Error:**
1. **Redundant output:** Error message shown twice
2. **Inconsistent:** Other commands show cleaner error messages
3. **Confusing:** Raw error format shown

**Reproduction Steps:**
1. Run `kanuka secrets import backup.tar.gz --merge --replace`
2. Observe error shown twice with raw error format

**Expected Behavior:**
```
✗ Cannot use both --merge and --replace flags.

→ Use --merge to add new files while keeping existing files,
   or use --replace to delete existing files and use only the backup.
```

---

### ERR-016: Log --oneline Not Actually One Line

**Test Case:** TEST-135
**Command:**
```bash
kanuka secrets log --oneline
```

**Issue:**
The `--oneline` flag doesn't actually format the output as a single line per entry. Instead, it shows each entry on its own line (which is the default behavior).

**Root Cause Analysis:**
Looking at `cmd/secrets_log.go:258-265` (outputLogOneline function):

```go
func outputLogOneline(entries []audit.Entry) error {
    for _, e := range entries {
        date := formatDate(e.Timestamp)
        details := formatDetailsOneline(e)
        fmt.Printf("%s %s %s %s\n", date, e.User, e.Operation, details)
    }
    return nil
}
```

The function is correctly outputting one line per entry (it uses `fmt.Printf` with `\n` at the end). The issue is that the user expected something different.

Looking at the test output:
```bash
kanuka secrets log --oneline
2026-01-13 aaron@guo.nz init acceptance_testing_2
2026-01-13 aaron2@guo.nz create aarons-macbook-pro-2local
2026-01-13 aaron@guo.nz register aaron2@guo.nz
2026-01-13 aaron@guo.nz register pubkey
2026-01-13 aaron@guo.nz register aaron2@guo.nz
2026-01-13 aaron@guo.nz encrypt 5 files
```

This IS one line per entry. The user's test notes say "Fail. I don't think this exported as one line."

Maybe the expected behavior was to have all entries on a SINGLE line? Or maybe the formatting is wrong?

Looking at the regular output in `outputLogDefault` (lines 267-273):

```go
func outputLogDefault(entries []audit.Entry) error {
    for _, e := range entries {
        datetime := formatDateTime(e.Timestamp)
        details := formatDetails(e)
        fmt.Printf("%-19s  %-25s  %-10s  %s\n", datetime, e.User, e.Operation, details)
    }
    return nil
}
```

The default output has more columns (datetime, user, operation, details). The `--oneline` version has fewer columns (date, user, operation, details). So it IS more compact.

Perhaps the issue is that there's no way to distinguish between multi-line and single-line output visually? Or perhaps the user expected even more compact format (e.g., tab-separated)?

Actually, looking at the test notes more carefully: "Fail. I don't think this exported as one line."

I think the user expected the output to look more like git log's `--oneline` format which is a single line with hash and message. Or perhaps they expected it to be on a single continuous line without breaks?

**Why This Is an Error:**
1. **Unclear specification:** What does "oneline" mean in this context?
2. **Potentially misleading:** Flag name suggests behavior that might not be implemented
3. **Inconsistent with common CLI tools:** Git and other tools have different `--oneline` semantics

**Reproduction Steps:**
1. Run `kanuka secrets log` to see default format
2. Run `kanuka secrets log --oneline`
3. Compare - the difference is minor (date format vs datetime)

**Expected Behavior:**
This depends on what the intended behavior is. Options:

**Option A:** Single line per entry (current behavior - may be correct):
```bash
2026-01-13 aaron@guo.nz init acceptance_testing_2
2026-01-13 aaron2@guo.nz create aarons-macbook-pro-2local
```

**Option B:** All entries on one line:
```bash
2026-01-13 aaron@guo.nz init acceptance_testing_2 | 2026-01-13 aaron2@guo.nz create aarons-macbook-pro-2local | ...
```

**Option C:** More compact format:
```bash
init acceptance_testing_2 | create aarons-macbook-pro-2local | register aaron2@guo.nz | ...
```

Documentation should clarify what `--oneline` means.

---

### ERR-017: Corrupted .kanuka File Shows Go Error

**Test Case:** TEST-151
**Command:**
```bash
echo "garbage" > .kanuka/secrets/<uuid>.kanuka
kanuka secrets decrypt
```

**Issue:**
When a `.kanuka` file is corrupted, the decrypt command shows a raw Go RSA decryption error instead of a user-friendly error.

**Root Cause Analysis:**
Looking at `cmd/secrets_decrypt.go:198-207`:

```go
Logger.Debugf("Decrypting symmetric key with private key")
symKey, err := secrets.DecryptWithPrivateKey(encryptedSymKey, privateKey)
if err != nil {
    Logger.Errorf("Failed to decrypt symmetric key: %v", err)
    finalMessage := ui.Error.Sprint("✗") + " Failed to decrypt your " +
        ui.Path.Sprint(".kanuka") + " file. Are you sure you have access?\n" +
        ui.Error.Sprint("Error: ") + err.Error()
    spinner.FinalMSG = finalMessage
    return nil
}
```

Same pattern as ERR-011 and ERR-012 - user-friendly message provided but raw Go error appended. The Go RSA library returns `crypto/rsa: decryption error` when decryption fails.

**Why This Is an Error:**
1. **Poor UX:** Technical error message shown to users
2. **Unhelpful:** "crypto/rsa: decryption error" doesn't help user fix the problem
3. **Security concern:** Users might try random fixes based on technical error

**Reproduction Steps:**
1. Corrupt a `.kanuka` file with garbage data
2. Run `kanuka secrets decrypt`
3. Observe the RSA decryption error

**Expected Behavior:**
```
✗ Failed to decrypt your .kanuka file. Are you sure you have access?

→ Your encrypted key file appears to be corrupted.
   Try asking the project administrator to revoke and re-register your access.
```

---

### ERR-018: Corrupted config.toml Shows Go Error

**Test Case:** TEST-152
**Command:**
```bash
echo "not valid toml [" > .kanuka/config.toml
kanuka secrets status
```

**Issue:**
When the project's `config.toml` is invalid/corrupt, commands show a raw Go TOML parsing error instead of a user-friendly error.

**Root Cause Analysis:**
Looking at `cmd/secrets_status.go:98-101`:

```go
// Load project config for project name.
projectConfig, err := configs.LoadProjectConfig()
if err != nil {
    return Logger.ErrorfAndReturn("failed to load project config: %v", err)
}
```

The error from `LoadProjectConfig()` is wrapped and returned. Looking at `internal/configs/toml.go`, the TOML parsing library returns errors like `toml: line 2: expected '.' or ']' to end table name`. This is shown directly to the user.

**Why This Is an Error:**
1. **Poor UX:** Technical TOML parsing error shown to users
2. **Unhelpful:** Error message doesn't explain how to fix
3. **Security concern:** Users might try editing config manually and make it worse

**Reproduction Steps:**
1. Corrupt `.kanuka/config.toml` with invalid TOML
2. Run any command that loads project config (e.g., `kanuka secrets status`)
3. Observe the TOML parsing error

**Expected Behavior:**
```
✗ Failed to load project configuration.

→ The .kanuka/config.toml file is not valid TOML.
   Line 2: Expected '.' or ']' to end table name

   To fix this issue:
   1. Restore the file from git: git checkout .kanuka/config.toml
   2. Or contact your project administrator for assistance
```

---

## Low Priority Issues

### ERR-019: Read-only Filesystem Doesn't Prevent Encrypt

**Test Case:** TEST-160
**Command:**
```bash
chmod 555 .kanuka
kanuka secrets encrypt
```

**Issue:**
When the `.kanuka` directory is read-only (mode 555), the encrypt command still succeeds without errors.

**Root Cause Analysis:**
Looking at the test output:
```bash
chmod 555 .kanuka

kanuka secrets encrypt
✓ Environment files encrypted successfully!
The following files were created:
    - /Users/aaron/Developer/testing/acceptance_testing/.env.kanuka
    - ...
```

The encryption succeeded despite the `.kanuka` directory being read-only. This is concerning because:
1. New `.kanuka` files are being created in the `.kanuka` directory
2. The `.kanuka` directory has mode 555 (read+execute only)
3. Files should NOT be writable in a read-only directory

Looking at `cmd/secrets_encrypt.go:215-222`:

```go
Logger.Infof("Encrypting %d files", len(listOfEnvFiles))
if err := secrets.EncryptFiles(symKey, listOfEnvFiles, verbose); err != nil {
    Logger.Errorf("Failed to encrypt files: %v", err)
    finalMessage := ui.Error.Sprint("✗") + " Failed to encrypt the project's " +
        ui.Path.Sprint(".env") + " files. Are you sure you have access?\n" +
        ui.Error.Sprint("Error: ") + err.Error()
    spinner.FinalMSG = finalMessage
    return nil
}
```

The `secrets.EncryptFiles` function should be returning an error if it can't write files. Let's look at `internal/secrets/files.go` for the encryption code.

The issue might be that the OS allows writing even with `chmod 555` in some cases, or there's a permission check that's not being done before attempting to write.

Actually, `chmod 555` on a directory means:
- Owner: read+execute (5)
- Group: read+execute (5)
- Other: read+execute (5)

This is write permission for the directory itself, but files INSIDE the directory can still be written if they have their own permissions. When you create a file, the OS doesn't check the directory write permission on macOS - it just checks if you have write permission to the FILE's parent directory.

Wait, that's not quite right. On Unix-like systems, to create a file inside a directory, you need write+execute permission on the directory. With mode 555, there's no write permission.

But the test shows it succeeded... This could be:
1. macOS-specific behavior
2. The test user has special permissions
3. The directory had write permission despite chmod 555

Actually, I suspect this might be a false positive in the test - perhaps the directory actually WAS writable, or the user ran the test incorrectly.

**Why This Is an Error:**
1. **Security:** Files should not be writable in read-only directories
2. **Inconsistent:** Expected permission denied error didn't occur
3. **Silent failure:** No error shown when write should have failed

**Note:** This issue might be environment-specific or a test error. More investigation needed to confirm the actual behavior.

**Reproduction Steps:**
1. Run `chmod 555 .kanuka` to make it read-only
2. Run `kanuka secrets encrypt`
3. Observe if encryption succeeds or fails with permission error

**Expected Behavior:**
```
✗ Permission denied: Cannot write to .kanuka directory

→ The .kanuka directory is read-only (mode 555).
   Run: chmod 755 .kanuka to make it writable
```

---

## Summary by Category

### Error Handling Issues
| ID | Command | Issue | Severity |
|----|----------|-------|----------|
| ERR-011 | encrypt | Shows Go error instead of friendly message | Medium |
| ERR-012 | register | Shows Go error instead of friendly message | Medium |
| ERR-014 | import | Shows Go error instead of friendly message | Medium |
| ERR-015 | import | Shows Go error instead of friendly message | Medium |
| ERR-017 | decrypt | Shows Go error instead of friendly message | Medium |
| ERR-018 | status/any | Shows Go error instead of friendly message | Medium |

### Data Integrity Issues
| ID | Command | Issue | Severity |
|----|----------|-------|----------|
| ERR-002 | create | Creates keys before checking project | Critical |
| ERR-003 | init | Creates .kanuka folder too early | Critical |
| ERR-010 | import | Creates blank config from invalid archive | High |
| ERR-007 | register | Creates wrong filename, doesn't copy pubkey | High |

### Functional Issues
| ID | Command | Issue | Severity |
|----|----------|-------|----------|
| ERR-004 | encrypt | Glob pattern encrypts all files | High |
| ERR-005 | decrypt | Ignores file path arguments | High |
| ERR-008 | access | Shows test-project when not in project | High |
| ERR-009 | set-device-name | Doesn't update project config | High |

### UX Issues
| ID | Command | Issue | Severity |
|----|----------|-------|----------|
| ERR-001 | encrypt/decrypt/access/status | Hang when not in project | Critical |
| ERR-006 | register | Misleading "files created" message | Medium |
| ERR-013 | revoke | Wrong error message for --device flag | Medium |
| ERR-016 | log | --oneline flag unclear or not implemented | Low |

---

## Recommendations

### Immediate Fixes (Critical Priority)

1. **ERR-003: Init Cleanup**
   - Move `.kanuka` folder creation to after all prompts complete
   - Implement proper cleanup if init is cancelled or fails
   - Add check for incomplete init on subsequent runs

2. **ERR-002: Create Validation**
   - Check for project existence before creating keypair
   - Provide clear error message when not in a project
   - Document that `create` requires project to be initialized

3. **ERR-004 & ERR-005: Glob/File Pattern Issues**
   - Debug and fix `secrets.ResolveFiles()` function
   - Ensure specific file paths are respected
   - Add tests for glob pattern matching

4. **ERR-007: Register --file Issues**
   - Decide on the desired behavior for custom file registration
   - Either: require UUID-named files, generate UUID, or remove this feature
   - Update project config when registering from file
   - Copy public key to project directory

5. **ERR-001: Command Hanging**
   - Fix spinner cleanup logic
   - Ensure all commands properly handle missing project state
   - Add timeout for spinner operations
   - Remove hard-coded "test-project" fallback

### High Priority Fixes

1. **ERR-009: Set-Device-Name Consistency**
   - Update both user config AND project config
   - Ensure `kanuka secrets access` shows updated device name
   - Add test for this behavior

2. **ERR-010: Import Validation**
   - Validate config.toml content after extraction
   - Check if config is empty/invalid
   - Fail with clear error message if config is invalid

3. **ERR-008: Access Display Issue**
   - Remove or fix hard-coded "test-project" fallback
   - Ensure consistent error messages across all commands

### Medium Priority Improvements

1. **Error Message Consistency**
   - Create a unified error handling module
   - Wrap all Go errors with user-friendly messages
   - Add recovery suggestions for common errors
   - Remove raw Go error strings from user-facing output

2. **ERR-013: Revoke Error Messages**
   - Reorder validation checks to check `--device` flag before general check
   - Provide specific error messages for each validation case

3. **ERR-016: Log --oneline Documentation**
   - Clarify what `--oneline` is supposed to do
   - Document the expected output format
   - Or implement the expected behavior if it's wrong

### Testing Recommendations

1. **Add Integration Tests**
   - Test error scenarios (invalid files, missing directories, etc.)
   - Test glob pattern matching with various patterns
   - Test cancellation scenarios
   - Test permission-related operations

2. **Add E2E Tests**
   - Test complete user workflows (init → create → register → encrypt)
   - Test error recovery scenarios
   - Test multi-user scenarios

3. **Improve Test Coverage**
   - Add tests for edge cases (empty files, large files, special chars)
   - Test permission handling
   - Test corrupted file handling

---

## Appendix: Code Analysis Notes

### File Pattern Resolution Flow

```
User Command
    ↓
cmd/secrets_encrypt.go or cmd/secrets_decrypt.go
    ↓
secrets.ResolveFiles(args, projectPath, forEncryption)
    ↓
secrets.resolvePattern(pattern, projectPath, forEncryption)
    ↓
├─→ expandGlob() → doublestar.FilepathGlob()
├─→ findFilesInDir() → filepath.WalkDir()
└─→ literal file → os.Stat() + validation
```

The issue in ERR-004 and ERR-005 appears to be in how the pattern is resolved or how the glob library is used.

### Error Message Handling Pattern

Most error handling follows this pattern:

```go
if err != nil {
    Logger.Errorf("Failed to do something: %v", err)
    finalMessage := ui.Error.Sprint("✗") + " User-friendly message\n" +
        ui.Error.Sprint("Error: ") + err.Error()
    spinner.FinalMSG = finalMessage
    return nil
}
```

The problem is the `err.Error()` part which exposes raw Go errors to users.

### Config File Loading

```
LoadProjectConfig()
    ↓
Reads ~/.kanuka/config.toml
    ↓
TOML parsing (internal/configs/toml.go)
    ↓
Returns ProjectConfig struct
    ↓
Used by various commands
```

Corrupted config files should be caught and handled gracefully.

---

## Test Environment Details

- **Tester:** Aaron Guo
- **Date:** 2026-01-13
- **Platform:** macOS (darwin/arm64)
- **Kanuka Version:** 1.2.1
- **Go Version:** 1.24.5
- **Test Duration:** Not specified

---

## Related Files

This document references issues found in the following files:

- `cmd/secrets_encrypt.go` - ERR-004, ERR-005, ERR-011
- `cmd/secrets_decrypt.go` - ERR-005, ERR-017
- `cmd/secrets_init.go` - ERR-003
- `cmd/secrets_create.go` - ERR-002
- `cmd/secrets_register.go` - ERR-006, ERR-007, ERR-012
- `cmd/secrets_access.go` - ERR-001, ERR-008
- `cmd/secrets_status.go` - ERR-018
- `cmd/secrets_revoke.go` - ERR-013
- `cmd/secrets_log.go` - ERR-016
- `cmd/secrets_import.go` - ERR-010, ERR-014, ERR-015
- `cmd/config_set_device_name.go` - ERR-009
- `internal/secrets/files.go` - ERR-004, ERR-005
- `internal/configs/settings.go` - ERR-002, ERR-003
- `internal/configs/toml.go` - ERR-018
