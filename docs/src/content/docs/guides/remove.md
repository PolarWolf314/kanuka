---
title: Removing Someone's Access
description: A guide to revoking a user's access to a repo's secrets using Kānuka.
---

You may no longer want someone to have access to the repo. You can revoke their
access using Kānuka.

:::caution
This feature doesn't exist yet. Treat this guide as a wishlist of features from
me. 😋
:::

## Removing access

Removing does the opposite of registering. It removes their public key and
their encrypted symmetric key from the repo.

```bash
kanuka secrets remove --user {username}
```

That's it! Kānuka will revoke their access to the repository's secrets. Commit the
changes to version control.

## Next steps

To learn more about `kanuka secrets remove`, see the [registration concepts](/concepts/registration) and the [command reference](/reference/references).

Or, continue reading to learn how to purge all access to a project's secrets in
the case of a security breach.
