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

## Next steps

To learn more about `kanuka secrets encrypt`, see the [secrets encryption
page]() and the [command reference]().

Or, continue reading to learn how to decrypt secrets using Kānuka.
