---
title: Entering Your Development Environment
description: A guide to entering and using your Grove development environment.
---

Once you've [initialized your environment](/grove-guides/environment-init) and [added some packages](/grove-guides/adding-packages), you'll want to actually use it. Grove makes this really simple.

:::tip
Grove environments are completely isolated from your system. This means you can have Python 3.9 in one project and Python 3.11 in another, without any conflicts!
:::

## Entering your environment

To start using your development environment, just run:

```bash
kanuka grove enter
# or use the shortcut
kanuka dev
```

This will drop you into a clean shell where all your packages are available and ready to use.

## Checking everything works

You can verify your environment is working by checking the versions:

```bash
# Inside your grove environment
node --version
python --version
go version
```

## Using AWS authentication

If you're working with AWS, Grove can handle authentication for you:

```bash
kanuka grove enter --auth
```

This will prompt you for AWS SSO details and authenticate you automatically.

## Using named environments

You can also use different environment configurations:

```bash
kanuka grove enter --env production
kanuka grove enter --env testing
```

## Next steps

To learn more about `kanuka grove enter`, see the [development environments concepts](/concepts/grove-environments) and the [command reference](/reference/references).

Or, continue reading to learn how to manage your environment.