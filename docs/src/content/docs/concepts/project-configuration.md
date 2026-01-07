---
title: Project Configuration
description: Understanding the shared project configuration in Kanuka.
---

Every Kanuka project has a configuration file that tracks the project identity
and all registered users. Unlike your [user configuration](/concepts/user-configuration/),
this file is shared with your team via version control.

## Location

The project configuration is stored at:

```
your-project/
└── .kanuka/
    └── config.toml
```

## File Structure

```toml
[project]
uuid = "550e8400-e29b-41d4-a716-446655440000"
name = "my-awesome-project"

[users]
"6ba7b810-9dad-11d1-80b4-00c04fd430c8" = "alice@example.com"
"8ba7b810-9dad-11d1-80b4-00c04fd430c9" = "bob@company.com"

[devices]
[devices."6ba7b810-9dad-11d1-80b4-00c04fd430c8"]
email = "alice@example.com"
name = "workstation"
created_at = 2025-01-06T10:00:00Z

[devices."8ba7b810-9dad-11d1-80b4-00c04fd430c9"]
email = "bob@company.com"
name = "macbook"
created_at = 2025-01-05T09:00:00Z
```

## Sections Explained

### Project Section

| Field | Description |
|-------|-------------|
| `uuid` | A unique identifier for this project, generated when you run `kanuka secrets init`. |
| `name` | The project name, defaulting to the directory name. |

The project UUID is used to:
- Organize your local keys by project
- Link your user configuration to specific projects
- Ensure uniqueness across all Kanuka projects

### Users Section

Maps user UUIDs to their email addresses:

```toml
[users]
"6ba7b810-..." = "alice@example.com"
"8ba7b810-..." = "bob@company.com"
```

This provides a human-readable way to identify who has access, while the
actual key files use UUIDs for naming.

### Devices Section

Tracks metadata for each registered device:

| Field | Description |
|-------|-------------|
| `email` | The user's email address (for display purposes). |
| `name` | The device name chosen by the user. |
| `created_at` | When this device was registered. |

Note that each user UUID represents a single device. If a user has multiple
devices, they have multiple UUIDs in the project config.

## How Users Are Represented

Each device a user registers gets its own UUID. For example, if Alice has
two devices:

```toml
[users]
"uuid-alice-workstation" = "alice@example.com"
"uuid-alice-laptop" = "alice@example.com"

[devices."uuid-alice-workstation"]
email = "alice@example.com"
name = "workstation"
created_at = 2025-01-06T10:00:00Z

[devices."uuid-alice-laptop"]
email = "alice@example.com"
name = "laptop"
created_at = 2025-01-07T14:30:00Z
```

This design allows:
- Per-device key management
- Revoking a single device without affecting others
- Clear audit trail of when devices were added

## How It's Created

The project configuration is created when you run:

```bash
kanuka secrets init
```

This command:
1. Creates the `.kanuka/` directory structure
2. Generates a project UUID
3. Creates the `config.toml` with your user as the first registered member

## Viewing Project Configuration

```bash
# Show project configuration
kanuka config show --project

# List all devices in the project
kanuka config list-devices

# Show as JSON (for scripts)
kanuka config show --project --json
```

## What Gets Committed

The entire `.kanuka/` directory should be committed to version control,
including:

- `config.toml` - The project configuration
- `public_keys/` - Public keys for all registered users
- `secrets/` - Encrypted symmetric keys for all registered users

This allows your team to:
- See who has access to the project
- Grant access to new team members
- Revoke access when needed

## Relationship to User Configuration

The project and user configurations work together:

| Action | Project Config | User Config |
|--------|---------------|-------------|
| `secrets init` | Created with your user entry | Updated with project entry |
| `secrets create` | Updated with new device | Updated with project entry |
| `secrets register` | Updated with new user/device | (Their config, not yours) |
| `secrets revoke` | User/device removed | (Their config, not yours) |
| `config rename-device` | Device name updated | Your entry updated (if your device) |

## Security Considerations

The project configuration contains:
- User email addresses (visible to anyone with repo access)
- Device names and creation dates
- UUIDs linking to encryption key files

It does **not** contain:
- Private keys (stored locally on each user's machine)
- Actual secrets (stored in encrypted `.env.kanuka` files)
- Symmetric keys (encrypted per-user in `.kanuka/secrets/`)

## Related Configuration

- [User Configuration](/concepts/user-configuration/) - Your personal config
- [Configuration Commands](/guides/config/) - All available config commands
- [File Structure](/concepts/structure/) - Where all Kanuka files are stored
