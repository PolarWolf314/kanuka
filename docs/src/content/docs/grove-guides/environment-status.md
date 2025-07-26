---
title: Checking Your Environment Status
description: A guide to viewing the status of your Grove development environment using Kānuka.
---

Understanding the current state of your Grove environment helps you troubleshoot issues and ensure everything is working correctly.

:::tip
Grove status shows you not just what's configured, but what's actually available and working in your environment. This helps catch configuration issues early!
:::

## Checking environment status

To see the status of your current environment:

```bash
kanuka grove status
```

This shows you:
- Whether your environment is properly configured.
- Which packages are available.
- Any configuration issues or warnings.
- AWS authentication status (if applicable).

## Understanding status output

The status command typically shows:

```
Environment: my-project
Status: Ready
Packages: 5 available
Languages: 2 enabled
AWS: Authenticated (expires in 2h 30m)
Channels: Up to date
```

## Status indicators

Grove status uses different indicators:

- **Ready**: Environment is properly configured and available.
- **Not initialized**: No Grove environment found in current directory.
- **Configuration error**: Issues with devenv.nix or kanuka.toml.
- **Missing dependencies**: Required tools (Nix, devenv) not available.

## Checking specific components

You can check specific parts of your environment:

```bash
# Check only AWS authentication
kanuka grove status --auth

# Check only package availability
kanuka grove status --packages

# Check channel status
kanuka grove status --channels
```

## Troubleshooting with status

Common issues the status command helps identify:

- **Missing Nix or devenv**: Shows if prerequisites aren't installed.
- **Outdated channels**: Indicates if your package sources need updating.
- **AWS authentication expired**: Shows when you need to re-authenticate.
- **Configuration conflicts**: Identifies issues in your environment files.

## Status in different directories

Grove status is context-aware:

```bash
# In a Grove-enabled project
cd my-project
kanuka grove status
# Shows: Environment ready

# In a directory without Grove
cd /tmp
kanuka grove status
# Shows: No Grove environment found
```

## Next steps

To learn more about `kanuka grove status`, see the [development environments concepts](/concepts/grove-environments) and the [command reference](/reference/references).

Or, continue reading to learn how to manage package channels.