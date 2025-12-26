---
title: Listing Your Environment Packages
description: A guide to viewing what's currently installed in your Grove environment using Kānuka.
---

It's useful to see what packages and languages are currently available in your Grove environment. This helps you understand what tools you have and plan any changes.

:::tip
Grove tracks everything you've added, so you can always see exactly what's in your environment without guessing or remembering what you installed!
:::

## Listing your packages

To see what's currently in your environment:

```bash
kanuka grove list
```

This shows you all packages and languages currently managed by Kānuka in your development environment.

## Understanding the output

The list output shows only items that were added through Kānuka commands:

```
✓ Kanuka-managed packages:
  • nodejs_20
  • python3

→ Run kanuka grove enter to use this environment
→ Use kanuka grove remove <item> to remove items
```

When you have no managed items, the command produces no output.

## Checking specific categories

You can filter the output to show specific types of items:

```bash
# List only packages
kanuka grove list --packages-only

# List only languages
kanuka grove list --languages-only
```

## Viewing detailed information

For more detailed information about your environment:

```bash
kanuka grove list --verbose
```

This shows additional details and logging information about the scanning process.

## Comparing with what's available

You can combine listing with searching to plan changes:

```bash
# See what you have
kanuka grove list

# Search for something new
kanuka grove search database

# Add what you need
kanuka grove add postgresql
```

## Checking environment files

You can also check your environment by looking at the configuration files:

- `kanuka.toml` - Shows what Grove has tracked.
- `devenv.nix` - Shows the full environment definition.

## Next steps

To learn more about `kanuka grove list`, see the [package management concepts](/concepts/grove-packages) and the [command reference](/reference/references).

Or, continue reading to learn how to check your environment status.