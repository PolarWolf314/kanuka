---
title: Exporting Secrets
description: A guide to creating backup archives of encrypted secrets using Kanuka.
---

The export command creates a backup archive of your project's encrypted secrets.
This is useful for disaster recovery, migration, and archival purposes.

## What gets exported

The export includes only encrypted data:

- `.kanuka/config.toml` - Project configuration
- `.kanuka/public_keys/*.pub` - All user public keys
- `.kanuka/secrets/*.kanuka` - Encrypted symmetric keys for each user
- All `*.kanuka` files - Encrypted secret files throughout the project

The export does **not** include:

- Private keys (these are stored locally, not in the project)
- Plaintext `.env` files (only encrypted versions are included)

## Creating an export

To export your encrypted secrets:

```bash
kanuka secrets export
```

This creates an archive with a date-stamped filename:

```
Exported secrets to kanuka-secrets-2024-01-15.tar.gz

Archive contents:
  .kanuka/config.toml
  .kanuka/public_keys/ (3 files)
  .kanuka/secrets/ (3 user keys)
  5 encrypted secret files

Note: This archive contains encrypted data only.
      Private keys are NOT included.
```

## Custom output path

Use the `-o` or `--output` flag to specify a custom output path:

```bash
kanuka secrets export -o /backups/project-secrets.tar.gz
```

## Archive format

The export creates a gzip-compressed tar archive (`.tar.gz`) with this structure:

```
kanuka-secrets-2024-01-15.tar.gz
├── .kanuka/
│   ├── config.toml
│   ├── public_keys/
│   │   ├── user1-uuid.pub
│   │   └── user2-uuid.pub
│   └── secrets/
│       ├── user1-uuid.kanuka
│       └── user2-uuid.kanuka
├── .env.kanuka
├── .env.local.kanuka
└── config/.env.production.kanuka
```

## Export examples

```bash
# Export with default filename (includes date)
kanuka secrets export

# Export to specific path
kanuka secrets export -o ~/backups/myproject-secrets.tar.gz

# Export to a shared backup location
kanuka secrets export -o /shared/backups/$(date +%Y%m%d)-secrets.tar.gz
```

## Storing exports safely

Since exports contain encrypted data, they are safe to store in most locations.
However, for best practices:

1. **Store in a secure location** - Use encrypted storage if available
2. **Maintain access control** - Limit who can access backup files
3. **Consider retention policy** - Old backups may contain outdated encryption
4. **Test restoration** - Periodically verify backups can be restored

:::note
While the secrets in the archive are encrypted, the project structure and file
names are visible. If this metadata is sensitive, consider encrypting the
archive itself.
:::

## Using exports for disaster recovery

To restore from an export:

1. Clone or set up a fresh project repository
2. Use `kanuka secrets import` to restore the encrypted files
3. Ensure you have your private key available
4. Run `kanuka secrets decrypt` to access the secrets

See the [Import guide](/guides/import/) for detailed restoration instructions.

## Next steps

- **[Import guide](/guides/import/)** - Restore secrets from an export
- **[Sync command](/guides/sync/)** - Rotate encryption keys
- **[Status command](/guides/status/)** - Check encryption status
