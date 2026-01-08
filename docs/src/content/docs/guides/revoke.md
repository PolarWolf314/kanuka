---
title: Revoking Someone's Access
description: A guide to revoking a user's access to a repo's secrets using Kanuka.
---

When a team member leaves or a device is compromised, you can revoke their access
to the project's secrets using Kanuka.

## Previewing revocation

Before revoking access, you can preview what would happen using the `--dry-run` flag:

```bash
kanuka secrets revoke --user alice@example.com --dry-run
```

This shows:
- Which files would be deleted (public keys and encrypted symmetric keys)
- Which config entries would be removed
- How many remaining users would have their keys rotated

No changes are made when using `--dry-run`, so you can safely verify the impact
before executing the revocation.

## Revoking by email

To revoke all access for a user across all their devices:

```bash
kanuka secrets revoke --user alice@example.com
```

This removes:
- Their public key(s) from `.kanuka/public_keys/`
- Their encrypted symmetric key(s) from `.kanuka/secrets/`
- Their entries from the project configuration

### Multiple devices confirmation

If the user has multiple devices registered, Kanuka will ask for confirmation:

```bash
$ kanuka secrets revoke --user alice@example.com

⚠ Warning: alice@example.com has 2 devices:
  - macbook-pro (created: Jan 15, 2024)
  - work-desktop (created: Jan 20, 2024)

This will revoke ALL devices for this user.
Proceed? [y/N]: 
```

To skip confirmation (useful for automation):

```bash
kanuka secrets revoke --user alice@example.com --yes
```

## Revoking a specific device

If a user's device is compromised but they should retain access on other devices,
revoke only that specific device:

```bash
kanuka secrets revoke --user alice@example.com --device macbook-pro
```

This is useful when:
- A laptop is lost or stolen
- A team member gets a new computer
- You want to clean up old device registrations

## Revoking by file path

You can also revoke by directly specifying the `.kanuka` file path:

```bash
kanuka secrets revoke --file .kanuka/secrets/a1b2c3d4-5678-90ab-cdef-1234567890ab.kanuka
```

This removes both the encrypted symmetric key and the corresponding public key.

## What happens after revocation

When you revoke a user, Kanuka automatically:

1. **Removes their files** - Public key and encrypted symmetric key are deleted
2. **Updates the config** - Their entry is removed from `.kanuka/config.toml`
3. **Rotates the symmetric key** - A new symmetric key is generated and encrypted
   for all remaining users

### Key rotation

The automatic key rotation ensures the revoked user cannot decrypt any secrets
encrypted after the revocation, even if they had previously obtained the symmetric key.

:::caution[Important]
The revoked user may still have access to **old secret values** from their local
git history. If the user was compromised or is a security concern, you should
also rotate your actual secret values (API keys, passwords, etc.) after revocation.
:::

## Revocation examples

```bash
# Preview revocation without making changes
kanuka secrets revoke --user alice@example.com --dry-run

# Revoke all devices for a user
kanuka secrets revoke --user alice@example.com

# Revoke a specific device
kanuka secrets revoke --user alice@example.com --device old-laptop

# Preview specific device revocation
kanuka secrets revoke --user alice@example.com --device old-laptop --dry-run

# Revoke without confirmation (for CI/CD automation)
kanuka secrets revoke --user alice@example.com --yes

# Revoke by file path
kanuka secrets revoke --file .kanuka/secrets/abc123.kanuka
```

## Using in CI/CD pipelines

In automated environments where your private key isn't stored on disk, you can
pipe it directly from a secrets manager using the `--private-key-stdin` flag:

```bash
# From HashiCorp Vault
vault read -field=private_key secret/kanuka | kanuka secrets revoke --user alice@example.com --yes --private-key-stdin

# From 1Password CLI
op read "op://Vault/Kanuka/private_key" | kanuka secrets revoke --user alice@example.com --yes --private-key-stdin

# From environment variable
echo "$KANUKA_PRIVATE_KEY" | kanuka secrets revoke --user alice@example.com --yes --private-key-stdin
```

Note the `--yes` flag to skip confirmation prompts in automated environments.

:::tip
If your private key is passphrase-protected, Kanuka will prompt for the
passphrase via `/dev/tty`, allowing you to pipe the key while still entering
the passphrase interactively.
:::

## After revoking

After revoking access:

1. **Commit the changes** - The file deletions and config updates
2. **Push to remote** - So the revocation takes effect for the team
3. **Consider rotating secrets** - If the revocation was security-related

```bash
git add .kanuka/
git commit -m "Revoke access for alice@example.com"
git push
```

## Next steps

- **[Registration concepts](/concepts/registration/)** - Understand the key exchange process
- **[Registration guide](/guides/register/)** - Add new team members
- **[CLI reference](/reference/references/)** - Full command documentation
