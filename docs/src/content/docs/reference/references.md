---
title: Command Reference
description: Complete reference for all Kānuka commands and their options.
---

This page provides a comprehensive reference for all Kānuka commands and their options.

## Secrets Management Commands

### `kanuka`

```
Usage:
  kanuka [flags]
  kanuka [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  config      Manage user and project configuration
  help        Help about any command
  secrets     Manage secrets stored in the repository

Flags:
  -h, --help   help for kanuka
```

## Completion

Generate autocompletion scripts for various shells.

### `kanuka completion`

```
Usage:
  kanuka completion [command]

Available Commands:
  bash        Generate the autocompletion script for bash
  fish        Generate the autocompletion script for fish
  powershell  Generate the autocompletion script for powershell
  zsh         Generate the autocompletion script for zsh

Flags:
  -h, --help   help for completion
```

## Secrets Management

Provides encryption, decryption, registration, revocation, and initialization of secrets.

### `kanuka secrets`

```
Usage:
  kanuka secrets [command]

  Available Commands:
  access      List users with access to the project's secrets
  clean       Remove orphaned keys and inconsistent state
  create      Creates and adds your public key, and gives instructions on how to gain access
  decrypt     Decrypts the .env.kanuka file back into .env using your Kānuka key
  doctor      Run health checks on the project
  encrypt     Encrypts the .env file into .env.kanuka using your Kānuka key
  export      Create a backup archive of encrypted secrets
  import      Restore secrets from a backup archive
  init        Initializes the secrets store
  register    Registers a new user to be given access to the repository's secrets
  revoke      Revokes access to the secret store
  rotate      Rotate your personal keypair
  status      Show encryption status of secret files
  sync        Re-encrypt all secrets with a new symmetric key

Flags:
  -h, --help   help for secrets
```

### `kanuka secrets create`

Creates and adds your public key, and gives instructions on how to gain access.

```
Usage:
  kanuka secrets create [flags]

Flags:
  -f, --force     force key creation
  -h, --help      help for create
  -v, --verbose   enable verbose output
```

### `kanuka secrets decrypt`

Decrypts the `.env.kanuka` file back into `.env` using your Kānuka key.

```
Usage:
  kanuka secrets decrypt [flags]

Flags:
      --dry-run   preview decryption without making changes
  -h, --help      help for decrypt
  -v, --verbose   enable verbose output
```

**Examples:**

```bash
# Preview which files would be decrypted
kanuka secrets decrypt --dry-run

# Decrypt all .kanuka files
kanuka secrets decrypt
```

### `kanuka secrets encrypt`

Encrypts the `.env` file into `.env.kanuka` using your Kānuka key.

```
Usage:
  kanuka secrets encrypt [flags]

Flags:
      --dry-run   preview encryption without making changes
  -h, --help      help for encrypt
  -v, --verbose   enable verbose output
```

**Examples:**

```bash
# Preview which files would be encrypted
kanuka secrets encrypt --dry-run

# Encrypt all .env files
kanuka secrets encrypt
```

### `kanuka secrets init`

Initializes the secrets store.

```
Usage:
  kanuka secrets init [flags]

Flags:
  -h, --help      help for init
  -v, --verbose   enable verbose output
```

### `kanuka secrets register`

Registers a new user to be given access to the repository's secrets.

```
Usage:
  kanuka secrets register [flags]

Flags:
      --dry-run                  preview registration without making changes
  -f, --file string              the path to a custom public key — will add public key to the project
      --force                    skip confirmation when updating existing user
  -h, --help                     help for register
      --private-key-stdin        read private key from stdin
      --pubkey string            OpenSSH or PEM public key content to be saved with the specified username
  -u, --user string              username to register for access
  -v, --verbose                  enable verbose output
```

**Examples:**

```bash
# Preview what would be created
kanuka secrets register --user alice@example.com --dry-run

# Register a user by email
kanuka secrets register --user alice@example.com

# Re-register existing user (skip confirmation)
kanuka secrets register --user alice@example.com --force

# Register using a public key file
kanuka secrets register --file path/to/key.pub
```

### `kanuka secrets revoke`

Revokes access to the secret store.

