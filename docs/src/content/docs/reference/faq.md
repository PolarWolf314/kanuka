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

## What private key formats does Kanuka support?

Kanuka supports RSA private keys in the following formats:

**PEM PKCS#1** - Traditional OpenSSL format:
```
-----BEGIN RSA PRIVATE KEY-----
```

**PEM PKCS#8** - Newer OpenSSL format:
```
-----BEGIN PRIVATE KEY-----
```

**OpenSSH** - Default format from modern ssh-keygen (OpenSSH 7.8+):
```
-----BEGIN OPENSSH PRIVATE KEY-----
```

Passphrase-protected keys are supported. Kanuka will prompt you for your
passphrase when needed. If you're running in a non-interactive environment
(like CI/CD), you can use the `--private-key-stdin` flag to pipe your key
from a secrets manager.

:::note
Only RSA keys are supported. Ed25519 and ECDSA keys are not compatible with
Kanuka. See [Why does Kanuka only support RSA keys?](#why-does-kanuka-only-support-rsa-keys)
for details.
:::

## Why does Kanuka only support RSA keys?

Kanuka intentionally supports only RSA keys to keep the implementation simple
and reliable. Here's the reasoning:

1. **RSA supports direct encryption** - Ed25519 is a signature-only algorithm
   and cannot encrypt data directly. Using it for encryption would require
   implementing ECIES (Elliptic Curve Integrated Encryption Scheme), adding
   significant complexity.

2. **Sufficient security** - RSA-2048 provides approximately 112 bits of
   security, which is sufficient for current threats. The performance
   advantages of Ed25519 are irrelevant for Kanuka's use case (encrypting
   small symmetric keys infrequently).

3. **Universal tooling support** - RSA keys can be generated and managed with
   any SSH or OpenSSL tooling, making them the most universally supported
   option.

4. **Simplicity over flexibility** - Supporting multiple key types would add
   significant implementation complexity, testing burden, and potential for
   user confusion without providing meaningful benefits.

If you only have Ed25519 keys, you'll need to generate an RSA key for use
with Kanuka:

```bash
ssh-keygen -t rsa -b 4096 -f ~/.ssh/kanuka_rsa
```

## Troubleshooting

### "unsupported private key format"

Your key may be in an unsupported format. Kanuka only supports RSA keys.
Check your key type:

```bash
ssh-keygen -l -f your_key
```

If the output shows `ED25519` or `ECDSA`, you'll need to generate an RSA key
instead:

```bash
ssh-keygen -t rsa -b 4096 -f new_rsa_key
```

### "private key is passphrase-protected" in non-interactive environment

This error occurs when Kanuka detects a passphrase-protected key but cannot
prompt for the passphrase (e.g., running in a script or CI pipeline).

**Options:**

1. **Use an unencrypted key for automation** - Generate a dedicated key
   without a passphrase for CI/CD use.

2. **Use `--private-key-stdin`** - Pipe your key from a secrets manager:
   ```bash
   vault read -field=private_key secret/kanuka | kanuka secrets decrypt --private-key-stdin
   ```

3. **Use a secrets manager** - Store and retrieve the unencrypted key securely:
   ```bash
   op read "op://Vault/Kanuka/private_key" | kanuka secrets decrypt --private-key-stdin
   ```

### "failed to parse private key" or "unsupported key type"

This usually means your key is not an RSA key. To check:

```bash
# Check key type
ssh-keygen -l -f your_key

# Expected output for RSA:
# 4096 SHA256:... your_key (RSA)

# If you see ED25519 or ECDSA, generate an RSA key instead
ssh-keygen -t rsa -b 4096 -f new_rsa_key
```

### Passphrase prompt not appearing

If you're piping a passphrase-protected key via `--private-key-stdin` and the
passphrase prompt doesn't appear, ensure your terminal supports `/dev/tty`
(or `CON` on Windows). The passphrase is read from the terminal device, not
stdin, when stdin is used for the key.

