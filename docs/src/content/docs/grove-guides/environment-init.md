---
title: Setting Up Your First Environment
description: A guide to initializing a development environment using Kānuka Grove.
---

You'll need Nix and devenv installed first. If you don't have them, check out the [installation guide](/getting-started/installation).

:::tip
Grove solves the "it works on my machine" problem by creating development environments that work exactly the same way on every computer, every time. No more spending hours setting up dependencies!
:::

## Creating your environment

To get started with Grove in your project, just run:

```bash
kanuka grove init
```

That's it! Kānuka will create everything you need:
- `devenv.nix` - where your environment is defined
- `devenv.yaml` - configuration for devenv
- `kanuka.toml` - Kānuka's own configuration file
- Updates your `.gitignore` so you don't commit the wrong files

## Adding container support

If you want to build containers from your environment, you can enable that too:

```bash
kanuka grove init --containers
```

## Next steps

To learn more about `kanuka grove init`, see the [development environments concepts](/concepts/grove-environments) and the [command reference](/reference/references).

Or, continue reading to learn how to add packages to your environment.