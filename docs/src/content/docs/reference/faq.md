---
title: Frequently Asked Questions
description: Common questions and answers about using Kanuka.
---

## Why do encrypted files change even when my secrets haven't?

This is expected behavior and a security feature. When you run
`kanuka secrets encrypt`, the output differs each time due to how AES-GCM
encryption works.

**Technical explanation:** AES-GCM requires a unique nonce (number used once)
for each encryption operation. Kānuka generates a random nonce each time,
so encrypting the same plaintext produces different ciphertext.

**Why this is secure:** If encryption were deterministic, attackers could:
- Detect when secrets are reused across files
- Build dictionaries to guess plaintext values
- Identify patterns in your encrypted data

**What to do about git diffs:**
1. Only run `encrypt` when you actually change secrets
2. Commit the `.kanuka` files immediately after encrypting
3. If you encrypted by accident, run `git checkout -- *.kanuka` to discard

For more details, see the [encryption guide](/guides/encryption/).

## Why do I need to provide my email?

Your email address serves as a human-readable identifier that links your
cryptographic keys to your identity. While Kanuka uses UUIDs internally for
key file naming (for uniqueness and privacy), your email helps other team
members identify who has access to the project secrets.

When you run `kanuka secrets create`, your email is stored in:
- Your local user configuration (`~/.config/kanuka/config.toml`)
- The project configuration (`.kanuka/project.toml`)

This allows commands like `kanuka secrets register --user alice@example.com`
to work intuitively, referencing users by their email rather than cryptic UUIDs.

## What if I have multiple devices?

Kanuka supports multiple devices per user. Each device gets its own RSA key
pair and is tracked separately in the project configuration.

When you run `kanuka secrets create` on a new device:
1. A new key pair is generated for that specific device
2. The device is registered with an auto-generated name (based on hostname)
3. You can specify a custom name with `--device-name`

To see all devices for a user, check the project configuration or use the
revoke command which will list available devices.

Example with multiple devices:
```bash
# On your laptop
kanuka secrets create --email alice@example.com --device-name macbook

# On your desktop
kanuka secrets create --email alice@example.com --device-name desktop
```

Each device needs to be registered separately by someone with access:
```bash
kanuka secrets register --user alice@example.com
```

## How do I revoke a compromised device?

If one of your devices is compromised, you should immediately revoke only that
device while keeping access on your other devices:

```bash
# Revoke only the compromised device
kanuka secrets revoke --user alice@example.com --device compromised-laptop
```

This will:
1. Remove the public key and encrypted symmetric key for that device
2. Rotate the symmetric key for all remaining users
3. Update the project configuration

After revocation:
1. Commit the changes to version control
2. **Important**: Rotate your actual secret values (API keys, passwords, etc.)
   since the compromised device may still have access to old secrets via git history

If you need to revoke all devices for a user:
```bash
kanuka secrets revoke --user alice@example.com
```

## Can I have multiple emails?

Currently, each user should use a single email address consistently across all
their devices. The email serves as the primary identifier for grouping devices
belonging to the same person.

If you need to use a different email:
1. Revoke access for the old email on all devices
2. Run `kanuka secrets create --email new@example.com` on each device
3. Have someone register the new email

Using multiple emails would result in being treated as separate users, which
means:
- Separate key management
- Separate revocation tracking
- Potential confusion for team members

For most use cases, stick to one email address (typically your work email) for
all your devices.
