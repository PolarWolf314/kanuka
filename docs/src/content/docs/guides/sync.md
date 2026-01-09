---
title: Syncing Secrets
description: A guide to rotating encryption keys and re-encrypting secrets using Kanuka.
---

The sync command re-encrypts all secret files with a newly generated symmetric key.
This is useful for periodic key rotation, after adding team members, or if you suspect
a key may have been compromised.

## When to use sync

Use `kanuka secrets sync` when you want to:

- **Rotate keys periodically** for security hygiene
- **Ensure consistency** after adding new team members
- **Respond to potential compromise** by invalidating the old key

After syncing, all users with access will receive the new symmetric key encrypted
with their public key. The old symmetric key will no longer work.

## Running sync

To sync all secrets with a new encryption key:

```bash
kanuka secrets sync
```

This command:
1. Decrypts all secret files using your current symmetric key
2. Generates a new symmetric key
3. Re-encrypts the symmetric key for each user with access
4. Re-encrypts all secret files with the new key
5. Writes the updated files to disk

After syncing, commit and push the changes so other team members can pull the
newly encrypted files.

## Previewing sync

Use the `--dry-run` flag to see what would happen without making any changes:

```bash
kanuka secrets sync --dry-run
```

This shows:
- How many secret files would be re-encrypted
- Which users would receive the new key
- No files are modified during a dry run

## Sync examples

```bash
# Standard sync with new key generation
kanuka secrets sync

# Preview without making changes
kanuka secrets sync --dry-run

# Verbose output for debugging
kanuka secrets sync --verbose
```

## Using in CI/CD pipelines

In automated environments where your private key isn't stored on disk, you can
pipe it directly from a secrets manager using the `--private-key-stdin` flag:

```bash
# From HashiCorp Vault
vault read -field=private_key secret/kanuka | kanuka secrets sync --private-key-stdin

# From 1Password CLI
op read "op://Vault/Kanuka/private_key" | kanuka secrets sync --private-key-stdin

# From environment variable
echo "$KANUKA_PRIVATE_KEY" | kanuka secrets sync --private-key-stdin
```

:::tip
If your private key is passphrase-protected, Kanuka will prompt for the
passphrase via `/dev/tty`, allowing you to pipe the key while still entering
the passphrase interactively.
:::

## What happens during sync

The sync operation is atomic - either all files are updated or none are:

1. All secret files are decrypted into memory
2. A new 256-bit symmetric key is generated using secure random bytes
3. The new key is encrypted for each user's public key
4. All secrets are re-encrypted with the new key
5. Only after all encryption succeeds are files written to disk

If any step fails, no files are modified.

## After syncing

After a successful sync:

1. **Commit the changes** - All `.kanuka` files have been updated
2. **Push to remote** - So team members get the new encrypted files
3. **Team members pull and decrypt** - They can decrypt with no additional steps

```bash
git add .
git commit -m "Rotate encryption key"
git push
```

## Next steps

- **[Status command](/guides/status/)** - Check encryption status of files
- **[Access command](/guides/access/)** - View who has access to secrets
- **[Revoke guide](/guides/revoke/)** - Remove a user's access (includes automatic sync)
