---
title: Audit Log
description: Understanding the audit log and viewing operation history.
---

Kanuka maintains an audit log of all secrets operations, providing visibility
into who did what and when. This is valuable for teams who need accountability
and a paper trail for security auditing.

## What gets logged

Every secrets operation is recorded:

- **Encrypt and decrypt** - Which files were processed
- **User registration and revocation** - Who was added or removed
- **Key rotation** - When `sync` or `rotate` was run
- **Initialization** - When a project was set up
- **Device creation** - When new devices were added
- **Cleanup operations** - When orphaned keys were removed
- **Import and export** - Backup and restore operations

## Log location

The audit log is stored at `.kanuka/audit.jsonl` and is committed to your
repository alongside other Kanuka files. This means:

- The log is versioned with git
- All team members can see the history
- No external dependencies required

## Viewing the log

Use the `log` command to view operation history:

```bash
kanuka secrets log
```

This displays entries in a human-readable format:

```
2024-01-15 10:30:00  alice@example.com    encrypt    .env, .env.local
2024-01-15 10:35:00  bob@example.com      register   charlie@example.com
2024-01-15 11:00:00  alice@example.com    revoke     charlie@example.com
```

See the [log command guide](/guides/log/) for filtering and formatting options.

## Log format

The log uses JSON Lines format (one JSON object per line), which is easy to
parse programmatically while remaining human-readable:

```json
{"ts":"2024-01-15T10:30:00.123456Z","user":"alice@example.com","uuid":"a1b2c3d4","op":"encrypt","files":[".env"]}
```

Each entry contains:

| Field | Description |
|-------|-------------|
| `ts` | Timestamp in RFC3339 format with microsecond precision |
| `user` | Email of the user who performed the operation |
| `uuid` | UUID of the user |
| `op` | Operation name (encrypt, decrypt, register, etc.) |

Additional fields vary by operation type (e.g., `files` for encrypt/decrypt,
`target_user` for register/revoke).

## Privacy considerations

The audit log contains:

- Timestamps of operations
- Email addresses of users
- File paths that were encrypted/decrypted
- User emails for register/revoke operations

It does **NOT** contain:

- Secret values
- Private keys
- Encryption keys
- File contents

## Handling merge conflicts

Since multiple team members may perform operations simultaneously, git merge
conflicts can occur in the audit log. These are easy to resolve:

1. Keep both sets of lines (the log is append-only)
2. Sort by timestamp if desired
3. Commit the resolved file

The audit log uses microsecond-precision timestamps to minimize the chance of
conflicts.

## Next steps

- Learn how to [filter and format the log](/guides/log/)
- See the [command reference](/reference/references/) for all log options
