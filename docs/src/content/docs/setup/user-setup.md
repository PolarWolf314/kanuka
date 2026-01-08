---
title: User Setup
description: Setting up your identity for Kanuka.
---

Before you can use Kanuka to manage secrets, you need to set up your user
identity. This is a one-time setup that creates your personal configuration
file, which stores your email, a unique identifier, and your default device
name.

## Automatic Setup

The easiest way to set up your identity is to simply run `kanuka secrets init`
in a project. If you haven't configured your identity yet, Kanuka will
automatically prompt you:

```bash
$ kanuka secrets init
⚠ User configuration not found.

Running initial setup...

Welcome to Kanuka! Let's set up your identity.

Email address: alice@example.com
Display name (optional): Alice Smith
Default device name [MacBook-Pro]: 

✓ User configuration saved to ~/.config/kanuka/config.toml

Your settings:
  Email:   alice@example.com
  Name:    Alice Smith
  Device:  MacBook-Pro
  User ID: 6ba7b810-9dad-11d1-80b4-00c04fd430c8

Initializing project...
✓ Kānuka initialized successfully!
→ Run kanuka secrets encrypt to encrypt your existing .env files
```

## Manual Setup

You can also set up your identity explicitly before initializing any projects:

```bash
kanuka config init
```

This will prompt you for:
- **Email address** (required) - Your identifier across all projects
- **Display name** (optional) - For audit log features
- **Default device name** - Defaults to your computer's hostname

### Non-Interactive Setup

For CI/CD pipelines or scripts, you can provide all values via flags:

```bash
kanuka config init --email alice@example.com --device my-laptop
```

Or with all options:

```bash
kanuka config init --email alice@example.com --name "Alice Smith" --device workstation
```

## What Gets Created

After setup, Kanuka creates a configuration file at `~/.config/kanuka/config.toml`:

```toml
[user]
email = "alice@example.com"
name = "Alice Smith"
uuid = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
default_device_name = "MacBook-Pro"

[projects]
# Project entries are added as you initialize projects
```

This file is personal to you and is never shared with your team.

## Viewing Your Configuration

To see your current user configuration:

```bash
kanuka config show
```

Example output:

```
User Configuration (~/.config/kanuka/config.toml):
  Email:          alice@example.com
  Name:           Alice Smith
  User ID:        6ba7b810-9dad-11d1-80b4-00c04fd430c8
  Default Device: MacBook-Pro

Projects:
  550e8400... -> workstation (my-awesome-project)
```

## Updating Your Configuration

To update your email or other settings:

```bash
kanuka config init --email newemail@example.com
```

Only the fields you provide will be updated; other fields remain unchanged.

## Next Steps

Once your identity is configured, you can:

- [Initialize a project](/guides/project-init/) to start managing secrets
- Learn about the [configuration concepts](/concepts/user-configuration/) in depth
- See all [configuration commands](/guides/config/) available
