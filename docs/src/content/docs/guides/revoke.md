---
title: Revoking Someone's Access
description: A guide to revoking a user's access to a repo's secrets using Kānuka.
---

You may no longer want someone to have access to the repo. You can revoke their
access using Kānuka.

## Revoking access

Revoking does the opposite of registering. It removes their public key and
their encrypted symmetric key from the repo.

```bash
kanuka secrets revoke --user {username}
```

That's it! Kānuka will revoke their access to the repository's secrets. Commit the
changes to version control.

## Next steps

To learn more about `kanuka secrets revoke`, see the [registration concepts](/concepts/registration) and the [command reference](/reference/references).
