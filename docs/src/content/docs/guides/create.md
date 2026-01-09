---
title: Creating Secrets for Access
description: A guide to gaining access to a repo's secrets using Kānuka.
---

Kānuka uses a combination of RSA key pairs and symmetric keys to encrypt and
decrypt files. If you weren't the person who ran `kanuka secrets init`, you
won't have access to decrypt secrets. This guide shows you how to request access.

## Creating your keys

When you join a project that uses Kānuka, you need to create your encryption keys:

```bash
kanuka secrets create
```

This command:
1. Generates a public/private RSA key pair for you
2. Stores your private key securely in your user data directory
3. Adds your public key to the project (named with your UUID)
4. Records your device in the project configuration

### First-time users

If this is your first time using Kānuka, you'll be prompted to set up your identity:

```bash
$ kanuka secrets create
Welcome to Kānuka! Let's set up your identity.

Enter your email: alice@example.com
Enter your name: Alice Smith
Enter a device name [alice-macbook]: 
```

Your identity is stored in your [user configuration](/concepts/user-configuration/)
and reused across all projects.

### Key naming

Keys are named using your unique device UUID (e.g., `a1b2c3d4-5678-90ab-cdef-1234567890ab.pub`).
This allows you to have multiple devices registered to the same email without conflicts.

### Recreating keys

If you need to create new keys (e.g., when switching devices or if keys are compromised):

```bash
kanuka secrets create --force
```

:::caution
Using `--force` will:
- Generate a completely new key pair
- Override your existing public key in the project
- Require re-registration by someone with access
:::

### Custom device names

You can specify a custom device name during creation:

```bash
kanuka secrets create --device-name work-laptop
```

## Requesting access

After creating your keys, someone with existing access needs to register you:

```bash
# They run this command with your email
kanuka secrets register --user alice@example.com
```

The registering user will:
1. Look up your public key in the project by your email
2. Encrypt the symmetric key with your public key
3. Create your `.kanuka` file in the secrets directory

For more information about granting access, see the [registration guide](/guides/register/)
or the [registration concepts](/concepts/registration/).

## How it works

When you run `kanuka secrets create`:

1. **Key generation**: A 4096-bit RSA key pair is generated
2. **Private key storage**: Stored at `~/.local/share/kanuka/keys/<project-uuid>/privkey`
3. **Public key storage**: Placed in `.kanuka/public_keys/<your-uuid>.pub`
4. **Config update**: Your device is recorded in `.kanuka/config.toml`

The project's `config.toml` tracks all registered users and their devices:

```toml
[users]
"a1b2c3d4-..." = "alice@example.com"
"e5f6g7h8-..." = "bob@example.com"

[devices."a1b2c3d4-..."]
name = "alice-macbook"
created_at = 2024-01-15T10:30:00Z
```

## Next steps

- **[Registration guide](/guides/register/)** - Learn how to grant access to others
- **[Registration concepts](/concepts/registration/)** - Understand the key exchange process
- **[Project configuration](/concepts/project-configuration/)** - How users are tracked
- **[CLI reference](/reference/references/)** - Full command documentation
