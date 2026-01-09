---
title: Cleaning Orphaned Entries
description: A guide to removing orphaned keys and inconsistent state using Kānuka.
---

The clean command removes orphaned entries detected by the `access` command.
An orphan is an encrypted symmetric key file (`.kanuka`) that has no corresponding
public key.

## When to use clean

Run `kanuka secrets clean` when:

- The `access` command shows entries with "orphan" status
- You want to clean up after a failed or interrupted operation
- You're tidying up the project after manual file changes

## Finding orphaned entries

First, check if there are any orphaned entries:

```bash
kanuka secrets access
```

If orphans exist, you'll see output like:

```
Users with access:

  UUID                                    EMAIL                     STATUS
  a1b2c3d4-e5f6-7890-abcd-ef1234567890    alice@example.com         active
  c3d4e5f6-a7b8-9012-cdef-123456789012    unknown                   orphan

Tip: Run 'kanuka secrets clean' to remove orphaned entries.
```

## Cleaning orphaned entries

To remove orphaned entries:

```bash
kanuka secrets clean
```

This shows the orphaned files and asks for confirmation:

```
Found 1 orphaned entry:

  UUID                                    FILE
  c3d4e5f6-a7b8-9012-cdef-123456789012    .kanuka/secrets/c3d4e5f6-...kanuka

This will permanently delete the orphaned files listed above.
These files cannot be recovered.

Do you want to continue? [y/N]:
```

Type `y` to confirm and remove the files.

## Previewing cleanup

Use the `--dry-run` flag to see what would be removed without making changes:

```bash
kanuka secrets clean --dry-run
```

This shows:
- Which files would be deleted
- No files are actually removed

## Skipping confirmation

In automated environments, use `--force` to skip the confirmation prompt:

```bash
kanuka secrets clean --force
```

:::caution
The `--force` flag will delete orphaned files without asking. Make sure you've
reviewed what will be deleted using `--dry-run` first.
:::

## Clean examples

```bash
# Preview what would be cleaned
kanuka secrets clean --dry-run

# Clean with confirmation prompt
kanuka secrets clean

# Clean without confirmation (for automation)
kanuka secrets clean --force
```

## What causes orphaned entries

Orphaned entries can occur when:

| Cause | Description |
|-------|-------------|
| Manual deletion | Someone deleted a public key file directly |
| Interrupted revoke | A revoke operation failed after deleting the public key |
| Partial restore | A backup was restored that didn't include public keys |
| File corruption | Files were lost or corrupted |

## After cleaning

After cleaning:

1. **Commit the changes** - The orphaned files have been removed
2. **Push to remote** - So the cleanup is reflected for the team

```bash
git add .kanuka/
git commit -m "Clean up orphaned entries"
git push
```

## Next steps

- **[Access command](/guides/access/)** - View who has access
- **[Doctor command](/guides/doctor/)** - Run health checks on the project
- **[Revoke guide](/guides/revoke/)** - Properly remove a user's access
