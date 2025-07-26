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

This shows you detailed information about your Grove environment including project details, managed packages, environment health, and helpful diagnostics.

## Understanding status output

The status command shows a comprehensive overview like this:

```
═══ Kanuka Grove Status ═══

Project Information
   ✓ Grove project: my-project
   → Project ID: b309d950...
   → Location: /path/to/project

Configuration Files
   ✓ kanuka.toml
   ✓ devenv.nix
   ! .env (not found)

Managed Items
   ✓ Packages (2):
     • nodejs_20
     • python3
   ! No managed languages

Environment Health
   ✓ Nix package manager
   ✓ devenv (devenv 1.8.0 (aarch64-darwin))
   ! AWS SSO (not configured)
   → Configure: Configure AWS SSO in ~/.aws/config

Container Support
   ! Container support not initialized
   → Initialize: kanuka grove container init
   → Or use: kanuka grove init --containers

Next Steps
   → Enter environment: kanuka grove enter
   → View managed items: kanuka grove list
```

## Status indicators

Grove status uses different indicators:

- **Ready**: Environment is properly configured and available.
- **Not initialized**: No Grove environment found in current directory.
- **Configuration error**: Issues with devenv.nix or kanuka.toml.
- **Missing dependencies**: Required tools (Nix, devenv) not available.

## Compact status view

For a shorter summary, you can use the compact flag:

```bash
kanuka grove status --compact
```

This provides a condensed view of your environment status without the detailed breakdown.

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