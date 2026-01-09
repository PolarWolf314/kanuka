---
title: Encrypting Secrets
description: A guide to encrypting your secrets using Kānuka.
---

Environment files hold secrets. A key problem that developers face is that you
_should not_ be committing these files to version control, because that would
mean anybody on the internet can access your secrets!

Kānuka makes it easy to share these secrets in a secure way.

:::tip
If you aren't familiar with `.env` files: A `.env` file is used to store
environment variables—like API keys, database URLs, and secret tokens—in a
simple `KEY=VALUE` format. It helps keep sensitive information out of your code
and makes configuration easier across different environments.
:::

## Encrypting secrets

As long as your project has been [initialised](/guides/project-init), and there
are any file containing `.env` in it (`.env.development`, `.env.production`,
`.env`, etc), you can run the following to encrypt the files:

```bash
kanuka secrets encrypt
```

That's it! Kānuka will automatically encrypt the files, and name the encrypted
secrets the same as the original with `.kanuka` added onto the end. You can now
safely commit these files to your version control.

## Encrypting specific files

By default, `encrypt` processes all `.env` files in your project. You can also
specify exactly which files to encrypt:

```bash
# Single file
kanuka secrets encrypt .env

# Multiple files
kanuka secrets encrypt .env .env.local .env.production

# Glob pattern (quote to prevent shell expansion)
kanuka secrets encrypt "services/*/.env"

# Recursive glob pattern
kanuka secrets encrypt "**/.env.production"

# All files in a directory
kanuka secrets encrypt services/api/
```

This is particularly useful for:

- **Monorepos** - Encrypt only specific services
- **Gradual adoption** - Start with production secrets, add others later
- **CI/CD pipelines** - Encrypt only the files that changed
- **Debugging** - Re-encrypt just one file after modification

See the [monorepo guide](/guides/monorepo/) for detailed workflows.

### Previewing encryption

Use the `--dry-run` flag to preview which files would be encrypted without
making any changes:

```bash
kanuka secrets encrypt --dry-run
```

This is useful for:
- Verifying which `.env` files Kānuka discovered in your project
- Checking file discovery in new projects before committing
- CI/CD pipelines for validation without side effects

## Non-Deterministic Encryption

You may notice that running `kanuka secrets encrypt` produces different output
each time, even when your `.env` file hasn't changed. This is expected behavior
and a security feature.

### Why This Happens

Kānuka uses AES-GCM encryption, which requires a unique nonce (number used once)
for each encryption operation. This nonce is randomly generated, so encrypting
the same plaintext twice produces different ciphertext.

### Why This Matters for Security

If encryption were deterministic, an attacker could:
- Detect when the same secret is reused across files
- Build a dictionary of encrypted values to guess plaintext
- Identify patterns in your secrets

Random nonces prevent these attacks, making your encrypted files more secure.

### Git Workflow Recommendations

Since encrypted files change on each run, you'll see git diffs even when secrets
haven't actually changed. This is normal. We recommend:

1. **Run `encrypt` only when you change secrets** - Don't re-encrypt unnecessarily
2. **Commit encrypted files immediately** - After running `encrypt`, commit the
   `.kanuka` files right away
3. **Don't worry about the diffs** - Different ciphertext for the same plaintext
   is expected and secure

:::tip
If you accidentally run `encrypt` without changing any secrets, you can safely
discard the changes with `git checkout -- *.kanuka` to avoid unnecessary commits.
:::

## Using in CI/CD pipelines

In automated environments where your private key isn't stored on disk, you can
pipe it directly from a secrets manager using the `--private-key-stdin` flag:

```bash
# From HashiCorp Vault
vault read -field=private_key secret/kanuka | kanuka secrets encrypt --private-key-stdin

# From 1Password CLI
op read "op://Vault/Kanuka/private_key" | kanuka secrets encrypt --private-key-stdin

# From AWS Secrets Manager
aws secretsmanager get-secret-value --secret-id kanuka-key --query SecretString --output text | kanuka secrets encrypt --private-key-stdin

# From environment variable
echo "$KANUKA_PRIVATE_KEY" | kanuka secrets encrypt --private-key-stdin
```

This approach:
- Avoids writing sensitive keys to disk
- Works with any secrets manager that can output to stdout
- Keeps your private key out of shell history (the key content isn't in the command)

:::tip
If your private key is passphrase-protected, Kānuka will prompt for the
passphrase via `/dev/tty`, allowing you to pipe the key while still entering
the passphrase interactively.
:::

## Next steps

To learn more about `kanuka secrets encrypt`, see the [encryption concepts](/concepts/encryption) and the [command reference](/reference/references).

Or, continue reading to learn how to decrypt secrets using Kānuka.
