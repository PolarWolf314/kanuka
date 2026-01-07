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

## Next steps

To learn more about `kanuka secrets encrypt`, see the [encryption concepts](/concepts/encryption) and the [command reference](/reference/references).

Or, continue reading to learn how to decrypt secrets using Kānuka.
