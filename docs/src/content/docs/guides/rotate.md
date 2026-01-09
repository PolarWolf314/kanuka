---
title: Rotating Your Keypair
description: A guide to rotating your personal encryption keypair using Kanuka.
---

The rotate command generates a new keypair for your user account. This is useful
for periodic security rotation or if you suspect your private key may have been
compromised.

## When to rotate

Consider rotating your keypair when:

- **Periodic security hygiene** - Regular key rotation limits exposure
- **Suspected compromise** - If your private key may have been accessed
- **Changing machines** - When moving to a new computer
- **After a security incident** - As part of incident response

## Rotating your keypair

To rotate your keypair:

```bash
kanuka secrets rotate
```

This prompts for confirmation before proceeding:

```
Warning: This will generate a new keypair and replace your current one.
  Your old private key will no longer work for this project.

Do you want to continue? [y/N]: y

Rotating your keypair...
  Generating new keypair...
  Decrypting symmetric key with old private key...
  Re-encrypting symmetric key with new public key...
  Updating public key in project...
  Saving new private key...
Done: Keypair rotated successfully

Your new public key has been added to the project.
Other users do not need to take any action.
```

## What happens during rotation

1. Your current private key decrypts the project's symmetric key
2. A new 4096-bit RSA keypair is generated
3. The symmetric key is re-encrypted with your new public key
4. Your new public key replaces the old one in the project
5. Your new private key is saved to your local key store
6. Your old private key is overwritten

After rotation:
- You can continue to decrypt secrets with no additional steps
- Other users are unaffected - they keep their existing keys
- The project's symmetric key remains the same

## Skipping confirmation

In automated environments, use `--force` to skip the confirmation prompt:

```bash
kanuka secrets rotate --force
```

:::caution
Using `--force` will immediately replace your keypair. Make sure you want to
proceed, as this cannot be undone.
:::

## Rotate examples

```bash
# Rotate with confirmation prompt
kanuka secrets rotate

# Rotate without confirmation (for automation)
kanuka secrets rotate --force
```

## Using with passphrase-protected keys

If your current private key is passphrase-protected, Kanuka will prompt for
the passphrase to decrypt the symmetric key.

When generating the new keypair, you can optionally protect it with a passphrase
as well.

## After rotating

After rotation:

1. **Commit the changes** - Your new public key needs to be shared
2. **Push to remote** - So the team has your updated public key

```bash
git add .kanuka/public_keys/
git commit -m "Rotate keypair for $(whoami)"
git push
```

:::note
Unlike `sync`, key rotation only affects your user. Other team members don't
need to do anything - they continue using their existing keys.
:::

## Rotation vs sync

| Command | What it rotates | Who is affected |
|---------|-----------------|-----------------|
| `rotate` | Your personal keypair | Only you |
| `sync` | Project's symmetric key | All users |

Use `rotate` for your personal key rotation.
Use `sync` to rotate the project-wide encryption key for everyone.

## Next steps

- **[Sync command](/guides/sync/)** - Rotate the project's symmetric key
- **[Access command](/guides/access/)** - View who has access
- **[Doctor command](/guides/doctor/)** - Check project health
