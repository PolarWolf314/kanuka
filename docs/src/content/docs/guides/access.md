---
title: Viewing Access
description: A guide to viewing who has access to a project's secrets using Kānuka.
---

The access command shows all users who have access to the project's secrets,
along with their current status.

## Viewing access

To see who has access to the project's secrets:

```bash
kanuka secrets access
```

This displays a table showing each user's UUID, email (if available), and status:

```
Project: my-project

Users with access:

  UUID                                    EMAIL                     STATUS
  a1b2c3d4-e5f6-7890-abcd-ef1234567890    alice@example.com         active
  b2c3d4e5-f6a7-8901-bcde-f12345678901    bob@example.com           active
  c3d4e5f6-a7b8-9012-cdef-123456789012    charlie@example.com       pending

Total: 3 users (2 active, 1 pending)
```

## Understanding user status

Each user can be in one of three states:

| Status | Meaning | Action needed |
|--------|---------|---------------|
| **active** | User has public key and encrypted symmetric key | None - user can decrypt |
| **pending** | User has public key but no encrypted symmetric key | Run `sync` to grant access |
| **orphan** | Encrypted key exists but no public key | Run `clean` to remove |

### Active users

Active users have both files present:
- A public key in `.kanuka/public_keys/<uuid>.pub`
- An encrypted symmetric key in `.kanuka/secrets/<uuid>.kanuka`

These users can decrypt secrets immediately.

### Pending users

Pending users have added their public key but haven't been granted access yet.
This typically happens when:

1. A new user runs `kanuka secrets create`
2. They commit and push their public key
3. But no one has run `register` or `sync` to encrypt the symmetric key for them

To grant pending users access:

```bash
kanuka secrets register --user pending-user@example.com
```

Or sync to grant access to all pending users at once:

```bash
kanuka secrets sync
```

### Orphaned entries

Orphaned entries have an encrypted symmetric key but no corresponding public key.
This inconsistent state can occur when:

- A public key was manually deleted
- A revoke operation was interrupted
- Files were partially restored from backup

To clean up orphaned entries:

```bash
kanuka secrets clean
```

## JSON output

For scripting and automation, use the `--json` flag:

```bash
kanuka secrets access --json
```

This outputs machine-readable JSON:

```json
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

## Access examples

```bash
# View all users with access
kanuka secrets access

# JSON output for scripting
kanuka secrets access --json

# Pipe to jq to filter active users
kanuka secrets access --json | jq '.users[] | select(.status == "active")'
```

## Next steps

- **[Register guide](/guides/register/)** - Grant access to new users
- **[Clean command](/guides/clean/)** - Remove orphaned entries
- **[Revoke guide](/guides/revoke/)** - Remove a user's access
