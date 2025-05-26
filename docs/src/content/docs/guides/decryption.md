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

## Next steps

To learn more about `kanuka secrets decrypt`, see the [secrets decryption
page]() and the [command reference]().

Or, continue reading to learn how to gain access to a project's secrets which
are managed by Kānuka.
