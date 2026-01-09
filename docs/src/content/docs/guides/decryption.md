---
title: Decrypting Secrets
description: A guide to decrypting your secrets using Kānuka.
---

`.kanuka` files are files which have been encrypted by Kānuka. You may wish to
decrypt these files to get back your original `.env` file.

## Decrypting secrets

As long as the following conditions are met:

1. Your project has been [initialised](/guides/project-init).
2. You have access.
3. There are any file containing `.kanuka` in it (`.env.kanuka`,
   `.env.production.kanuka`, etc).

You can run the following to decrypt the files:

```bash
kanuka secrets decrypt
```

That's it! Kānuka will automatically decrypt the files, and return the original
`.env`, as long as you have access.

## Decrypting specific files

By default, `decrypt` processes all `.kanuka` files in your project. You can
also specify exactly which files to decrypt:

```bash
# Single file
kanuka secrets decrypt .env.kanuka

# Multiple files
kanuka secrets decrypt .env.kanuka .env.local.kanuka

# Glob pattern (quote to prevent shell expansion)
kanuka secrets decrypt "services/*/.env.kanuka"

# Recursive glob pattern
kanuka secrets decrypt "**/.env.production.kanuka"

# All files in a directory
kanuka secrets decrypt services/api/
```

This is particularly useful for:

- **CI/CD pipelines** - Decrypt only the secrets needed for a specific job
- **Monorepos** - Work with only the services you need
- **Debugging** - Decrypt a single file to inspect its contents

See the [monorepo guide](/guides/monorepo/) for detailed workflows.

## Previewing decryption

Use the `--dry-run` flag to preview which files would be decrypted without
making any changes:

```bash
kanuka secrets decrypt --dry-run
```

This shows:
- Which `.kanuka` files would be decrypted
- The target `.env` files that would be created
- Whether any existing `.env` files would be overwritten

This is especially useful to check if you have local `.env` modifications that
would be lost during decryption.

## Using in CI/CD pipelines

In automated environments where your private key isn't stored on disk, you can
pipe it directly from a secrets manager using the `--private-key-stdin` flag:

```bash
# From HashiCorp Vault
vault read -field=private_key secret/kanuka | kanuka secrets decrypt --private-key-stdin

# From 1Password CLI
op read "op://Vault/Kanuka/private_key" | kanuka secrets decrypt --private-key-stdin

# From AWS Secrets Manager
aws secretsmanager get-secret-value --secret-id kanuka-key --query SecretString --output text | kanuka secrets decrypt --private-key-stdin

# From environment variable
echo "$KANUKA_PRIVATE_KEY" | kanuka secrets decrypt --private-key-stdin
```

This approach:
- Avoids writing sensitive keys to disk
- Works with any secrets manager that can output to stdout
- Keeps your private key out of shell history (the key content isn't in the command)

:::tip
If your private key is passphrase-protected, Kanuka will prompt for the
passphrase via `/dev/tty`, allowing you to pipe the key while still entering
the passphrase interactively.
:::

## Next steps

To learn more about `kanuka secrets decrypt`, see the [encryption concepts](/concepts/encryption) and the [command reference](/reference/references).

Or, continue reading to learn how to gain access to a project's secrets which
are managed by Kānuka.
