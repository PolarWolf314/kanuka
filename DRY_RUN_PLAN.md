# Dry-Run Flag Implementation Plan

This document analyzes all Kanuka CLI commands to identify where a `--dry-run` flag would provide meaningful value, with honest assessments of priority and implementation complexity.

## Executive Summary

After reviewing all commands, **4 commands** are strong candidates for `--dry-run`:
- `secrets revoke` (HIGH priority)
- `secrets encrypt` (MEDIUM priority)  
- `secrets decrypt` (MEDIUM priority)
- `secrets register` (LOW priority)

Several commands are explicitly **not recommended** for dry-run due to low value or added complexity without benefit.

---

## Command Analysis

### HIGH Priority

#### `kanuka secrets revoke`

| Aspect | Details |
|--------|---------|
| **What it modifies** | Deletes `.pub` and `.kanuka` files, updates `config.toml`, rotates symmetric key for all remaining users |
| **Reversibility** | Partially reversible via git, but key rotation is irreversible |
| **Dry-run value** | **Very High** - Shows exactly which files will be deleted, which users affected, confirms key rotation will occur |
| **Implementation complexity** | Low |

**Why this is the top priority:**
- Destructive operation that deletes files and rotates keys
- Already has `--yes` flag, indicating users want control over execution
- Multi-device users get a confirmation prompt, but single-device users don't
- Key rotation affects all remaining users (re-encrypts their symmetric keys)

**Dry-run output would show:**
```
[dry-run] Would revoke access for alice@example.com

Files that would be deleted:
  - .kanuka/public_keys/a1b2c3d4-5678-90ab-cdef-1234567890ab.pub
  - .kanuka/secrets/a1b2c3d4-5678-90ab-cdef-1234567890ab.kanuka

Config changes:
  - Remove user UUID a1b2c3d4-... from [users]
  - Remove device entry from [devices]

Post-revocation:
  - Symmetric key would be rotated for 3 remaining users

No changes made. Run without --dry-run to execute.
```

**Implementation notes:**
- Collect files to delete in `getFilesToRevoke()` (already done)
- Skip actual `os.Remove()` calls and `configs.SaveProjectConfig()`
- Skip key rotation
- Print summary instead

---

### MEDIUM Priority

#### `kanuka secrets encrypt`

| Aspect | Details |
|--------|---------|
| **What it modifies** | Creates/overwrites `.kanuka` files from `.env` files |
| **Reversibility** | Fully reversible (can re-encrypt, old `.kanuka` files in git history) |
| **Dry-run value** | **Medium** - Shows which files would be created/updated |
| **Implementation complexity** | Low |

**Why this is useful:**
- Users may want to preview which files will be encrypted before committing
- Helps verify the right `.env` files are being picked up
- Useful when running in a new project to confirm file discovery

**Dry-run output would show:**
```
[dry-run] Would encrypt 3 environment files:

  .env                    → .env.kanuka
  src/config/.env.local   → src/config/.env.local.kanuka
  tests/.env.test         → tests/.env.test.kanuka

No changes made. Run without --dry-run to execute.
```

**Implementation notes:**
- Already discovers files via `secrets.FindEnvOrKanukaFiles()`
- Skip `secrets.EncryptFiles()` call
- Print file mapping instead

---

#### `kanuka secrets decrypt`

| Aspect | Details |
|--------|---------|
| **What it modifies** | Creates/overwrites `.env` files from `.kanuka` files |
| **Reversibility** | Reversible (can re-run encrypt, but may lose local changes to `.env`) |
| **Dry-run value** | **Medium** - Shows which files would be created, warns about overwrites |
| **Implementation complexity** | Low-Medium |

**Why this is useful:**
- **Overwrite warning**: If `.env` files already exist with local modifications, users may want to know before overwriting
- Helps verify correct `.kanuka` files are being decrypted
- Useful in CI/CD pipelines for validation

**Dry-run output would show:**
```
[dry-run] Would decrypt 3 encrypted files:

  .env.kanuka                    → .env (exists, would be overwritten)
  src/config/.env.local.kanuka   → src/config/.env.local (new file)
  tests/.env.test.kanuka         → tests/.env.test (new file)

Warning: 1 existing file would be overwritten.

No changes made. Run without --dry-run to execute.
```

**Implementation notes:**
- Already discovers files via `secrets.FindEnvOrKanukaFiles()`
- Check if destination files exist before skipping decrypt
- Print file mapping with overwrite warnings

---

### LOW Priority

#### `kanuka secrets register`

| Aspect | Details |
|--------|---------|
| **What it modifies** | Creates `.kanuka` file for target user, may copy public key |
| **Reversibility** | Fully reversible (delete created files) |
| **Dry-run value** | **Low** - Operation is simple and clearly communicated in output |
| **Implementation complexity** | Medium |

**Why this is lower priority:**
- Non-destructive (only creates files)
- Current output already clearly shows what was created
- Single user affected per invocation
- Easy to undo by deleting created files

**Dry-run output would show:**
```
[dry-run] Would register alice@example.com

Files that would be created:
  - .kanuka/secrets/a1b2c3d4-5678-90ab-cdef-1234567890ab.kanuka

No changes made. Run without --dry-run to execute.
```