```
Usage:
  kanuka secrets revoke [flags]

Flags:
  -d, --device string   revoke a specific device only
      --dry-run         preview revocation without making changes
  -f, --file string     path to the .kanuka file to revoke
  -h, --help            help for revoke
  -u, --user string     user email to revoke
  -v, --verbose         enable verbose output
  -y, --yes             skip confirmation prompts
```

**Examples:**

```bash
# Preview what would be revoked
kanuka secrets revoke --user alice@example.com --dry-run

# Revoke all devices for a user
kanuka secrets revoke --user alice@example.com

# Revoke a specific device
kanuka secrets revoke --user alice@example.com --device old-laptop --dry-run

# Revoke by file path
kanuka secrets revoke --file .kanuka/secrets/uuid.kanuka
```

### `kanuka secrets sync`

Re-encrypts all secrets with a newly generated symmetric key.

```
Usage:
  kanuka secrets sync [flags]

Flags:
      --dry-run             preview sync without making changes
  -h, --help                help for sync
      --private-key-stdin   read private key from stdin
  -v, --verbose             enable verbose output
```

**Examples:**

```bash
# Preview what would happen
kanuka secrets sync --dry-run

# Rotate encryption key and re-encrypt all secrets
kanuka secrets sync

# Use in CI/CD with piped private key
echo "$KANUKA_PRIVATE_KEY" | kanuka secrets sync --private-key-stdin
```

### `kanuka secrets rotate`

Rotates your personal keypair, generating a new RSA key pair and updating your access.

```
Usage:
  kanuka secrets rotate [flags]

Flags:
      --force               skip confirmation prompt
  -h, --help                help for rotate
      --private-key-stdin   read private key from stdin
  -v, --verbose             enable verbose output
```

**Examples:**

```bash
# Rotate keypair with confirmation
kanuka secrets rotate

# Rotate keypair without confirmation
kanuka secrets rotate --force
```

### `kanuka secrets access`

Lists all users who have access to the project's secrets.

```
Usage:
  kanuka secrets access [flags]

Flags:
  -h, --help      help for access
      --json      output in JSON format
  -v, --verbose   enable verbose output
```

**Examples:**

```bash
# View all users with access
kanuka secrets access

# JSON output for scripting
kanuka secrets access --json
```

### `kanuka secrets status`

Shows the encryption status of all secret files in the project.

```
Usage:
  kanuka secrets status [flags]

Flags:
  -h, --help      help for status
      --json      output in JSON format
  -v, --verbose   enable verbose output
```

**Examples:**

```bash
# View status of all secret files
kanuka secrets status

# JSON output for scripting
kanuka secrets status --json
```

### `kanuka secrets clean`

Removes orphaned keys and inconsistent state.

```
Usage:
  kanuka secrets clean [flags]

Flags:
      --dry-run     preview cleanup without making changes
      --force       skip confirmation prompt
  -h, --help        help for clean
  -v, --verbose     enable verbose output
```

**Examples:**

```bash
# Preview what would be cleaned
kanuka secrets clean --dry-run

# Clean with confirmation
kanuka secrets clean

# Clean without confirmation
kanuka secrets clean --force
```

### `kanuka secrets doctor`

Runs health checks on the project and provides actionable suggestions.

```
Usage:
  kanuka secrets doctor [flags]

Flags:
  -h, --help      help for doctor
      --json      output in JSON format
  -v, --verbose   enable verbose output
```

**Examples:**

```bash
# Run all health checks
kanuka secrets doctor

# JSON output for CI/CD
kanuka secrets doctor --json
```

**Exit codes:**
- `0` - All checks passed
- `1` - Warnings found
- `2` - Errors found

### `kanuka secrets export`

Creates a backup archive of encrypted secrets.

```
Usage:
  kanuka secrets export [flags]

Flags:
  -h, --help            help for export
  -o, --output string   output file path (default: kanuka-secrets-YYYY-MM-DD.tar.gz)
  -v, --verbose         enable verbose output
```

**Examples:**

```bash
# Export with default filename
kanuka secrets export

# Export to custom path
kanuka secrets export -o /backups/project-secrets.tar.gz
```

### `kanuka secrets import`

Restores secrets from a backup archive.

