---
title: Creating Secrets for Access
description: A guide to gaining access to a repo's secrets using Kānuka.
---

Kānuka uses a combination of RSA key pairs and symmetric keys to encrypt and
decrypt files. If you aren't the person who ran `kanuka secrets init`, you
won't have access. You may wish to gain access to a project's secrets.

## Creating your keys

Creating your public key is very easy with Kānuka. You just need to run:

```bash
kanuka secrets create
```

That's it! Kānuka will create your public/private key pair automatically, and
store them securely.

:::tip
Kānuka names the public key based on the system username. If you need to create
your public key again, just add a `--force` flag. **This will override the
existing public key and remove access for that user**.

```bash
kanuka secrets create --force
```

:::

## Requesting access

Only people with access to the secrets can grant that privilege to others.
Someone with access will need to run:

```bash
kanuka secrets register --user {your_username}
```

For more information about granting secrets to others, feel free to read the
[guide on registering secrets](/guides/register), or read the [registration concepts](/concepts/registration) and the [command reference](/reference/references).

:::caution[Note]
We are aware of some rough edges around UX. For example, what if two people
have the same username? Kānuka is still under heavy development, so these
features will come soon. In the meantime, if you have some good ideas, please
[create a GitHub issue](https://github.com/PolarWolf314/kanuka/issues)!
:::

## Next steps

To learn more about `kanuka secrets create`, see the [registration concepts](/concepts/registration) and the [command reference](/reference/references).

Or, continue reading to learn how to give access to a project's secrets which
are managed by Kānuka.
