---
title: Removing Packages from Your Environment
description: A guide to removing tools and languages from your Grove environment using Kānuka.
---

Sometimes you need to clean up your environment by removing packages you no longer need. Grove makes this simple while keeping your environment clean and consistent.

:::tip
Removing packages from Grove also removes all their dependencies that aren't needed by other packages. This keeps your environment lean and avoids dependency bloat!
:::

## Removing packages

To remove a package from your environment:

```bash
kanuka grove remove nodejs
kanuka grove remove python3
kanuka grove remove docker
```

That's it! Kānuka will:
- Remove the package from your `devenv.nix` file.
- Update `kanuka.toml` to track the removal.
- Clean up any dependencies that are no longer needed.

## Removing multiple packages

You can remove several packages at once:

```bash
kanuka grove remove nodejs python3 git
```

## Removing language environments

Language environments can be removed just like packages:

```bash
kanuka grove remove typescript
kanuka grove remove rust
kanuka grove remove go
```

## What happens when you remove packages

When you run `kanuka grove remove`, Grove:

1. Updates your `devenv.nix` configuration to remove the package.
2. Updates `kanuka.toml` to track what was removed.
3. The next time you enter your environment, the package won't be available.
4. Dependencies that are no longer needed by any package are automatically cleaned up.

## Checking what's installed

Before removing packages, you might want to see what's currently installed:

```bash
kanuka grove list
```

This shows you all the packages and languages in your current environment.

## Next steps

To learn more about `kanuka grove remove`, see the [package management concepts](/concepts/grove-packages) and the [command reference](/reference/references).

Or, continue reading to learn how to search for new packages to add.