---
title: Configuration Commands
description: A guide to managing Kānuka configuration settings.
---

Kānuka provides configuration commands for managing user and project settings,
including device names and user information. To understand how configuration
works at a deeper level, see the [configuration concepts](/concepts/configuration/)
page.

## First-Time Setup

When you first use Kānuka, you need to set up your user identity. This is done
automatically when you run `kanuka secrets init`, but you can also do it
explicitly:

```bash
kanuka config init
```

This will prompt you for:
- **Email address** (required) - Your identifier across all projects.
- **Display name** (optional) - For audit log features.
- **Default device name** - Defaults to your computer's hostname.

For non-interactive setup (useful in CI/CD or scripts):

```bash
kanuka config init --email alice@example.com --device my-laptop
```

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

To see the project configuration (must be in a project directory):

```bash
kanuka config show --project
```

For JSON output (useful for scripts):

```bash
kanuka config show --json
kanuka config show --project --json
```

## Listing Devices

To see all devices registered in the current project:

```bash
kanuka config list-devices
```

This displays all users and their devices, including device names, UUIDs, and
creation dates.

Example output:

```
Devices in project 'my-awesome-project':

alice@example.com (6ba7b810...)
  - workstation (created: Jan 6, 2025)
  - laptop (created: Jan 7, 2025)

bob@company.com (8ba7b810...)
  - macbook (created: Jan 5, 2025)
```

To filter by a specific user:

```bash
kanuka config list-devices --user alice@example.com
```

## Setting Your Device Name

You can set your preferred device name for a project. This name is stored in
your local user configuration and will be used when you create keys.

```bash
kanuka config set-device-name my-laptop
```

This sets the device name for the current project. To set a device name for a
specific project by UUID:

```bash
kanuka config set-device-name --project-uuid 550e8400-e29b-41d4-a716-446655440000 workstation
```

## Renaming Devices

To rename a device in the project configuration (requires project access):

```bash
# Rename a user's only device
kanuka config rename-device --user alice@example.com new-laptop

# Rename a specific device when the user has multiple
kanuka config rename-device --user alice@example.com --old-name macbook personal-macbook
```

The `--old-name` flag is required when a user has multiple devices registered.

When you rename your own device, Kānuka automatically updates both the project
configuration and your local user configuration to keep them in sync.

## Device Name Requirements

Device names must:
- Start with an alphanumeric character
- Contain only alphanumeric characters, hyphens, and underscores
- Be unique per user within a project

## Common Workflows

### Adding a New Device

When you want to access a project from a new computer:

1. Clone the repository on your new device.
2. Run `kanuka config init` to set up your identity (use the same email).
3. Set a unique device name for this project:
   ```bash
   kanuka config set-device-name work-laptop
   ```
4. Run `kanuka secrets create` to generate keys for this device.
5. Ask a teammate to register your new device:
   ```bash
   kanuka secrets register --user your@email.com
   ```
6. Pull the latest changes and decrypt:
   ```bash
   git pull && kanuka secrets decrypt
   ```

### Checking Who Has Access

To see all users and devices with access to the project:

```bash
kanuka config list-devices
```

Or view the full project configuration:

```bash
kanuka config show --project
```

### Cleaning Up Old Devices

If you no longer use a device, you should revoke its access:

```bash
kanuka secrets revoke --user your@email.com --device old-laptop
```

See the [revoke guide](/guides/revoke/) for more details.

## Next Steps

- Learn about [creating keys](/guides/create) for a new device
- Learn about [revoking access](/guides/revoke) for compromised devices
- Understand the [configuration concepts](/concepts/configuration/)
- See the [command reference](/reference/references) for all available options
