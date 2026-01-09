# New Features Plan

This document outlines the next set of features planned for Kanuka.

---

## Table of Contents

1. [KAN-012: Remove Undocumented KANUKA_DATA_DIR](#kan-012-remove-undocumented-kanuka_data_dir)
2. [KAN-013: Audit Log](#kan-013-audit-log)
3. [KAN-014: Log Command](#kan-014-log-command)
4. [KAN-015: Selective File Encryption](#kan-015-selective-file-encryption)

---

## Implementation Order

```
KAN-012 (Remove KANUKA_DATA_DIR)  ─── Quick docs cleanup, do first

KAN-013 (Audit Log)               ─── Core feature
     │
     └── KAN-014 (Log Command)    ─── Depends on KAN-013

KAN-015 (Selective Encryption)    ─── Independent, can be done anytime
```

---

## KAN-012: Remove Undocumented KANUKA_DATA_DIR

### Summary

Remove the `KANUKA_DATA_DIR` environment variable from documentation. It was documented but never implemented, and there's no demonstrated need for it.

### Priority

**Low** - Quick documentation cleanup

### Context

The configuration documentation at `docs/src/content/docs/configuration/configuration.mdx` advertises a `KANUKA_DATA_DIR` environment variable that doesn't actually exist in the codebase. This is configuration for the sake of configuration.

Users who need custom paths can use symlinks. If real demand emerges, we can add this later as a simple enhancement.

### Acceptance Criteria

- [ ] Remove `KANUKA_DATA_DIR` section from `docs/src/content/docs/configuration/configuration.mdx`
- [ ] Verify no other docs reference this variable

### Definition of Done

- [ ] Documentation updated
- [ ] No dead references to `KANUKA_DATA_DIR`

---

## KAN-013: Audit Log

### Summary

Implement an append-only audit log that records who performed what operation and when, providing a paper trail for team visibility.

### Priority

**High** - Valuable for teams using Kanuka in professional contexts

### Context

Teams need visibility into secrets management activity. An audit log answers questions like:
- Who encrypted the secrets last?
- When was this user revoked?
- Who registered that new team member?

This is table stakes for security tooling in a team environment.

### Design Decisions

#### Log Format

Use JSON Lines (`.jsonl`) format - one JSON object per line:

```jsonl
{"ts":"2024-01-15T10:30:00.123456Z","user":"alice@example.com","uuid":"a1b2c3...","op":"encrypt","files":[".env",".env.local"]}
{"ts":"2024-01-15T10:35:00.456789Z","user":"bob@example.com","uuid":"b2c3d4...","op":"register","target_user":"charlie@example.com","target_uuid":"c3d4e5..."}
{"ts":"2024-01-15T11:00:00.789012Z","user":"alice@example.com","uuid":"a1b2c3...","op":"revoke","target_user":"charlie@example.com","target_uuid":"c3d4e5..."}
```

**Why JSON Lines:**
- Easy to parse programmatically
- Easy to append (no array wrapper)
- Easy to grep/filter with standard tools
- Human-readable enough for debugging

#### Log Location

Store at `.kanuka/audit.jsonl` - committed to the repository alongside other Kanuka files.

#### Conflict Handling

Accept git merge conflicts. Rationale:
- Conflicts are rare (requires simultaneous operations)
- When they occur, resolution is trivial (keep both lines - it's append-only)
- The audit log is not a security control (malicious actors can delete it)
- Complexity of conflict avoidance isn't worth it

Use microsecond-precision timestamps to reduce (but not eliminate) collision likelihood.

#### No Digital Signatures

Skip signing log entries for v1:
- The log is protected by git's commit history (and commit signatures if the team uses them)
- Signing adds complexity around key rotation and compromised keys
- The threat model doesn't justify the complexity
- Can be added later as a non-breaking enhancement if needed

### Operations to Log

| Operation | Fields |
|-----------|--------|
| `init` | project_name, project_uuid |
| `create` | device_name |
| `encrypt` | files[] |
| `decrypt` | files[] |
| `register` | target_user, target_uuid |
| `revoke` | target_user, target_uuid, device (if specific) |
| `sync` | users_count, files_count |
| `rotate` | (no additional fields) |
| `clean` | removed_count |
| `import` | mode (merge/replace), files_count |
| `export` | output_path |

### Log Entry Structure

```go
type AuditEntry struct {
    Timestamp   string   `json:"ts"`        // RFC3339 with microseconds
    User        string   `json:"user"`      // Email of user performing action
    UserUUID    string   `json:"uuid"`      // UUID of user performing action
    Operation   string   `json:"op"`        // Operation name
    Files       []string `json:"files,omitempty"`       // For encrypt/decrypt
    TargetUser  string   `json:"target_user,omitempty"` // For register/revoke
    TargetUUID  string   `json:"target_uuid,omitempty"` // For register/revoke
    Device      string   `json:"device,omitempty"`      // For device-specific revoke
    Count       int      `json:"count,omitempty"`       // For sync/clean/import
    Mode        string   `json:"mode,omitempty"`        // For import (merge/replace)
    Output      string   `json:"output,omitempty"`      // For export
    ProjectName string   `json:"project,omitempty"`     // For init
    ProjectUUID string   `json:"project_uuid,omitempty"`// For init
}
```

### Technical Details

#### Files to Create

| File | Purpose |
|------|---------|
| `internal/audit/audit.go` | Core audit logging functionality |

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

#### Implementation

```go
// internal/audit/audit.go

package audit

import (
    "encoding/json"
    "os"
    "path/filepath"
    "time"
)

// Log appends an entry to the audit log.
func Log(entry AuditEntry) error {
    entry.Timestamp = time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
    
    logPath := filepath.Join(".kanuka", "audit.jsonl")
    
    f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer f.Close()
    
    data, err := json.Marshal(entry)
    if err != nil {
        return err
    }
    
    _, err = f.Write(append(data, '\n'))
    return err
}
```

### Acceptance Criteria

- [ ] Audit log created at `.kanuka/audit.jsonl`
- [ ] All secrets commands log their operations
- [ ] Log entries contain timestamp, user, operation, and relevant details
- [ ] Log is append-only (no modifications to existing entries)
- [ ] Logging failures are warnings, not errors (don't break operations if logging fails)

### Testing Requirements

| Test Case | Description |
|-----------|-------------|
| `TestAuditLog_CreatesFile` | First operation creates the log file |
| `TestAuditLog_AppendsEntries` | Multiple operations append to same file |
| `TestAuditLog_JsonFormat` | Entries are valid JSON |
| `TestAuditLog_AllOperations` | Each command type logs correctly |

### Definition of Done

- [ ] `internal/audit/audit.go` implemented
- [ ] All secrets commands log operations
- [ ] Integration tests pass
- [ ] `golangci-lint run` passes

---

## KAN-014: Log Command

### Summary

Implement `kanuka secrets log` command to view the audit log with filtering and formatting options.

### Priority

**High** - Natural companion to KAN-013

### Dependencies

- **KAN-013** (Audit Log) must be completed first

### New Behavior

```bash
# View full log (most recent last)
$ kanuka secrets log
2024-01-15 10:30:00  alice@example.com    encrypt    .env, .env.local
2024-01-15 10:35:00  bob@example.com      register   charlie@example.com
2024-01-15 11:00:00  alice@example.com    revoke     charlie@example.com

# Limit number of entries
$ kanuka secrets log -n 5

# Reverse order (most recent first)
$ kanuka secrets log --reverse

# Filter by user
$ kanuka secrets log --user alice@example.com

# Filter by operation
$ kanuka secrets log --operation encrypt

# Filter by date
$ kanuka secrets log --since 2024-01-01
$ kanuka secrets log --until 2024-01-31

# Combine filters
$ kanuka secrets log --user alice@example.com --operation encrypt --since 2024-01-01

# Compact one-line format
$ kanuka secrets log --oneline
a1b2c3 2024-01-15 alice@example.com encrypt
b2c3d4 2024-01-15 bob@example.com register charlie@example.com

# JSON output for scripting
$ kanuka secrets log --json
```

### Command Structure

```go
var logCmd = &cobra.Command{
    Use:   "log",
    Short: "View the audit log",
    Long: `Displays the audit log of secrets operations.

Shows who performed what operation and when. Use filters to narrow down
the results.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
    },
}

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
    logCmd.Flags().IntVarP(&logLimit, "number", "n", 0, "limit number of entries (0 = no limit)")
    logCmd.Flags().BoolVar(&logReverse, "reverse", false, "show most recent first")
    logCmd.Flags().StringVar(&logUser, "user", "", "filter by user email")
    logCmd.Flags().StringVar(&logOperation, "operation", "", "filter by operation type")
    logCmd.Flags().StringVar(&logSince, "since", "", "show entries after date (YYYY-MM-DD)")
    logCmd.Flags().StringVar(&logUntil, "until", "", "show entries before date (YYYY-MM-DD)")
    logCmd.Flags().BoolVar(&logOneline, "oneline", false, "compact one-line format")
    logCmd.Flags().BoolVar(&logJSON, "json", false, "output as JSON array")
}
```

### Output Formats

#### Default Format

```
2024-01-15 10:30:00  alice@example.com    encrypt    .env, .env.local
2024-01-15 10:35:00  bob@example.com      register   charlie@example.com
```

Columns: timestamp, user, operation, details (varies by operation)

#### Oneline Format

```
a1b2c3 2024-01-15 alice@example.com encrypt
b2c3d4 2024-01-15 bob@example.com register charlie@example.com
```

Short hash prefix, date only, minimal details.

#### JSON Format

```json
[
  {"ts":"2024-01-15T10:30:00.123456Z","user":"alice@example.com","op":"encrypt","files":[".env"]},
  {"ts":"2024-01-15T10:35:00.456789Z","user":"bob@example.com","op":"register","target_user":"charlie@example.com"}
]
```

### Acceptance Criteria

- [ ] `kanuka secrets log` command implemented
- [ ] `-n` flag limits output
- [ ] `--reverse` flag reverses order
- [ ] `--user` filter works
- [ ] `--operation` filter works
- [ ] `--since` and `--until` filters work
- [ ] `--oneline` format works
- [ ] `--json` format works
- [ ] Graceful handling when log file doesn't exist
- [ ] Help text is clear

### Testing Requirements

| Test Case | Description |
|-----------|-------------|
| `TestLog_EmptyLog` | No log file, shows appropriate message |
| `TestLog_BasicOutput` | Shows entries in correct format |
| `TestLog_LimitFlag` | `-n` limits entries correctly |
| `TestLog_ReverseFlag` | `--reverse` reverses order |
| `TestLog_UserFilter` | `--user` filters correctly |
| `TestLog_OperationFilter` | `--operation` filters correctly |
| `TestLog_DateFilters` | `--since` and `--until` work |
| `TestLog_CombinedFilters` | Multiple filters work together |
| `TestLog_OnelineFormat` | `--oneline` format correct |
| `TestLog_JsonFormat` | `--json` output is valid JSON |

### Definition of Done

- [ ] `cmd/secrets_log.go` implemented
- [ ] All filter flags work
- [ ] Output formats correct
- [ ] Integration tests pass
- [ ] `golangci-lint run` passes
- [ ] Documentation added

---

## KAN-015: Selective File Encryption

### Summary

Allow users to specify which files to encrypt/decrypt using positional arguments, while keeping the default behavior of processing all files in the current directory and children.

### Priority

**Medium** - Flexibility improvement

### Context

Currently, `kanuka secrets encrypt` and `decrypt` process all matching files with no way to be selective. This can be limiting for:
- Large monorepos where you only want to encrypt specific services
- Gradual adoption (encrypt production secrets first)
- CI/CD pipelines that only need specific files

### Current Behavior

```bash
# Encrypts ALL .env* files from project root recursively
kanuka secrets encrypt
```

### New Behavior

```bash
# Default: encrypt all .env* files in current directory and children (unchanged)
kanuka secrets encrypt

# Encrypt specific file
kanuka secrets encrypt .env

# Encrypt multiple specific files
kanuka secrets encrypt .env .env.local

# Encrypt with glob pattern
kanuka secrets encrypt "services/*/.env"

# Encrypt specific directory
kanuka secrets encrypt services/api/

# Decrypt specific file
kanuka secrets decrypt .env.kanuka

# Decrypt with glob
kanuka secrets decrypt "services/*/.env.kanuka"
```

### Design Decisions

#### Positional Arguments, Not Config

Use positional arguments rather than a config file listing managed files:
- No new state to track
- Explicit is better than implicit
- Easier to understand what will happen

#### Default Behavior Unchanged

When no arguments provided, process all files (current behavior). This maintains backward compatibility.

#### Glob Support

Support shell-style globs for flexibility:
- `*` matches any characters except `/`
- `**` matches any characters including `/`
- `?` matches single character

#### Directory Support

If a directory is specified, process all matching files within it recursively.

### Technical Details

#### Files to Modify

| File | Changes |
|------|---------|
| `cmd/secrets_encrypt.go` | Accept positional args, implement file filtering |
| `cmd/secrets_decrypt.go` | Accept positional args, implement file filtering |
| `internal/secrets/files.go` | Add glob/path filtering logic |

#### Command Changes

```go
var encryptCmd = &cobra.Command{
    Use:   "encrypt [files...]",
    Short: "Encrypts .env files into .kanuka files",
    Long: `Encrypts environment files using your Kānuka key.

If no files are specified, encrypts all .env files in the current
directory and subdirectories.

You can specify individual files, directories, or glob patterns:
  kanuka secrets encrypt .env
  kanuka secrets encrypt .env .env.local
  kanuka secrets encrypt "services/*/.env"
  kanuka secrets encrypt services/api/`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // If args provided, use them; otherwise, find all files
    },
}
```

#### File Resolution Logic

```go
// ResolveFiles takes user-provided paths/globs and returns matching files.
func ResolveFiles(patterns []string, forEncryption bool) ([]string, error) {
    if len(patterns) == 0 {
        // Default: find all .env* or *.kanuka files recursively
        return FindAllSecretFiles(forEncryption)
    }
    
    var files []string
    for _, pattern := range patterns {
        // Check if it's a directory
        if info, err := os.Stat(pattern); err == nil && info.IsDir() {
            // Find all matching files in directory
            dirFiles, _ := FindSecretFilesInDir(pattern, forEncryption)
            files = append(files, dirFiles...)
            continue
        }
        
        // Try as glob pattern
        matches, err := filepath.Glob(pattern)
        if err != nil {
            return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
        }
        
        if len(matches) == 0 {
            // Treat as literal path
            files = append(files, pattern)
        } else {
            files = append(files, matches...)
        }
    }
    
    return files, nil
}
```

### Monorepo Considerations

This feature is a stepping stone toward monorepo support but does not fully solve it. For now:

**Supported workflows:**

1. **Single `.kanuka` at root, selective encryption:**
   ```bash
   cd monorepo
   kanuka secrets encrypt services/api/.env
   ```

2. **Separate `.kanuka` per service (current workaround):**
   ```bash
   cd monorepo/services/api
   kanuka secrets init
   kanuka secrets encrypt
   ```

**Future consideration (not in scope):**
- Path-based access control (different users for different paths)
- This would require significant design work and is deferred

### Acceptance Criteria

- [ ] `encrypt` accepts positional file arguments
- [ ] `decrypt` accepts positional file arguments
- [ ] Glob patterns work (`"*/.env"`)
- [ ] Directory arguments work
- [ ] Multiple arguments work
- [ ] Default behavior unchanged (no args = all files)
- [ ] Dry-run works with specific files
- [ ] Error handling for non-existent files
- [ ] Help text updated

### Testing Requirements

| Test Case | Description |
|-----------|-------------|
| `TestEncrypt_DefaultAllFiles` | No args encrypts all files |
| `TestEncrypt_SingleFile` | Single file argument works |
| `TestEncrypt_MultipleFiles` | Multiple file arguments work |
| `TestEncrypt_GlobPattern` | Glob patterns expand correctly |
| `TestEncrypt_Directory` | Directory argument processes contents |
| `TestEncrypt_NonExistentFile` | Appropriate error for missing file |
| `TestDecrypt_SpecificFile` | Decrypt specific file works |
| `TestDecrypt_GlobPattern` | Decrypt with glob works |

### Definition of Done

- [ ] `encrypt` command accepts file arguments
- [ ] `decrypt` command accepts file arguments
- [ ] Glob patterns work
- [ ] Directory support works
- [ ] Integration tests pass
- [ ] `golangci-lint run` passes
- [ ] Documentation updated
- [ ] Help text updated

---

## Future Considerations (Out of Scope)

### Monorepo Path-Based Access Control

A future enhancement could allow different users to have access to different paths:

```toml
# .kanuka/config.toml
[access."services/api/*"]
users = ["alice@example.com", "bob@example.com"]

[access."services/admin/*"]
users = ["charlie@example.com"]
```

This is deferred until there's demonstrated demand from real users.

### Audit Log Signing

Digital signatures on audit entries could be added if there's a need for tamper-evidence. This would require:
- Deciding on signature format
- Handling key rotation
- Verification tooling

Deferred as the current threat model doesn't justify the complexity.

### Custom Config/Data Paths

If users request `KANUKA_DATA_DIR` or `KANUKA_CONFIG_DIR` environment variables, they can be added as simple enhancements. For now, the XDG defaults are sufficient.
