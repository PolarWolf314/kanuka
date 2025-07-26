---
title: Purging Access
description: A guide to revoking all access to a repo's secrets using Kānuka.
---

Because there is only one symmetric key for all the secrets and all users, in
the event of a security breach you may want to revoke all access _and_ rewrite
version control history to remove all references to Kānuka.

:::caution
This feature doesn't exist yet. Treat this guide as a wishlist of features from
me. 😋
:::

## Purging secrets

Purging secrets is a destructive action and rewrites all version control
history. To do it, run the following command.

```bash
kanuka secrets purge

# Then confirm the safety prompt by typing CONFIRM
```

That's it! Kānuka will revoke all access and purge the version control history
of any references to Kānuka.

## Next steps

To learn more about `kanuka secrets purge`, see the [purge concepts](/concepts/purge) and the [command reference](/reference/references).

Or, continue reading to learn core concepts on how Kānuka works.