If running in a container or environment without a TTY, consider using an
unencrypted key stored in a secrets manager.

## How do I check who has been accessing secrets?

Use the audit log to see all operations:

```bash
kanuka secrets log
```

This shows all operations with timestamps and user emails. You can filter by
user, operation type, or date range:

```bash
# Filter by user
kanuka secrets log --user alice@example.com

# Filter by operation
kanuka secrets log --operation encrypt,decrypt

# Filter by date
kanuka secrets log --since 2024-01-01
```

See the [audit log guide](/guides/audit-log/) for more details.

## Can I encrypt just one file?

Yes, specify the file as an argument:

```bash
kanuka secrets encrypt .env
```

You can also use glob patterns and directories:

```bash
# Multiple files
kanuka secrets encrypt .env .env.local

# Glob pattern
kanuka secrets encrypt "services/*/.env"

# All .env files in a directory
kanuka secrets encrypt services/api/
```

See the [encryption guide](/guides/encryption/) for more details.

## How do I use Kanuka in a monorepo?

You have two options:

1. **Single store at root** - Initialize once, use selective encryption:
   ```bash
   kanuka secrets init
   kanuka secrets encrypt services/api/.env
   ```

2. **Per-service stores** - Initialize in each service:
   ```bash
   cd services/api && kanuka secrets init
   cd services/web && kanuka secrets init
   ```

See the [monorepo guide](/guides/monorepo/) for detailed workflows.

## How do I check who has access to my project's secrets?

Use the `access` command to see all users with access:

```bash
kanuka secrets access
```

This shows each user's UUID, email, and status (active, pending, or orphan).
For machine-readable output, add the `--json` flag.

See the [access guide](/guides/access/) for more details.

## How often should I rotate encryption keys?

Key rotation frequency depends on your security requirements:

- **After revoking a user** - Automatic, happens as part of revoke
- **Periodic rotation** - Recommended every 3-6 months for high-security projects
- **After suspected compromise** - Immediately

To manually rotate the project's symmetric key:

```bash
kanuka secrets sync
```

To rotate your personal keypair:

```bash
kanuka secrets rotate
```

See the [sync guide](/guides/sync/) and [rotate guide](/guides/rotate/) for details.

## What's the difference between sync and rotate?

| Command | What it rotates | Who is affected |
|---------|-----------------|-----------------|
| `sync` | Project's symmetric key | All users |
| `rotate` | Your personal RSA keypair | Only you |

Use `sync` for project-wide key rotation. Use `rotate` for your personal keypair.

## How do I backup my project's secrets?

Use the export command to create a backup archive:

```bash
kanuka secrets export -o backup.tar.gz
```

This creates a gzip-compressed archive containing all encrypted secrets and
configuration. Private keys and plaintext files are NOT included.

To restore from backup:

```bash
kanuka secrets import backup.tar.gz --replace
```

See the [export guide](/guides/export/) and [import guide](/guides/import/) for details.

## What does the "orphan" status mean?

An orphaned entry is an encrypted symmetric key file (`.kanuka`) that has no
corresponding public key. This inconsistent state can occur when:

- A public key was manually deleted
- A revoke operation was interrupted
- Files were partially restored from backup

To clean up orphaned entries:

```bash
kanuka secrets clean
```

See the [clean guide](/guides/clean/) for more details.

## How do I check if my project is healthy?

Use the doctor command to run health checks:

```bash
kanuka secrets doctor
```

This checks for common issues like missing keys, incorrect permissions,
unencrypted files, and inconsistent state. It provides actionable suggestions
for any issues found.

See the [doctor guide](/guides/doctor/) for details.

## Can I see which .env files need to be encrypted?

Yes, use the status command:

```bash
kanuka secrets status
```

This shows all secret files and their encryption status:
- **up to date** - Encrypted and current
- **stale** - Plaintext modified after encryption
- **not encrypted** - No encrypted version exists
- **encrypted only** - Encrypted file with no plaintext

See the [status guide](/guides/status/) for more details.