**Implementation notes:**
- Validate user exists and public key is accessible
- Skip `secrets.SaveKanukaKeyToProject()` call
- Print summary instead

---

### NOT RECOMMENDED

#### `kanuka secrets init`

| Aspect | Details |
|--------|---------|
| **What it modifies** | Creates `.kanuka/` directory structure, config files, RSA keys |
| **Reversibility** | Fully reversible (`rm -rf .kanuka/`) |
| **Dry-run value** | **Very Low** |
| **Recommendation** | **Skip** |

**Why dry-run doesn't make sense:**
- Already fails gracefully if project is already initialized
- Non-destructive (only creates new files/directories)
- One-time operation per project
- Simple to undo if needed

---

#### `kanuka secrets create`

| Aspect | Details |
|--------|---------|
| **What it modifies** | Creates RSA key pair, copies public key to project, updates configs |
| **Reversibility** | Mostly reversible (delete files), but private key generation is one-way |
| **Dry-run value** | **Very Low** |
| **Recommendation** | **Skip** |

**Why dry-run doesn't make sense:**
- Already has `--force` flag for explicit override behavior
- Fails gracefully if public key already exists
- User needs to run this to gain access - previewing doesn't help
- Output already clearly explains next steps

---

#### `kanuka config init`

| Aspect | Details |
|--------|---------|
| **What it modifies** | Creates/updates `~/.config/kanuka/config.toml` |
| **Reversibility** | Fully reversible (edit or delete file) |
| **Dry-run value** | **Very Low** |
| **Recommendation** | **Skip** |

**Why dry-run doesn't make sense:**
- Interactive command with prompts - user sees exactly what they're entering
- Already shows summary of what was saved after completion
- Non-destructive to project files
- User config is personal and easily editable

---

#### `kanuka config rename-device`

| Aspect | Details |
|--------|---------|
| **What it modifies** | Updates device name in project `config.toml`, may update user config |
| **Reversibility** | Fully reversible (rename again) |
| **Dry-run value** | **Very Low** |
| **Recommendation** | **Skip** |

**Why dry-run doesn't make sense:**
- Already validates extensively before making changes
- Shows clear error messages if device not found
- Simple string change, easy to undo
- Command name makes the action obvious

---

#### `kanuka config set-device-name`

| Aspect | Details |
|--------|---------|
| **What it modifies** | Updates device name preference in user config |
| **Reversibility** | Fully reversible (set again) |
| **Dry-run value** | **Very Low** |
| **Recommendation** | **Skip** |

**Why dry-run doesn't make sense:**
- Only affects user's local config
- Already shows before/after if updating existing value
- Trivial to change again

---

#### `kanuka config show` / `kanuka config list-devices`

| Aspect | Details |
|--------|---------|
| **What it modifies** | Nothing (read-only) |
| **Dry-run value** | **None** |
| **Recommendation** | **N/A** |

These are read-only commands - dry-run is not applicable.

---

## Implementation Recommendations

### Phase 1: High Value (Do First)

1. **`secrets revoke --dry-run`**
   - Highest impact, lowest effort
   - Users already expect confirmation for destructive operations
   - Complements existing `--yes` flag

### Phase 2: Quality of Life

2. **`secrets encrypt --dry-run`**
3. **`secrets decrypt --dry-run`**
   - Both use similar file discovery patterns
   - Can share implementation approach
   - Useful for CI/CD validation

### Phase 3: Completeness (Optional)

4. **`secrets register --dry-run`**
   - Lower value but completes the "secrets" command family
   - Only implement if users request it

---

## Implementation Pattern

All dry-run implementations should follow this pattern:

```go
var dryRun bool

func init() {
    cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview changes without executing")
}

// In RunE:
if dryRun {
    // Perform all validation and discovery
    // Skip actual file operations
    // Print detailed summary
    fmt.Println(color.YellowString("[dry-run]") + " No changes made. Run without --dry-run to execute.")
    return nil
}
```

### Output Format Guidelines

1. Use `[dry-run]` prefix in yellow for visibility
2. List all files that would be created/modified/deleted
3. Show config changes that would occur
4. End with clear "no changes made" message
5. Suggest running without `--dry-run` to execute

---

## Testing Considerations

Each dry-run implementation should have tests verifying:

1. No files are created/modified/deleted
2. No config changes are persisted
3. Output correctly describes what would happen
4. All validation still runs (invalid input still fails)

---

## Estimated Effort

| Command | Effort | Notes |
|---------|--------|-------|
| `secrets revoke` | 2-3 hours | Refactor to separate validation from execution |
| `secrets encrypt` | 1-2 hours | Simple - skip encrypt call, print file list |
| `secrets decrypt` | 2-3 hours | Add overwrite detection logic |
| `secrets register` | 2-3 hours | Multiple registration paths to handle |

**Total estimated effort: 7-11 hours**

---

## Questions for Review

1. Should `--dry-run` be a global flag on the `secrets` parent command, or individual flags per subcommand?
   - **Recommendation**: Individual flags - not all subcommands benefit equally

2. Should dry-run output be machine-readable (JSON) for CI/CD use cases?
   - **Recommendation**: Start with human-readable, add `--dry-run --json` later if needed

3. Should dry-run validate that the user has access (decrypt symmetric key) or skip that check?
   - **Recommendation**: Perform full validation - dry-run should catch permission errors too
