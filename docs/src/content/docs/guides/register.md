---
title: Registering Other Users
description: A guide to giving access to a repo's secrets using Kanuka.
---

Kanuka uses a symmetric key to encrypt and decrypt files, and uses RSA key
pairs to encrypt the symmetric key. Any user who can decrypt the symmetric key
can grant access to others.

## Granting access to team members

Once a team member has created their keys with `kanuka secrets create`, you can
register them using their email address:

```bash
kanuka secrets register --user alice@example.com
```

This command:
1. Looks up the user's public key in `.kanuka/public_keys/` by their email
2. Encrypts the project's symmetric key with their public key
3. Creates their encrypted key file in `.kanuka/secrets/`

Commit these changes and they'll have access after pulling.

### Re-registering existing users

If you try to register a user who already has access, Kanuka will warn you:

```bash
$ kanuka secrets register --user alice@example.com
Warning: alice@example.com already has access to this project.
  Continuing will replace their existing key.
  If they generated a new keypair, this is expected.
  If not, they may lose access.

Do you want to continue? [y/N]:
```

This is useful when a user has generated a new keypair (e.g., on a new machine)
and needs their access updated.

To skip the confirmation prompt, use the `--force` flag:

```bash
kanuka secrets register --user alice@example.com --force
```

### Previewing registration

Use the `--dry-run` flag to preview what would be created without making changes:

```bash
kanuka secrets register --user alice@example.com --dry-run
```

This verifies that the user exists in the project config, their public key is
available, and shows which files would be created.

### Multiple devices

Users can have multiple devices registered under the same email. When you register
a user by email, Kanuka registers all of their devices that have public keys in
the project:

```bash
# Alice has two devices: macbook and desktop
kanuka secrets register --user alice@example.com
# Both devices are now registered
```

## Using a custom public key

You can register users who haven't yet created keys in the project by providing
their public key directly.

:::tip
Kanuka accepts both OpenSSH and PEM formats for RSA public keys.
:::

### Passing a key file path

Register a user by providing the path to their public key file:

```bash
kanuka secrets register --file path/to/their-key.pub
```

Kanuka will:
1. Copy the public key to `.kanuka/public_keys/`
2. Create their encrypted symmetric key
3. Add them to the project configuration

### Passing key contents directly

You can also pass the public key contents as a string. This requires specifying
a name for identification:

```bash
# Paste the contents of an OpenSSH format public key
kanuka secrets register --pubkey "ssh-rsa AAAAB3NzaC1..." --user teammate@example.com

# Or pass the key dynamically
kanuka secrets register --pubkey "$(cat path/to/pubkey)" --user teammate@example.com
```

:::tip
The `--user` flag is required with `--pubkey` because the key contents don't
include any identifying information.
:::

## Using in CI/CD pipelines

In automated environments where your private key isn't stored on disk, you can
pipe it directly from a secrets manager using the `--private-key-stdin` flag:

```bash
# From HashiCorp Vault
vault read -field=private_key secret/kanuka | kanuka secrets register --user alice@example.com --private-key-stdin

# From 1Password CLI
op read "op://Vault/Kanuka/private_key" | kanuka secrets register --user alice@example.com --private-key-stdin

# From environment variable
echo "$KANUKA_PRIVATE_KEY" | kanuka secrets register --user alice@example.com --private-key-stdin
```

This is useful for automated onboarding workflows where you need to register
new team members without manual intervention.

:::tip
If your private key is passphrase-protected, Kanuka will prompt for the
passphrase via `/dev/tty`, allowing you to pipe the key while still entering
the passphrase interactively.
:::

## Viewing registered users

The project's registered users are tracked in `.kanuka/config.toml`:

```toml
[users]
"a1b2c3d4-5678-90ab-cdef-1234567890ab" = "alice@example.com"
"e5f6g7h8-1234-56cd-efgh-9876543210ab" = "bob@example.com"

[devices."a1b2c3d4-5678-90ab-cdef-1234567890ab"]
name = "alice-macbook"
created_at = 2024-01-15T10:30:00Z
```

You can also see registered users by listing the public keys directory:

```bash
ls .kanuka/public_keys/
# a1b2c3d4-5678-90ab-cdef-1234567890ab.pub
# e5f6g7h8-1234-56cd-efgh-9876543210ab.pub
```

## Registration workflow

Here's the typical workflow for adding a new team member:

1. **New member joins**: They clone the repository
2. **Create keys**: They run `kanuka secrets create`
3. **Commit public key**: They commit and push `.kanuka/public_keys/<uuid>.pub`
4. **Register**: You pull their changes and run `kanuka secrets register --user their@email.com`
5. **Grant access**: You commit and push the changes
6. **Decrypt**: They pull and can now run `kanuka secrets decrypt`

## Next steps

- **[Registration concepts](/concepts/registration/)** - Understand the key exchange process
- **[Revoking access](/guides/revoke/)** - Remove a user's access
- **[CLI reference](/reference/references/)** - Full command documentation
