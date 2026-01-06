---
title: Configuration Commands
description: A guide to managing Kanuka configuration settings.
---

Kanuka provides configuration commands for managing user and project settings,
including device names and user information.

## Listing Devices

To see all devices registered in the current project:

```bash
kanuka config list-devices
```

This displays all users and their devices, including device names, UUIDs, and
creation dates.

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

## Device Name Requirements

Device names must:
- Start with an alphanumeric character
- Contain only alphanumeric characters, hyphens, and underscores
- Be unique per user within a project

## Next Steps

- Learn about [creating keys](/guides/create) for a new device
- Learn about [revoking access](/guides/revoke) for compromised devices
- See the [command reference](/reference/references) for all available options
