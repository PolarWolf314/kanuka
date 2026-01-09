---
title: Running Health Checks
description: A guide to checking project health and detecting common issues using Kanuka.
---

The doctor command runs a series of health checks on your Kanuka project and
provides actionable suggestions for any issues found.

## Running doctor

To check the health of your project:

```bash
kanuka secrets doctor
```

This runs all health checks and displays the results:

```
Running health checks...

[pass] Project configuration valid
[pass] User configuration valid
[pass] Private key exists for this project
[pass] Private key permissions correct (0600)
[pass] All public keys have corresponding .kanuka files
[pass] All .kanuka files have corresponding public keys
[pass] .env patterns found in .gitignore
[warn] Found 1 unencrypted .env file (run 'kanuka secrets status')
[fail] 2 .env files not in .gitignore

Summary: 7 passed, 1 warning, 1 error

Suggestions:
  - Run 'kanuka secrets encrypt' to encrypt unprotected files
  - Add '.env*' to your .gitignore file
```

## Understanding results

Each check can have one of three results:

| Result | Meaning |
|--------|---------|
| **pass** | Check passed, no issues found |
| **warn** | Potential issue that should be addressed |
| **fail** | Critical issue that needs immediate attention |

## Health checks performed

The doctor command runs these checks:

| Check | Severity | What it checks |
|-------|----------|----------------|
| Project configuration | fail | `.kanuka/config.toml` exists and is valid |
| User configuration | fail | User config exists and is valid |
| Private key exists | fail | Private key file exists for this project |
| Private key permissions | warn | Private key has secure permissions (0600) |
| Public key consistency | fail | Every public key has a matching `.kanuka` file |
| Kanuka file consistency | fail | Every `.kanuka` user file has a matching public key |
| Gitignore patterns | warn | `.env` patterns are in `.gitignore` |
| Unencrypted files | warn | No plaintext `.env` files without encryption |

## Exit codes

The doctor command uses exit codes to indicate overall health:

| Exit code | Meaning |
|-----------|---------|
| 0 | All checks passed |
| 1 | Warnings found (but no errors) |
| 2 | Errors found |

This makes it easy to use in scripts and CI pipelines:

```bash
if kanuka secrets doctor; then
  echo "Project is healthy"
else
  echo "Issues found, check output above"
fi
```

## JSON output

For scripting and automation, use the `--json` flag:

```bash
kanuka secrets doctor --json
```

This outputs machine-readable JSON:

```json
{
  "checks": [
    {"name": "Project configuration valid", "status": "pass", "message": ""},
    {"name": "Private key permissions", "status": "warn", "message": "Permissions are 0644, should be 0600", "suggestion": "Run: chmod 600 ~/.kanuka/keys/project-uuid.pem"}
  ],
  "summary": {"pass": 7, "warn": 1, "fail": 1},
  "healthy": false
}
```

## Doctor examples

```bash
# Run all health checks
kanuka secrets doctor

# JSON output for scripting
kanuka secrets doctor --json

# Use in CI to fail on any issues
kanuka secrets doctor || exit 1
```

## Fixing common issues

### Private key permissions too open

```bash
# Fix permissions on your private key
chmod 600 ~/.kanuka/keys/<project-uuid>.pem
```

### .env files not in .gitignore

Add these patterns to your `.gitignore`:

```
# Environment files
.env
.env.*
!.env.example
!.env.*.kanuka
```

### Unencrypted .env files

```bash
# Encrypt all .env files
kanuka secrets encrypt
```

### Inconsistent user state (orphans or pending)

```bash
# View current access state
kanuka secrets access

# Clean up orphaned entries
kanuka secrets clean

# Grant access to pending users
kanuka secrets sync
```

## Next steps

- **[Status command](/guides/status/)** - Check encryption status of files
- **[Access command](/guides/access/)** - View who has access
- **[Clean command](/guides/clean/)** - Remove orphaned entries
