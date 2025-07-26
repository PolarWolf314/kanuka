---
title: Adding Packages to Your Environment
description: A guide to adding tools and languages to your Grove environment using Kānuka.
---

Adding packages to your Grove environment is really simple. You get access to thousands of packages from the Nix ecosystem without any of the usual dependency headaches.

:::tip
Grove automatically handles all dependencies for you. When you add Node.js, you automatically get npm, all required system libraries, and proper environment variables - no manual setup required!
:::

## Adding packages

As long as your project has been [initialized](/grove-guides/environment-init), you can add any package you need:

```bash
kanuka grove add nodejs
kanuka grove add python3
kanuka grove add git
kanuka grove add docker
kanuka grove add awscli2
```

That's it! Kānuka will add these packages to your environment and update your configuration files automatically.

## Adding specific versions

Sometimes you need a specific version of a package. You can do that too:

```bash
kanuka grove add nodejs_18  # Node.js version 18
kanuka grove add python39   # Python 3.9
kanuka grove add go_1_19    # Go version 1.19
```

## Using different channels

You can also choose which channel (version source) to use:

```bash
kanuka grove add nodejs --channel stable      # From stable channel
kanuka grove add python3 --channel unstable   # From unstable channel
```

## Next steps

To learn more about `kanuka grove add`, see the [package management concepts](/concepts/grove-packages) and the [command reference](/reference/references).

Or, continue reading to learn how to enter your development environment.