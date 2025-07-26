---
title: Registering Other Users
description: A guide to giving access to a repo's secrets using Kānuka.
---

Kānuka uses a symmetric key to encrypt and decrypt files, and uses an RSA key
pair to encrypt the symmetric key. Any user who can decrypt the symmetric key
is able to grant access to others. You may wish to do such a thing.

## Granting access to others

Granting other users access is very easy with Kānuka. Provided they have
committed their public key, you just need to run:

```bash
kanuka secrets register --user {their_username}
```

That's it! Kānuka will create their encrypted symmetric key. Commit this to
version control and they will have access.

## Using a custom public key

If you wish to directly pass in a public key, there are two ways.

:::tip
Kānuka accepts both OpenSSH and PEM formats for RSA key pairs. Future work
includes having custom key types.
:::

### Passing in the path to a key

You are able to pass in the path to any public key, and Kānuka will handle
adding it and giving access.

```bash
kanuka secrets register --file path/to/pubkey
```

### Passing in the contents of a key

You are also able to pass in the contents of any public key, and Kānuka will
handle it.

```bash
# Pasting in the contents of an OpenSSH format public key
kanuka secrets register --pubkey "ssh-rsa AAAAB3NzaC1..." --user {username}

# You can also pass in the key dynamically
kanuka secrets register --pubkey "$(cat path/to/pubkey)" --user {username}

# Or you could use shell variables
PUBKEY_CONTENT=$(cat path/to/pubkey)
kanuka secrets register --pubkey "$PUBKEY_CONTENT" --user {username}
```

:::tip
Because passing in the contents of the file inherently provides no information
about the name of the file, the `--user` flag is required so Kānuka knows what
to name the public key.
:::

:::caution[Note]
We are aware of some rough edges around UX. For example, what if two people
have the same username? Kānuka is still under heavy development, so these
features will come soon. In the meantime, if you have some good ideas, please
[create a GitHub issue](https://github.com/PolarWolf314/kanuka/issues)!
:::

## Next steps

To learn more about `kanuka secrets register`, see the [registration concepts](/concepts/registration) and the [command reference](/reference/references).

Or, continue reading to learn how to remove someone's access to a project's
secrets.
