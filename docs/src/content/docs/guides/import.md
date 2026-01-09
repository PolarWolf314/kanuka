---
title: Importing Secrets
description: A guide to restoring secrets from a backup archive using Kānuka.
---

The import command restores encrypted secrets from an export archive. This is
useful for disaster recovery, setting up new machines, or migrating projects.

## Prerequisites

Before importing, ensure you have:

1. An export archive created with `kanuka secrets export`
2. Your private key available (required to decrypt after import)
3. Write access to the project directory

## Importing an archive

To import secrets from an archive:

```bash
kanuka secrets import kanuka-secrets-2024-01-15.tar.gz
```

If the project already has a `.kanuka` directory, you'll be prompted to choose
how to handle conflicts:

```
Importing secrets...

Found existing .kanuka directory. How do you want to proceed?
  [m] Merge - Add new files, keep existing
  [r] Replace - Delete existing, use backup
  [c] Cancel

Choice:
```

## Merge vs Replace

### Merge mode

Merge mode adds files from the archive that don't exist locally, while keeping
existing files intact:

```bash
kanuka secrets import backup.tar.gz --merge
```

Use merge when:
- You want to add missing files from a backup
- You have local changes you want to preserve
- You're combining secrets from multiple sources

Example output:

```
Importing files:
  .kanuka/config.toml (skipped - exists)
  .kanuka/public_keys/user1-uuid.pub (skipped - exists)
  .kanuka/public_keys/user3-uuid.pub (added)
  .env.kanuka (skipped - exists)
  config/.env.production.kanuka (added)

Summary:
  2 files added
  3 files skipped (already exist)
```

### Replace mode

Replace mode deletes all existing encrypted files and replaces them with the
archive contents:

```bash
kanuka secrets import backup.tar.gz --replace
```

Use replace when:
- You want to fully restore from backup
- Your local state is corrupted or inconsistent
- You're setting up a clean environment

:::caution
Replace mode will delete all existing `.kanuka` directory contents and encrypted
files before importing. This cannot be undone.
:::

## Previewing import

Use the `--dry-run` flag to see what would happen without making changes:

```bash
kanuka secrets import backup.tar.gz --dry-run
```

This shows:
- Which files would be added
- Which files would be skipped (in merge mode)
- Which files would be deleted (in replace mode)

## Import examples

```bash
# Import with interactive prompt for merge/replace
kanuka secrets import backup.tar.gz

# Merge new files, keep existing
kanuka secrets import backup.tar.gz --merge

# Replace all with backup contents
kanuka secrets import backup.tar.gz --replace

# Preview import without making changes
kanuka secrets import backup.tar.gz --dry-run

# Preview replace mode
kanuka secrets import backup.tar.gz --replace --dry-run
```

## After importing

After a successful import:

1. **Verify the import** - Check that expected files are present
2. **Decrypt to test** - Run `kanuka secrets decrypt` to verify you have access
3. **Commit if needed** - If import added files, commit them

```bash
# Verify files were imported
kanuka secrets status

# Test decryption
kanuka secrets decrypt

# Commit new files if any were added
git add .kanuka/ *.kanuka
git commit -m "Restore secrets from backup"
```

## Disaster recovery workflow

Complete workflow for restoring from backup:

```bash
# 1. Clone fresh repository
git clone https://github.com/org/project.git
cd project

# 2. Import backup
kanuka secrets import /backups/kanuka-secrets-2024-01-15.tar.gz --replace

# 3. Ensure private key is available
# (Copy from secure backup or another machine)
cp /backup/private-key.pem ~/.kanuka/keys/<project-uuid>.pem
chmod 600 ~/.kanuka/keys/<project-uuid>.pem

# 4. Decrypt and verify
kanuka secrets decrypt

# 5. Commit restored files
git add .
git commit -m "Restore project secrets from backup"
```

## Archive validation

Before importing, Kānuka validates the archive structure to ensure it contains
the expected files:

- Must contain `.kanuka/config.toml`
- Must be a valid gzip-compressed tar archive

If validation fails, the import is aborted with an error message.

## Next steps

- **[Export guide](/guides/export/)** - Create backup archives
- **[Status command](/guides/status/)** - Verify encryption status
- **[Doctor command](/guides/doctor/)** - Check project health after import