```
Usage:
  kanuka secrets import [archive] [flags]

Flags:
      --dry-run     preview import without making changes
  -h, --help        help for import
      --merge       add new files, keep existing
      --replace     delete existing, use backup
  -v, --verbose     enable verbose output
```

**Examples:**

```bash
# Import with interactive prompt
kanuka secrets import backup.tar.gz

# Merge new files, keep existing
kanuka secrets import backup.tar.gz --merge

# Replace all with backup
kanuka secrets import backup.tar.gz --replace

# Preview import
kanuka secrets import backup.tar.gz --dry-run
```

## Configuration Management

Provides commands for managing user and project configuration settings.

### `kanuka config`

```
Usage:
  kanuka config [command]

Available Commands:
  init            Initialize your user configuration
  list-devices    List all devices in the project
  rename-device   Rename a device in the project
  set-device-name Set your device name for a project
  show            Display current configuration

Flags:
  -d, --debug     enable debug output
  -h, --help      help for config
  -v, --verbose   enable verbose output
```

### `kanuka config init`

Sets up your Kānuka user identity. Creates or updates your user configuration file at `~/.config/kanuka/config.toml`.

```
Usage:
  kanuka config init [flags]

Flags:
      --device string   default device name (defaults to hostname)
  -e, --email string    your email address
  -h, --help            help for init
  -n, --name string     your display name (optional)

Global Flags:
  -d, --debug     enable debug output
  -v, --verbose   enable verbose output
```

**Examples:**

```bash
# Interactive setup
kanuka config init

# Non-interactive setup
kanuka config init --email alice@example.com --device macbook

# With all options
kanuka config init --email alice@example.com --name "Alice Smith" --device workstation
```

### `kanuka config show`

Displays the current Kānuka configuration. By default, shows user configuration. Use `--project` to show project configuration.

```
Usage:
  kanuka config show [flags]

Flags:
  -h, --help      help for show
      --json      output in JSON format
  -p, --project   show project configuration instead of user configuration

Global Flags:
  -d, --debug     enable debug output
  -v, --verbose   enable verbose output
```

**Examples:**

```bash
# Show user configuration
kanuka config show

# Show project configuration (must be in a project directory)
kanuka config show --project

# Output in JSON format
kanuka config show --json
```

### `kanuka config set-device-name`

Sets your preferred device name for a project in your local user configuration. This name is used when you create keys for a project.

```
Usage:
  kanuka config set-device-name [device-name] [flags]

Flags:
  -h, --help                  help for set-device-name
      --project-uuid string   project UUID (defaults to current project)

Global Flags:
  -d, --debug     enable debug output
  -v, --verbose   enable verbose output
```

**Examples:**

```bash
# Set device name for the current project
kanuka config set-device-name my-laptop

# Set device name for a specific project
kanuka config set-device-name --project-uuid 550e8400-e29b-41d4-a716-446655440000 workstation
```

### `kanuka config rename-device`

Renames a device in the project configuration. You must specify the user email whose device you want to rename.

```
Usage:
  kanuka config rename-device [new-name] [flags]

Flags:
  -h, --help              help for rename-device
      --old-name string   old device name (required if user has multiple devices)
  -u, --user string       user email (required)

Global Flags:
  -d, --debug     enable debug output
  -v, --verbose   enable verbose output
```

**Examples:**

```bash
# Rename the only device for a user
kanuka config rename-device --user alice@example.com new-laptop

# Rename a specific device when user has multiple
kanuka config rename-device --user alice@example.com --old-name macbook personal-macbook
```

### `kanuka config list-devices`

Lists all devices registered in the project configuration.

```
Usage:
  kanuka config list-devices [flags]

Flags:
  -h, --help          help for list-devices
  -u, --user string   filter by user email

Global Flags:
  -d, --debug     enable debug output
  -v, --verbose   enable verbose output
```

**Examples:**

```bash
# List all devices in the project
kanuka config list-devices

# List devices for a specific user
kanuka config list-devices --user alice@example.com
```

## Shell Completion Setup

Use `kanuka completion [shell]` to generate completion scripts for your preferred shell:

- **Bash**: `kanuka completion bash`
- **Zsh**: `kanuka completion zsh`
- **Fish**: `kanuka completion fish`
- **PowerShell**: `kanuka completion powershell`

Refer to each sub-command's help for details on how to use the generated script.
