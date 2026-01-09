---
title: Checking Encryption Status
description: A guide to viewing the encryption status of secret files using Kanuka.
---

The status command shows the encryption status of all secret files in your project.
It helps you understand which files are encrypted, which need attention, and
your overall security posture.

## Viewing status

To see the encryption status of all secret files:

```bash
kanuka secrets status
```

This displays a table showing each file and its status:

```
Project: my-project
Secret files status:

  FILE                      STATUS
  .env                      encrypted (up to date)
  .env.local                encrypted (up to date)
  config/.env.production    stale (plaintext modified after encryption)
  scripts/.env.test         not encrypted
  .env.backup.kanuka        encrypted only (no plaintext)

Summary:
  2 files up to date
  1 file stale (run 'kanuka secrets encrypt' to update)
  1 file not encrypted (run 'kanuka secrets encrypt' to secure)
  1 file encrypted only (plaintext removed, this is normal)
```

## Understanding file status

Each file can be in one of four states:

| Status | Meaning | Action needed |
|--------|---------|---------------|
| **encrypted (up to date)** | Encrypted file is newer than plaintext | None |
| **stale** | Plaintext modified after encryption | Run `encrypt` |
| **not encrypted** | Plaintext exists with no encrypted version | Run `encrypt` |
| **encrypted only** | Encrypted file exists, no plaintext | None (or `decrypt` if needed) |

### Up to date files

Files are "up to date" when the encrypted `.kanuka` file was created after the
plaintext file was last modified. This means the encrypted version contains
the latest content.

### Stale files

Stale files have been modified since they were encrypted. The plaintext is
newer than the encrypted version, meaning the encrypted file is out of date.

To update stale files:

```bash
kanuka secrets encrypt
```

### Unencrypted files

Unencrypted files are plaintext `.env` files that have no corresponding
`.kanuka` encrypted version. These files are a security risk if committed
to version control.

To encrypt them:

```bash
kanuka secrets encrypt
```

### Encrypted only files

These are `.kanuka` files where the plaintext has been removed. This is normal
and expected - many teams delete plaintext after encryption for security.

To restore the plaintext:

```bash
kanuka secrets decrypt
```

## JSON output

For scripting and automation, use the `--json` flag:

```bash
kanuka secrets status --json
```

This outputs machine-readable JSON:

```json
{
  "files": [
    {"path": ".env", "status": "current", "plaintextMtime": "2024-01-15T10:00:00Z", "encryptedMtime": "2024-01-15T10:30:00Z"},
    {"path": ".env.local", "status": "current", "plaintextMtime": "2024-01-14T09:00:00Z", "encryptedMtime": "2024-01-15T10:30:00Z"},
    {"path": "config/.env.production", "status": "stale", "plaintextMtime": "2024-01-15T11:00:00Z", "encryptedMtime": "2024-01-15T10:30:00Z"},
    {"path": "scripts/.env.test", "status": "unencrypted", "plaintextMtime": "2024-01-15T09:00:00Z", "encryptedMtime": null}
  ],
  "summary": {"current": 2, "stale": 1, "unencrypted": 1, "encryptedOnly": 0}
}
```

## Status examples

```bash
# View status of all secret files
kanuka secrets status

# JSON output for scripting
kanuka secrets status --json

# Check if any files need encryption (for CI)
kanuka secrets status --json | jq '.summary.stale + .summary.unencrypted'
```

## Using in CI/CD

You can use the status command to fail CI builds if secrets are out of date:

```bash
#!/bin/bash
# Fail if any secrets are stale or unencrypted
status=$(kanuka secrets status --json | jq '.summary.stale + .summary.unencrypted')
if [ "$status" -gt 0 ]; then
  echo "Error: $status secret file(s) need encryption"
  kanuka secrets status
  exit 1
fi
```

## Next steps

- **[Encryption guide](/guides/encryption/)** - Encrypt secret files
- **[Decryption guide](/guides/decryption/)** - Decrypt secret files
- **[Doctor command](/guides/doctor/)** - Run health checks on the project
