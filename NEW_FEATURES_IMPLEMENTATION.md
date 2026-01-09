# New Features Implementation Guide

This document contains detailed implementation tickets for Kanuka's next set of features. Each ticket is self-contained with full context, allowing any developer to pick it up and implement it.

---

## Table of Contents

1. [KAN-012: Remove KANUKA_DATA_DIR from Documentation](#kan-012-remove-kanuka_data_dir-from-documentation)
2. [KAN-013: Audit Log](#kan-013-audit-log)
3. [KAN-014: Log Command](#kan-014-log-command)
4. [KAN-015: Selective File Encryption](#kan-015-selective-file-encryption)
5. [KAN-016: Update Init Command for Monorepo Guidance](#kan-016-update-init-command-for-monorepo-guidance)
6. [KAN-017: Documentation Updates](#kan-017-documentation-updates)

---

## Implementation Order & Dependencies

```
KAN-012 (Remove KANUKA_DATA_DIR)    ─── DONE
     │
KAN-013 (Audit Log)                 ─── DONE
     │
     └── KAN-014 (Log Command)      ─── DONE

KAN-015 (Selective Encryption)      ─── DONE
     │
     └── KAN-016 (Init Monorepo)    ─── Logically follows KAN-015

KAN-017 (Documentation)             ─── Do last, covers all new features
```

---

## KAN-012: Remove KANUKA_DATA_DIR from Documentation

### Summary

Remove the `KANUKA_DATA_DIR` environment variable from documentation. It was documented speculatively but never implemented.

### Priority

**Low** - Quick documentation cleanup

### Context

The configuration documentation at `docs/src/content/docs/configuration/configuration.mdx` advertises a `KANUKA_DATA_DIR` environment variable that doesn't exist in the codebase. This is a case of documentation preceding implementation, which creates confusion.

**Why not implement it instead?**

- No demonstrated user need
- XDG defaults are correct for each platform
- Users who need custom paths can use symlinks
- Every config option adds maintenance burden
- If demand emerges later, it's a trivial addition

### Current State

```mdx
### `KANUKA_DATA_DIR`

Overrides the default data directory where Kānuka stores user-specific files like private keys.

**Default**: 
- Linux/macOS: `$XDG_DATA_HOME/kanuka` (usually `~/.local/share/kanuka`)
- Windows: `%APPDATA%\kanuka`

**Example**:
```bash
export KANUKA_DATA_DIR=/custom/path/to/kanuka
kanuka init
```
```

### New State

Remove the entire `KANUKA_DATA_DIR` section from the documentation.

### Files to Modify

| File | Changes |
|------|---------|
| `docs/src/content/docs/configuration/configuration.mdx` | Remove `KANUKA_DATA_DIR` section |

### Acceptance Criteria

- [ ] `KANUKA_DATA_DIR` section removed from `configuration.mdx`
- [ ] No other documentation references this variable
- [ ] Build docs locally to verify no broken links

### Testing Requirements

None - documentation only.

### Definition of Done

- [ ] Documentation updated
- [ ] No dead references to `KANUKA_DATA_DIR`
- [ ] Docs build successfully

---

## KAN-013: Audit Log

### Summary

Implement an append-only audit log that records who performed what operation and when, providing a paper trail for team visibility.

### Priority

**High** - Valuable for teams using Kanuka in professional contexts

### Context

Teams using Kanuka for secrets management need visibility into activity:

- "Who encrypted the secrets last?"
- "When was this user revoked?"
- "Who registered that new team member?"

An audit log is table stakes for security tooling in a team environment. It provides accountability and helps with debugging when things go wrong.

### Design Decisions

#### Log Format: JSON Lines

Use JSON Lines (`.jsonl`) format - one JSON object per line:

```jsonl
{"ts":"2024-01-15T10:30:00.123456Z","user":"alice@example.com","uuid":"a1b2c3d4","op":"encrypt","files":[".env",".env.local"]}
{"ts":"2024-01-15T10:35:00.456789Z","user":"bob@example.com","uuid":"b2c3d4e5","op":"register","target_user":"charlie@example.com","target_uuid":"c3d4e5f6"}
{"ts":"2024-01-15T11:00:00.789012Z","user":"alice@example.com","uuid":"a1b2c3d4","op":"revoke","target_user":"charlie@example.com","target_uuid":"c3d4e5f6"}
```

**Rationale:**
- Easy to parse programmatically (`json.Unmarshal` per line)
- Easy to append (no array wrapper to manage)
- Easy to grep/filter with standard Unix tools
- Human-readable enough for quick debugging
- Standard format with broad tooling support

#### Log Location

Store at `.kanuka/audit.jsonl` - committed to the repository alongside other Kanuka files.

**Rationale:**
- Versioned with the project via git
- Visible to anyone with repo access
- No external dependencies

#### Conflict Handling: Accept Conflicts

Accept git merge conflicts when they occur.

**Rationale:**
- Conflicts are rare (requires simultaneous operations by different users)
- Resolution is trivial (keep both lines - it's append-only data)
- The audit log is not a security control (attackers can delete it anyway)
- Complexity of conflict avoidance far exceeds the benefit

**Mitigation:** Use microsecond-precision timestamps to reduce collision likelihood.

#### No Digital Signatures

Skip signing log entries.

**Rationale:**
- The log is protected by git's commit history (and commit signatures if used)
- Signing adds complexity: key rotation, compromised keys, verification tooling
- The threat model doesn't justify the complexity (malicious actors can delete the file)
- Can be added later as a non-breaking enhancement if needed

### Operations to Log

| Operation | Additional Fields |
|-----------|-------------------|
| `init` | `project_name`, `project_uuid` |
| `create` | `device_name` |
| `encrypt` | `files[]` |
| `decrypt` | `files[]` |
| `register` | `target_user`, `target_uuid` |
| `revoke` | `target_user`, `target_uuid`, `device` (if specific) |
| `sync` | `users_count`, `files_count` |
| `rotate` | (no additional fields) |
| `clean` | `removed_count` |
| `import` | `mode` (merge/replace), `files_count` |
| `export` | `output_path` |

### Technical Details

#### Files to Create

| File | Purpose |
|------|---------|
| `internal/audit/audit.go` | Core audit logging functionality |
| `internal/audit/audit_test.go` | Unit tests for audit package |

#### Files to Modify

| File | Changes |
|------|---------|
| `cmd/secrets_init.go` | Log init operation |
| `cmd/secrets_create.go` | Log create operation |
| `cmd/secrets_encrypt.go` | Log encrypt operation |
| `cmd/secrets_decrypt.go` | Log decrypt operation |
| `cmd/secrets_register.go` | Log register operation |
| `cmd/secrets_revoke.go` | Log revoke operation |
| `cmd/secrets_sync.go` | Log sync operation |
| `cmd/secrets_rotate.go` | Log rotate operation |
| `cmd/secrets_clean.go` | Log clean operation |
| `cmd/secrets_import.go` | Log import operation |
| `cmd/secrets_export.go` | Log export operation |

#### Data Structures

```go
// internal/audit/audit.go

package audit

import (
    "encoding/json"
    "os"
    "path/filepath"
    "time"

    "github.com/PolarWolf314/kanuka/internal/configs"
)

// Entry represents a single audit log entry.
type Entry struct {
    Timestamp   string   `json:"ts"`                      // RFC3339 with microseconds
    User        string   `json:"user"`                    // Email of user performing action
    UserUUID    string   `json:"uuid"`                    // UUID of user performing action
    Operation   string   `json:"op"`                      // Operation name
    
    // Optional fields depending on operation
    Files       []string `json:"files,omitempty"`         // For encrypt/decrypt
    TargetUser  string   `json:"target_user,omitempty"`   // For register/revoke
    TargetUUID  string   `json:"target_uuid,omitempty"`   // For register/revoke
    Device      string   `json:"device,omitempty"`        // For device-specific revoke
    UsersCount  int      `json:"users_count,omitempty"`   // For sync
    FilesCount  int      `json:"files_count,omitempty"`   // For sync/import
    RemovedCount int     `json:"removed_count,omitempty"` // For clean
    Mode        string   `json:"mode,omitempty"`          // For import (merge/replace)
    OutputPath  string   `json:"output_path,omitempty"`   // For export
    ProjectName string   `json:"project_name,omitempty"`  // For init
    ProjectUUID string   `json:"project_uuid,omitempty"`  // For init
    DeviceName  string   `json:"device_name,omitempty"`   // For create
}
```

#### Implementation

```go
// internal/audit/audit.go

// Log appends an entry to the audit log.
// If logging fails, it logs a warning but does not return an error.
// Operations should not fail just because audit logging failed.
func Log(entry Entry) {
    // Set timestamp if not already set
    if entry.Timestamp == "" {
        entry.Timestamp = time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
    }
    
    // Get project path
    projectPath := configs.ProjectKanukaSettings.ProjectPath
    if projectPath == "" {
        // Project not initialized, skip logging
        return
    }
    
    logPath := filepath.Join(projectPath, ".kanuka", "audit.jsonl")
    
    // Open file for appending (create if doesn't exist)
    f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        // Log warning but don't fail the operation
        // Use stderr or internal logger
        return
    }
    defer f.Close()
    
    // Marshal entry to JSON
    data, err := json.Marshal(entry)
    if err != nil {
        return
    }
    
    // Write entry with newline
    _, _ = f.Write(append(data, '\n'))
}

// LogWithUser is a convenience function that populates user fields from config.
func LogWithUser(op string) Entry {
    userConfig, err := configs.LoadUserConfig()
    if err != nil {
        return Entry{Operation: op}
    }
    
    return Entry{
        User:      userConfig.User.Email,
        UserUUID:  userConfig.User.UUID,
        Operation: op,
    }
}
```

#### Usage in Commands

```go
// Example: cmd/secrets_encrypt.go

import "github.com/PolarWolf314/kanuka/internal/audit"

// After successful encryption:
entry := audit.LogWithUser("encrypt")
entry.Files = encryptedFiles
audit.Log(entry)
```

```go
// Example: cmd/secrets_register.go

// After successful registration:
entry := audit.LogWithUser("register")
entry.TargetUser = targetEmail
entry.TargetUUID = targetUUID
audit.Log(entry)
```

```go
// Example: cmd/secrets_init.go

// After successful init:
entry := audit.LogWithUser("init")
entry.ProjectName = projectName
entry.ProjectUUID = projectConfig.Project.UUID
audit.Log(entry)
```

#### Error Handling

Audit logging should **never** cause an operation to fail:

```go
// CORRECT: Log failures are silent
audit.Log(entry)

// WRONG: Don't propagate audit errors
if err := audit.Log(entry); err != nil {
    return err  // NO! Don't do this
}
```

The audit log is informational. If it can't be written (permissions, disk full, etc.), the actual operation should still succeed.

### Acceptance Criteria

- [ ] `internal/audit/audit.go` implemented with `Log()` and `LogWithUser()` functions
- [ ] `Entry` struct defined with all necessary fields
- [ ] Audit log created at `.kanuka/audit.jsonl` on first logged operation
- [ ] All 11 secrets commands log their operations
- [ ] Log entries contain timestamp, user, operation, and relevant details
- [ ] Timestamps use RFC3339 with microsecond precision
- [ ] Log is append-only (no modifications to existing entries)
- [ ] Logging failures are silent (don't break operations)
- [ ] Log file permissions are 0644 (readable by all, writable by owner)

### Testing Requirements

#### Unit Tests

Create `internal/audit/audit_test.go`:

| Test Case | Description |
|-----------|-------------|
| `TestLog_CreatesFile` | First log call creates the file |
| `TestLog_AppendsEntries` | Multiple calls append to same file |
| `TestLog_ValidJSON` | Entries are valid JSON |
| `TestLog_TimestampFormat` | Timestamp matches expected format |
| `TestLog_OmitsEmptyFields` | Empty optional fields are omitted |
| `TestLogWithUser_PopulatesUserFields` | User fields are populated from config |

#### Integration Tests

Create `test/integration/audit/audit_test.go`:

| Test Case | Description |
|-----------|-------------|
| `TestAudit_EncryptLogsOperation` | Encrypt command logs to audit |
| `TestAudit_DecryptLogsOperation` | Decrypt command logs to audit |
| `TestAudit_RegisterLogsOperation` | Register command logs with target user |
| `TestAudit_RevokeLogsOperation` | Revoke command logs with target user |
| `TestAudit_InitLogsOperation` | Init command logs with project info |
| `TestAudit_AllOperationsLogged` | Verify all command types log correctly |

### Definition of Done

- [ ] `internal/audit/audit.go` implemented
- [ ] `internal/audit/audit_test.go` with unit tests
- [ ] All secrets commands log their operations
- [ ] Integration tests pass
- [ ] `golangci-lint run` passes
- [ ] `go test -v ./...` passes

---

## KAN-014: Log Command

### Summary

Implement `kanuka secrets log` command to view the audit log with filtering and formatting options, following git's familiar interface patterns.

### Priority

**High** - Natural companion to KAN-013

### Dependencies

- **KAN-013** (Audit Log) must be completed first

### Context

Once an audit log exists, users need a way to view it. The log command should feel familiar to git users, with similar flags for limiting output, filtering, and formatting.

### Current Behavior

Command does not exist.

### New Behavior

```bash
# View full log (chronological order, oldest first)
$ kanuka secrets log
2024-01-15 10:30:00  alice@example.com    encrypt    .env, .env.local
2024-01-15 10:35:00  bob@example.com      register   charlie@example.com
2024-01-15 11:00:00  alice@example.com    revoke     charlie@example.com
2024-01-15 11:30:00  alice@example.com    sync       3 users, 5 files

# Limit number of entries (most recent)
$ kanuka secrets log -n 5

# Reverse order (most recent first, like git log)
$ kanuka secrets log --reverse

# Filter by user
$ kanuka secrets log --user alice@example.com

# Filter by operation type
$ kanuka secrets log --operation encrypt
$ kanuka secrets log --operation register,revoke

# Filter by date range
$ kanuka secrets log --since 2024-01-01
$ kanuka secrets log --until 2024-01-31
$ kanuka secrets log --since 2024-01-01 --until 2024-01-31

# Combine filters
$ kanuka secrets log --user alice@example.com --operation encrypt -n 10

# Compact one-line format
$ kanuka secrets log --oneline
2024-01-15 alice@example.com encrypt 2 files
2024-01-15 bob@example.com register charlie@example.com
2024-01-15 alice@example.com revoke charlie@example.com

# JSON output for scripting
$ kanuka secrets log --json
[
  {"ts":"2024-01-15T10:30:00.123456Z","user":"alice@example.com","op":"encrypt","files":[".env"]},
  {"ts":"2024-01-15T10:35:00.456789Z","user":"bob@example.com","op":"register","target_user":"charlie@example.com"}
]

# Empty log
$ kanuka secrets log
No audit log entries found.

# Log file doesn't exist
$ kanuka secrets log
No audit log found. Operations will be logged after running any secrets command.
```

### Technical Details

#### Files to Create

| File | Purpose |
|------|---------|
| `cmd/secrets_log.go` | Command implementation |

#### Command Structure

```go
package cmd

import (
    "encoding/json"
    "fmt"
    "os"
    "strings"
    "time"

    "github.com/spf13/cobra"
)

var (
    logLimit     int
    logReverse   bool
    logUser      string
    logOperation string
    logSince     string
    logUntil     string
    logOneline   bool
    logJSON      bool
)

func init() {
    logCmd.Flags().IntVarP(&logLimit, "number", "n", 0, "limit number of entries shown")
    logCmd.Flags().BoolVar(&logReverse, "reverse", false, "show most recent entries first")
    logCmd.Flags().StringVar(&logUser, "user", "", "filter by user email")
    logCmd.Flags().StringVar(&logOperation, "operation", "", "filter by operation type (comma-separated)")
    logCmd.Flags().StringVar(&logSince, "since", "", "show entries after date (YYYY-MM-DD)")
    logCmd.Flags().StringVar(&logUntil, "until", "", "show entries before date (YYYY-MM-DD)")
    logCmd.Flags().BoolVar(&logOneline, "oneline", false, "compact one-line format")
    logCmd.Flags().BoolVar(&logJSON, "json", false, "output as JSON array")
    
    SecretsCmd.AddCommand(logCmd)
}

var logCmd = &cobra.Command{
    Use:   "log",
    Short: "View the audit log",
    Long: `Displays the audit log of secrets operations.

Shows who performed what operation and when. Use filters to narrow down
the results.

Examples:
  kanuka secrets log                              # View full log
  kanuka secrets log -n 10                        # Last 10 entries
  kanuka secrets log --reverse                    # Most recent first
  kanuka secrets log --user alice@example.com     # Filter by user
  kanuka secrets log --operation encrypt,decrypt  # Filter by operation
  kanuka secrets log --since 2024-01-01           # Filter by date
  kanuka secrets log --json                       # JSON output`,
    RunE: runLog,
}

func runLog(cmd *cobra.Command, args []string) error {
    // Implementation
}
```

#### Implementation Steps

1. **Read log file:**
   ```go
   logPath := filepath.Join(configs.ProjectKanukaSettings.ProjectPath, ".kanuka", "audit.jsonl")
   
   data, err := os.ReadFile(logPath)
   if os.IsNotExist(err) {
       fmt.Println("No audit log found. Operations will be logged after running any secrets command.")
       return nil
   }
   if err != nil {
       return fmt.Errorf("failed to read audit log: %w", err)
   }
   ```

2. **Parse entries:**
   ```go
   var entries []audit.Entry
   lines := strings.Split(strings.TrimSpace(string(data)), "\n")
   for _, line := range lines {
       if line == "" {
           continue
       }
       var entry audit.Entry
       if err := json.Unmarshal([]byte(line), &entry); err != nil {
           // Skip malformed entries, log warning
           continue
       }
       entries = append(entries, entry)
   }
   ```

3. **Apply filters:**
   ```go
   filtered := entries
   
   if logUser != "" {
       filtered = filterByUser(filtered, logUser)
   }
   
   if logOperation != "" {
       ops := strings.Split(logOperation, ",")
       filtered = filterByOperations(filtered, ops)
   }
   
   if logSince != "" {
       sinceTime, _ := time.Parse("2006-01-02", logSince)
       filtered = filterSince(filtered, sinceTime)
   }
   
   if logUntil != "" {
       untilTime, _ := time.Parse("2006-01-02", logUntil)
       filtered = filterUntil(filtered, untilTime)
   }
   ```

4. **Apply ordering and limit:**
   ```go
   if logReverse {
       // Reverse the slice
       for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
           filtered[i], filtered[j] = filtered[j], filtered[i]
       }
   }
   
   if logLimit > 0 && len(filtered) > logLimit {
       if logReverse {
           filtered = filtered[:logLimit]
       } else {
           filtered = filtered[len(filtered)-logLimit:]
       }
   }
   ```

5. **Output:**
   ```go
   if len(filtered) == 0 {
       fmt.Println("No audit log entries found.")
       return nil
   }
   
   if logJSON {
       return outputJSON(filtered)
   }
   
   if logOneline {
       return outputOneline(filtered)
   }
   
   return outputDefault(filtered)
   ```

#### Output Formats

**Default format:**
```
2024-01-15 10:30:00  alice@example.com    encrypt    .env, .env.local
2024-01-15 10:35:00  bob@example.com      register   charlie@example.com
```

Format: `YYYY-MM-DD HH:MM:SS  <user padded to 25 chars>  <op padded to 10 chars>  <details>`

**Oneline format:**
```
2024-01-15 alice@example.com encrypt 2 files
2024-01-15 bob@example.com register charlie@example.com
```

Format: `YYYY-MM-DD <user> <op> <brief details>`

**JSON format:**
```json
[
  {"ts":"2024-01-15T10:30:00.123456Z","user":"alice@example.com","op":"encrypt","files":[".env"]}
]
```

Full entries as JSON array.

#### Details Rendering

| Operation | Details Format |
|-----------|----------------|
| `encrypt` | File list or "N files" if > 3 |
| `decrypt` | File list or "N files" if > 3 |
| `register` | Target user email |
| `revoke` | Target user email (+ device if specific) |
| `sync` | "N users, M files" |
| `rotate` | (empty) |
| `clean` | "removed N entries" |
| `import` | Mode + "N files" |
| `export` | Output path |
| `init` | Project name |
| `create` | Device name |

### Acceptance Criteria

- [ ] `kanuka secrets log` command implemented
- [ ] Default output shows entries in chronological order
- [ ] `-n` flag limits number of entries
- [ ] `--reverse` flag shows most recent first
- [ ] `--user` filter works
- [ ] `--operation` filter works (supports comma-separated list)
- [ ] `--since` and `--until` date filters work
- [ ] Filters can be combined
- [ ] `--oneline` format works
- [ ] `--json` format outputs valid JSON array
- [ ] Graceful handling when log file doesn't exist
- [ ] Graceful handling when log is empty
- [ ] Help text is clear with examples

### Testing Requirements

#### Integration Tests

Create `test/integration/log/log_test.go`:

| Test Case | Description |
|-----------|-------------|
| `TestLog_NoLogFile` | Shows appropriate message |
| `TestLog_EmptyLog` | Shows appropriate message |
| `TestLog_BasicOutput` | Shows entries in correct format |
| `TestLog_LimitFlag` | `-n` limits entries correctly |
| `TestLog_ReverseFlag` | `--reverse` reverses order |
| `TestLog_UserFilter` | `--user` filters correctly |
| `TestLog_OperationFilter` | `--operation` filters correctly |
| `TestLog_MultipleOperationFilter` | Comma-separated operations work |
| `TestLog_SinceFilter` | `--since` filters correctly |
| `TestLog_UntilFilter` | `--until` filters correctly |
| `TestLog_CombinedFilters` | Multiple filters work together |
| `TestLog_OnelineFormat` | `--oneline` format correct |
| `TestLog_JsonFormat` | `--json` output is valid JSON |
| `TestLog_InvalidDateFormat` | Shows helpful error for bad dates |

### Definition of Done

- [ ] `cmd/secrets_log.go` implemented
- [ ] All filter flags work correctly
- [ ] All output formats correct
- [ ] Integration tests pass
- [ ] `golangci-lint run` passes
- [ ] Help text with examples
- [ ] Command added to `SecretsCmd`

---

## KAN-015: Selective File Encryption

### Summary

Allow users to specify which files to encrypt/decrypt using positional arguments and glob patterns, while keeping the default behavior of processing all files in the current directory and children.

### Priority

**Medium** - Flexibility improvement

### Context

Currently, `kanuka secrets encrypt` and `decrypt` process all matching files with no way to be selective. This is limiting for:

- **Large monorepos** - Only want to encrypt specific services
- **Gradual adoption** - Encrypt production secrets first, then others
- **CI/CD pipelines** - Only decrypt the specific files needed for a job
- **Debugging** - Re-encrypt just one file that was modified

### Current Behavior

```bash
# Encrypts ALL .env* files from project root recursively - no choice
$ kanuka secrets encrypt
Encrypting 15 files...

# Same for decrypt
$ kanuka secrets decrypt
Decrypting 15 files...
```

### New Behavior

```bash
# Default: unchanged - encrypt all .env* files in current directory and children
$ kanuka secrets encrypt

# Encrypt specific file
$ kanuka secrets encrypt .env

# Encrypt multiple specific files
$ kanuka secrets encrypt .env .env.local config/.env.production

# Encrypt with glob pattern (quote to prevent shell expansion)
$ kanuka secrets encrypt "services/*/.env"
$ kanuka secrets encrypt "**/.env.production"

# Encrypt all files in a specific directory
$ kanuka secrets encrypt services/api/

# Same patterns work for decrypt
$ kanuka secrets decrypt .env.kanuka
$ kanuka secrets decrypt "services/*/.env.kanuka"
$ kanuka secrets decrypt services/api/

# Combine with dry-run to preview
$ kanuka secrets encrypt .env --dry-run
[dry-run] Would encrypt:
  .env -> .env.kanuka
```

### Design Decisions

#### Positional Arguments, Not Config

Use positional arguments rather than a config file:

**Why:**
- No new state to track
- Explicit is better than implicit
- Users can see exactly what will happen
- No "which files are managed?" confusion

#### Default Behavior Unchanged

When no arguments provided, process all files. This maintains backward compatibility and is the right default for most users.

#### Glob Support

Support shell-style globs:
- `*` - matches any characters except `/`
- `**` - matches any characters including `/` (recursive)
- `?` - matches single character

Note: Users should quote patterns to prevent shell expansion (`"*/.env"` not `*/.env`).

#### Directory Support

If a directory is specified, process all matching files within it recursively.

### Technical Details

#### Files to Modify

| File | Changes |
|------|---------|
| `cmd/secrets_encrypt.go` | Accept positional args, call file resolver |
| `cmd/secrets_decrypt.go` | Accept positional args, call file resolver |
| `internal/secrets/files.go` | Add `ResolveFiles()` function |

#### Command Changes

```go
// cmd/secrets_encrypt.go

var encryptCmd = &cobra.Command{
    Use:   "encrypt [files...]",
    Short: "Encrypts .env files into .kanuka files",
    Long: `Encrypts environment files using your Kānuka key.

If no files are specified, encrypts all .env files in the current
directory and subdirectories.

You can specify individual files, directories, or glob patterns:

  kanuka secrets encrypt                      # All .env files
  kanuka secrets encrypt .env                 # Single file
  kanuka secrets encrypt .env .env.local      # Multiple files
  kanuka secrets encrypt "services/*/.env"    # Glob pattern
  kanuka secrets encrypt services/api/        # Directory`,
    RunE: runEncrypt,
}
```

```go
// cmd/secrets_decrypt.go

var decryptCmd = &cobra.Command{
    Use:   "decrypt [files...]",
    Short: "Decrypts .kanuka files into .env files",
    Long: `Decrypts encrypted files using your Kānuka key.

If no files are specified, decrypts all .kanuka files in the current
directory and subdirectories.

You can specify individual files, directories, or glob patterns:

  kanuka secrets decrypt                          # All .kanuka files
  kanuka secrets decrypt .env.kanuka              # Single file
  kanuka secrets decrypt "services/*/.env.kanuka" # Glob pattern
  kanuka secrets decrypt services/api/            # Directory`,
    RunE: runDecrypt,
}
```

#### File Resolution Logic

```go
// internal/secrets/files.go

import (
    "os"
    "path/filepath"
    "strings"
)

// ResolveFiles takes user-provided paths/globs and returns matching files.
// If patterns is empty, returns all matching files (default behavior).
// forEncryption=true finds .env* files, forEncryption=false finds *.kanuka files.
func ResolveFiles(patterns []string, forEncryption bool) ([]string, error) {
    if len(patterns) == 0 {
        // Default: find all matching files recursively from current dir
        return FindAllSecretFiles(forEncryption)
    }
    
    var files []string
    seen := make(map[string]bool) // Deduplicate
    
    for _, pattern := range patterns {
        resolved, err := resolvePattern(pattern, forEncryption)
        if err != nil {
            return nil, err
        }
        
        for _, f := range resolved {
            if !seen[f] {
                seen[f] = true
                files = append(files, f)
            }
        }
    }
    
    if len(files) == 0 {
        return nil, fmt.Errorf("no matching files found")
    }
    
    return files, nil
}

func resolvePattern(pattern string, forEncryption bool) ([]string, error) {
    // Check if it's a directory
    info, err := os.Stat(pattern)
    if err == nil && info.IsDir() {
        return findFilesInDir(pattern, forEncryption)
    }
    
    // Check if it contains glob characters
    if strings.ContainsAny(pattern, "*?[") {
        return expandGlob(pattern, forEncryption)
    }
    
    // Treat as literal file path
    if _, err := os.Stat(pattern); os.IsNotExist(err) {
        return nil, fmt.Errorf("file not found: %s", pattern)
    }
    
    return []string{pattern}, nil
}

func expandGlob(pattern string, forEncryption bool) ([]string, error) {
    // Use doublestar for ** support
    matches, err := doublestar.FilepathGlob(pattern)
    if err != nil {
        return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
    }
    
    // Filter to only include appropriate file types
    var filtered []string
    for _, m := range matches {
        if forEncryption && isEnvFile(m) {
            filtered = append(filtered, m)
        } else if !forEncryption && isKanukaFile(m) {
            filtered = append(filtered, m)
        }
    }
    
    return filtered, nil
}

func findFilesInDir(dir string, forEncryption bool) ([]string, error) {
    var files []string
    
    err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if d.IsDir() {
            // Skip .kanuka directory
            if d.Name() == ".kanuka" {
                return filepath.SkipDir
            }
            return nil
        }
        
        if forEncryption && isEnvFile(path) {
            files = append(files, path)
        } else if !forEncryption && isKanukaFile(path) {
            files = append(files, path)
        }
        
        return nil
    })
    
    return files, err
}

func isEnvFile(path string) bool {
    base := filepath.Base(path)
    return strings.HasPrefix(base, ".env") && !strings.HasSuffix(base, ".kanuka")
}

func isKanukaFile(path string) bool {
    return strings.HasSuffix(path, ".kanuka") && !strings.Contains(path, ".kanuka/")
}
```

#### Dependency

Add `github.com/bmatcuk/doublestar/v4` for `**` glob support:

```bash
go get github.com/bmatcuk/doublestar/v4
```

#### Integration with Commands

```go
// cmd/secrets_encrypt.go

func runEncrypt(cmd *cobra.Command, args []string) error {
    // ... existing setup ...
    
    // Resolve files from arguments
    files, err := secrets.ResolveFiles(args, true /* forEncryption */)
    if err != nil {
        return err
    }
    
    Logger.Infof("Found %d files to encrypt", len(files))
    
    // ... rest of encryption logic using 'files' ...
}
```

### Acceptance Criteria

- [x] `encrypt` command accepts positional file arguments
- [x] `decrypt` command accepts positional file arguments
- [x] No arguments = default behavior (all files)
- [x] Single file argument works
- [x] Multiple file arguments work
- [x] Glob patterns work (including `**`)
- [x] Directory arguments work (recursive)
- [x] Patterns are deduplicated (no file processed twice)
- [x] Non-existent files show clear error
- [x] Invalid glob patterns show clear error
- [x] `--dry-run` works with specific files
- [x] Help text updated with examples
- [x] `.kanuka/` directory contents are never included

### Testing Requirements

#### Unit Tests

Created `internal/secrets/files_test.go` with tests for:

| Test Case | Description |
|-----------|-------------|
| `TestResolveFiles_EmptyPatterns` | Empty patterns return nil |
| `TestResolveFiles_SingleFile` | Single file works |
| `TestResolveFiles_MultipleFiles` | Multiple files work |
| `TestResolveFiles_Directory` | Directory resolves files within |
| `TestResolveFiles_GlobPattern` | Glob patterns expand correctly |
| `TestResolveFiles_DoubleStarGlob` | `**` pattern works recursively |
| `TestResolveFiles_NonExistentFile` | Shows error for non-existent |
| `TestResolveFiles_Deduplication` | Files are deduplicated |
| `TestResolveFiles_ExcludesKanukaDir` | Never processes .kanuka/ contents |
| `TestResolveFiles_ForDecryption` | Works for .kanuka files |
| `TestResolveFiles_WrongFileType` | Rejects wrong file types |
| `TestIsEnvFile` | Helper function tests |
| `TestIsKanukaFile` | Helper function tests |
| `TestIsInKanukaDir` | Helper function tests |

#### Integration Tests

Create `test/integration/encrypt/encrypt_selective_test.go`:

| Test Case | Description |
|-----------|-------------|
| `TestEncrypt_DefaultAllFiles` | No args encrypts all .env files |
| `TestEncrypt_SingleFile` | Single file argument works |
| `TestEncrypt_MultipleFiles` | Multiple file arguments work |
| `TestEncrypt_GlobPattern` | Glob patterns expand correctly |
| `TestEncrypt_DoubleStarGlob` | `**` pattern works recursively |
| `TestEncrypt_Directory` | Directory argument processes contents |
| `TestEncrypt_NonExistentFile` | Shows clear error |
| `TestEncrypt_InvalidGlob` | Shows clear error for malformed glob |
| `TestEncrypt_Deduplication` | Same file via different patterns only processed once |
| `TestEncrypt_DryRunWithFiles` | Dry run shows correct files |
| `TestEncrypt_ExcludesKanukaDir` | Never processes .kanuka/ contents |

Create `test/integration/decrypt/decrypt_selective_test.go`:

| Test Case | Description |
|-----------|-------------|
| `TestDecrypt_SingleFile` | Single file argument works |
| `TestDecrypt_GlobPattern` | Glob pattern works for .kanuka files |
| `TestDecrypt_Directory` | Directory processes .kanuka files within |

### Definition of Done

- [x] `encrypt` command accepts file arguments
- [x] `decrypt` command accepts file arguments
- [x] `ResolveFiles()` function implemented
- [x] Glob support with `**` works
- [x] Unit tests pass
- [x] `golangci-lint run` passes
- [x] Help text updated
- [x] `doublestar` dependency added to go.mod

---

## KAN-016: Update Init Command for Monorepo Guidance

### Summary

Update the `kanuka secrets init` success message to inform users about their options when working in a monorepo, now that selective encryption is available.

### Priority

**Low** - UX improvement

### Dependencies

- **KAN-015** (Selective File Encryption) should be completed first

### Context

With selective file encryption (KAN-015), monorepo users have two viable workflows:

1. **Single `.kanuka` at root** - One secrets store, selective encrypt/decrypt
2. **Separate `.kanuka` per service** - Run `init` in each service directory

The init command should inform users of these options so they can make an informed decision.

### Current Behavior

```bash
$ kanuka secrets init
✓ Kānuka initialized successfully!
→ Run kanuka secrets encrypt to encrypt your existing .env files
```

### New Behavior

```bash
$ kanuka secrets init
✓ Kānuka initialized successfully!

→ Run kanuka secrets encrypt to encrypt your existing .env files

Tip: Working in a monorepo? You have two options:
  1. Keep this single .kanuka at the root and use selective encryption:
     kanuka secrets encrypt services/api/.env
  2. Initialize separate .kanuka stores in each service:
     cd services/api && kanuka secrets init
```

### Technical Details

#### Files to Modify

| File | Changes |
|------|---------|
| `cmd/secrets_init.go` | Update final success message |

#### Implementation

```go
// cmd/secrets_init.go

// Update the finalMessage around line 232

finalMessage := color.GreenString("✓") + " Kānuka initialized successfully!\n\n" +
    color.CyanString("→") + " Run " + color.YellowString("kanuka secrets encrypt") + " to encrypt your existing .env files\n\n" +
    color.CyanString("Tip:") + " Working in a monorepo? You have two options:\n" +
    "  1. Keep this single .kanuka at the root and use selective encryption:\n" +
    "     " + color.YellowString("kanuka secrets encrypt services/api/.env") + "\n" +
    "  2. Initialize separate .kanuka stores in each service:\n" +
    "     " + color.YellowString("cd services/api && kanuka secrets init")

spinner.FinalMSG = finalMessage
```

#### Optional: Detect Monorepo

Could optionally detect if this looks like a monorepo and only show the tip then:

```go
func looksLikeMonorepo() bool {
    // Check for common monorepo indicators
    indicators := []string{
        "packages",
        "services", 
        "apps",
        "libs",
        "pnpm-workspace.yaml",
        "lerna.json",
        "nx.json",
        "turbo.json",
    }
    
    for _, indicator := range indicators {
        if _, err := os.Stat(indicator); err == nil {
            return true
        }
    }
    return false
}
```

However, this adds complexity. Recommendation: Always show the tip. It's useful information that doesn't hurt non-monorepo users.

### Acceptance Criteria

- [ ] Init success message updated with monorepo guidance
- [ ] Message includes both monorepo workflow options
- [ ] Examples use `color.YellowString` for commands (consistent with existing style)
- [ ] Message is not overly long or cluttered

### Testing Requirements

#### Integration Tests

Update `test/integration/init/init_test.go`:

| Test Case | Description |
|-----------|-------------|
| `TestInit_SuccessMessageIncludesMonorepoTip` | Verify monorepo guidance is in output |

### Definition of Done

- [ ] `cmd/secrets_init.go` updated
- [ ] Success message includes monorepo options
- [ ] Integration tests pass
- [ ] `golangci-lint run` passes

---

## KAN-017: Documentation Updates

### Summary

Update all documentation to cover the new features: audit log, log command, selective encryption, and monorepo workflows.

### Priority

**Medium** - Should be done after features are implemented

### Dependencies

- **KAN-013** (Audit Log)
- **KAN-014** (Log Command)  
- **KAN-015** (Selective File Encryption)
- **KAN-016** (Init Monorepo Guidance)

All should be completed before this ticket.

### Context

Documentation needs to be updated to:

1. Remove the non-existent `KANUKA_DATA_DIR` (part of KAN-012)
2. Document the audit log feature
3. Document the `log` command
4. Document selective file encryption
5. Add monorepo workflow guidance
6. Update command reference

### Documentation Changes

#### 1. Remove KANUKA_DATA_DIR

**File:** `docs/src/content/docs/configuration/configuration.mdx`

Remove the `KANUKA_DATA_DIR` section entirely.

#### 2. New Guide: Audit Log

**File:** `docs/src/content/docs/guides/audit-log.md` (create)

```markdown
---
title: Audit Log
description: Understanding the audit log and viewing operation history.
---

Kanuka maintains an audit log of all secrets operations, providing visibility
into who did what and when.

## What gets logged

Every secrets operation is recorded:
- Encrypt and decrypt operations
- User registration and revocation
- Key rotation (sync and rotate)
- Cleanup operations
- Import and export

## Log location

The audit log is stored at `.kanuka/audit.jsonl` and is committed to your
repository alongside other Kanuka files.

## Viewing the log

Use the log command to view operation history:

\`\`\`bash
kanuka secrets log
\`\`\`

See the [log command guide](/guides/log/) for filtering and formatting options.

## Log format

The log uses JSON Lines format (one JSON object per line):

\`\`\`json
{"ts":"2024-01-15T10:30:00.123456Z","user":"alice@example.com","op":"encrypt","files":[".env"]}
\`\`\`

## Privacy considerations

The audit log contains:
- Timestamps of operations
- Email addresses of users
- File paths that were encrypted/decrypted
- User emails for register/revoke operations

It does NOT contain:
- Secret values
- Private keys
- Encryption keys
```

#### 3. New Guide: Log Command

**File:** `docs/src/content/docs/guides/log.md` (create)

```markdown
---
title: Viewing Operation History
description: Using the log command to view and filter the audit log.
---

The log command displays the history of secrets operations.

## Basic usage

\`\`\`bash
kanuka secrets log
\`\`\`

## Filtering

### By user

\`\`\`bash
kanuka secrets log --user alice@example.com
\`\`\`

### By operation

\`\`\`bash
kanuka secrets log --operation encrypt
kanuka secrets log --operation register,revoke
\`\`\`

### By date

\`\`\`bash
kanuka secrets log --since 2024-01-01
kanuka secrets log --until 2024-01-31
\`\`\`

## Limiting output

\`\`\`bash
# Last 10 entries
kanuka secrets log -n 10

# Most recent first
kanuka secrets log --reverse
\`\`\`

## Output formats

### Compact format

\`\`\`bash
kanuka secrets log --oneline
\`\`\`

### JSON output

\`\`\`bash
kanuka secrets log --json
\`\`\`
```

#### 4. Update Encryption Guide

**File:** `docs/src/content/docs/guides/encryption.md`

Add section on selective encryption:

```markdown
## Encrypting specific files

By default, `encrypt` processes all `.env` files. You can specify files:

\`\`\`bash
# Single file
kanuka secrets encrypt .env

# Multiple files
kanuka secrets encrypt .env .env.local

# Glob pattern
kanuka secrets encrypt "services/*/.env"

# Directory
kanuka secrets encrypt services/api/
\`\`\`
```

#### 5. Update Decryption Guide

**File:** `docs/src/content/docs/guides/decryption.md`

Add section on selective decryption:

```markdown
## Decrypting specific files

By default, `decrypt` processes all `.kanuka` files. You can specify files:

\`\`\`bash
# Single file
kanuka secrets decrypt .env.kanuka

# Glob pattern
kanuka secrets decrypt "services/*/.env.kanuka"

# Directory
kanuka secrets decrypt services/api/
\`\`\`
```

#### 6. New Guide: Monorepo Workflows

**File:** `docs/src/content/docs/guides/monorepo.md` (create)

```markdown
---
title: Working with Monorepos
description: Strategies for managing secrets in monorepo projects.
---

Kanuka supports two approaches for monorepos.

## Option 1: Single secrets store at root

Initialize once at the monorepo root:

\`\`\`bash
cd my-monorepo
kanuka secrets init
\`\`\`

Use selective encryption to manage specific services:

\`\`\`bash
# Encrypt only the API service
kanuka secrets encrypt services/api/.env

# Encrypt multiple services
kanuka secrets encrypt "services/*/.env"

# Decrypt just what you need
kanuka secrets decrypt services/api/.env.kanuka
\`\`\`

**Pros:**
- Single source of truth for access control
- One set of keys to manage
- Simpler team onboarding

**Cons:**
- All registered users can decrypt all secrets
- No per-service access control

## Option 2: Separate secrets stores per service

Initialize in each service that needs secrets:

\`\`\`bash
cd my-monorepo/services/api
kanuka secrets init

cd ../admin
kanuka secrets init
\`\`\`

**Pros:**
- Different teams can have access to different services
- Isolated key rotation per service

**Cons:**
- More key management overhead
- Team members may need access to multiple stores
- Must remember to `cd` to correct directory

## Recommendation

Start with Option 1 (single store) unless you have a specific need for 
per-service access control. You can always migrate to Option 2 later.
```

#### 7. Update Command Reference

**File:** `docs/src/content/docs/reference/references.md`

Add `log` command:

```markdown
### `kanuka secrets log`

Displays the audit log of secrets operations.

\`\`\`
Usage:
  kanuka secrets log [flags]

Flags:
  -n, --number int        limit number of entries shown
      --reverse           show most recent entries first
      --user string       filter by user email
      --operation string  filter by operation type (comma-separated)
      --since string      show entries after date (YYYY-MM-DD)
      --until string      show entries before date (YYYY-MM-DD)
      --oneline           compact one-line format
      --json              output as JSON array
  -h, --help              help for log
\`\`\`
```

Update `encrypt` and `decrypt` commands to show file arguments:

```markdown
### `kanuka secrets encrypt`

\`\`\`
Usage:
  kanuka secrets encrypt [files...] [flags]
\`\`\`

### `kanuka secrets decrypt`

\`\`\`
Usage:
  kanuka secrets decrypt [files...] [flags]
\`\`\`
```

#### 8. Update FAQ

**File:** `docs/src/content/docs/reference/faq.md`

Add:

```markdown
## How do I see who has been accessing secrets?

Use the audit log:

\`\`\`bash
kanuka secrets log
\`\`\`

This shows all operations with timestamps and user emails.

## Can I encrypt just one file?

Yes, specify the file:

\`\`\`bash
kanuka secrets encrypt .env
\`\`\`

## How do I use Kanuka in a monorepo?

See the [monorepo guide](/guides/monorepo/) for detailed options.
```

#### 9. Update Sidebar

**File:** `docs/astro.config.mjs`

Add new guides to sidebar:

```javascript
{
  label: "Secrets Management",
  items: [
    // ... existing items ...
    "guides/audit-log",
    "guides/log",
    "guides/monorepo",
  ],
},
```

#### 10. Update README

**File:** `README.md`

Add to features:
```markdown
- **Audit Trail**: Track who performed what operations and when
```

Add to commands:
```markdown
- `kanuka secrets log`: View audit log of operations
```

### Files to Create

| File | Purpose |
|------|---------|
| `docs/src/content/docs/guides/audit-log.md` | Audit log overview |
| `docs/src/content/docs/guides/log.md` | Log command guide |
| `docs/src/content/docs/guides/monorepo.md` | Monorepo workflows |

### Files to Modify

| File | Changes |
|------|---------|
| `docs/src/content/docs/configuration/configuration.mdx` | Remove KANUKA_DATA_DIR |
| `docs/src/content/docs/guides/encryption.md` | Add selective encryption |
| `docs/src/content/docs/guides/decryption.md` | Add selective decryption |
| `docs/src/content/docs/reference/references.md` | Add log command, update encrypt/decrypt |
| `docs/src/content/docs/reference/faq.md` | Add new FAQs |
| `docs/astro.config.mjs` | Add new guides to sidebar |
| `README.md` | Add audit trail feature, log command |

### Acceptance Criteria

- [ ] KANUKA_DATA_DIR removed from configuration docs
- [ ] Audit log guide created
- [ ] Log command guide created
- [ ] Monorepo guide created
- [ ] Encryption guide updated with selective encryption
- [ ] Decryption guide updated with selective decryption
- [ ] Command reference updated
- [ ] FAQ updated
- [ ] Sidebar updated
- [ ] README updated
- [ ] Docs build without errors

### Testing Requirements

```bash
cd docs
npm run build
```

Verify no build errors and spot-check the generated pages.

### Definition of Done

- [ ] All documentation files created/updated
- [ ] Sidebar includes new pages
- [ ] Docs build successfully
- [ ] No broken internal links
- [ ] README reflects new features

---

## Future Considerations (Out of Scope)

These items have been discussed but are explicitly deferred:

### Monorepo Path-Based Access Control

Allow different users to access different paths:

```toml
[access."services/api/*"]
users = ["alice@example.com"]
```

Deferred until there's demonstrated demand.

### Audit Log Signing

Digital signatures on audit entries for tamper-evidence. Deferred as the threat model doesn't justify the complexity.

### Custom Config/Data Paths

`KANUKA_CONFIG_DIR` or `KANUKA_DATA_DIR` environment variables. Can be added trivially if users request it.
