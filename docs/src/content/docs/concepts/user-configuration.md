---
title: User Configuration
description: Understanding your personal Kanuka configuration.
---

Kanuka stores your personal identity and preferences in a user configuration
file. This file is local to your machine and is never shared with others.

## Location

Your user configuration is stored at:

| Platform | Location                                           |
| -------- | -------------------------------------------------- |
| Linux    | `~/.config/kanuka/config.toml`                     |
| macOS    | `~/Library/Application Support/kanuka/config.toml` |
| Windows  | `%APPDATA%\kanuka\config.toml`                     |

## File Structure

```toml
[user]
email = "alice@example.com"
name = "Alice Smith"
uuid = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
default_device_name = "MacBook-Pro"

[projects]
[projects."550e8400-e29b-41d4-a716-446655440000"]
device_name = "workstation"
project_name = "my-awesome-project"

[projects."7ba7b810-9dad-11d1-80b4-00c04fd430c8"]
device_name = "laptop"
project_name = "another-project"
```

## Fields Explained

### User Section

| Field                 | Required | Description                                                                                                                      |
| --------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------- |
| `email`               | Yes      | Your email address, used as your identifier across all projects. This is how teammates recognize you when granting access.       |
| `name`                | No       | Your display name for audit log features (future functionality).                                                                 |
| `uuid`                | Yes      | A unique identifier generated automatically. This links your identity across projects without exposing your email in file names. |
| `default_device_name` | No       | The default name for your devices when creating keys. Defaults to your computer's hostname.                                      |

### Projects Section

The `[projects]` section maps project UUIDs to your device information for
each project you've joined. Each entry contains:

| Field          | Description                                        |
| -------------- | -------------------------------------------------- |
| `device_name`  | The device name you use for this specific project. |
| `project_name` | A human-readable name for the project.             |

This allows you to use different device names for different projects if needed.

## How It's Created

Your user configuration is created when you run:

1. **`kanuka config init`** - Explicit setup of your identity
2. **`kanuka secrets init`** - Automatically prompts for setup if not configured

During setup, you provide:

- Your email address (required)
- A display name (optional)
- A default device name (defaults to hostname)

Kanuka generates a UUID automatically to uniquely identify you.

## Why UUIDs?

Kanuka uses UUIDs instead of emails for file naming because:

1. **Uniqueness** - Guaranteed unique identifiers avoid naming conflicts
2. **Flexibility** - You can change your email without renaming files

The email is stored in the project configuration to provide a human-readable
way to identify users.

## Multiple Devices

If you work on the same project from multiple computers, each device gets its
own:

- RSA key pair (stored locally)
- Entry in the project configuration
- Encrypted symmetric key

Your user configuration tracks which device name you use for each project.

## Viewing Your Configuration

```bash
# Show user configuration
kanuka config show

# Show as JSON (for scripts)
kanuka config show --json
```

## Updating Your Configuration

```bash
# Update email
kanuka config init --email newemail@example.com

# Update device name
kanuka config init --device new-laptop

# Update multiple fields
kanuka config init --email alice@company.com --name "Alice Smith" --device macbook
```

Only the fields you specify are updated; other fields remain unchanged.

## Related Configuration

- [Project Configuration](/concepts/project-configuration/) - The shared project config
- [Configuration Commands](/guides/config/) - All available config commands
- [File Structure](/concepts/structure/) - Where all Kanuka files are stored
