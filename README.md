# Kānuka

Kānuka is a powerful command-line tool written in Go for secure secrets management in your projects. It provides a simple yet robust interface for encrypting and decrypting environment files using industry-standard cryptography.

## Documentation

**For complete documentation, examples, and guides, visit our [official documentation site](https://kanuka.guo.nz).**

The documentation includes:

- Detailed installation guides
- Step-by-step tutorials
- Configuration examples
- Best practices
- API reference

## What's with the name?

Kānuka (_Kunzea ericoides_) is a tree that is endemic to Aotearoa New Zealand. It is a robust species, critical to restoring wildlife destroyed by fire as it quickly propagates and regenerates the land. Its leaves have a characteristically soft touch, and it's one of few plants that can survive the heat of geothermal features.

It is fast, resilient, yet pleasant to touch. This is the vision of Kānuka.

## Features

- **Secure Secret Management**: Store and encrypt environment variables using industry-standard encryption (AES-256 and RSA-2048)
- **Team Collaboration**: Register and manage team member access to shared secrets
- **Key Rotation**: Rotate encryption keys with automatic re-encryption of all secrets
- **Access Control**: View who has access, revoke users securely with full key rotation
- **Audit Trail**: Track who performed what operations and when with the audit log
- **Selective Encryption**: Encrypt specific files, directories, or use glob patterns
- **Monorepo Support**: Flexible workflows for managing secrets in monorepos
- **Health Checks**: Run diagnostics to detect configuration issues and inconsistent state
- **Backup & Recovery**: Export and import encrypted secrets for disaster recovery
- **User-friendly Interface**: Simple commands for managing secrets across your team
- **Cross-platform Support**: Works on Linux, macOS, and Windows
- **Shell Autocompletion**: Supports bash, zsh, fish, and PowerShell

## Prerequisites

- Go 1.21 or later (for building from source)
- Git (for version control)

## Installation

### Using Go

The recommended way to install Kānuka is using Go:

```bash
go install github.com/PolarWolf314/kanuka@latest
```

Make sure your Go binaries directory is in your PATH:

- **Linux**: Add `export PATH=$HOME/go/bin:$PATH` to your `~/.bashrc`
- **macOS**: Add `export PATH=$HOME/go/bin:$PATH` to your `~/.zshrc`
- **Windows**: Add `%USERPROFILE%\go\bin` to your user environment variables

### Using GitHub Releases

You can also download pre-built binaries from the [GitHub Releases page](https://github.com/PolarWolf314/kanuka/releases).

## Quick Start

1. **Initialize Secrets Store**:

   ```bash
   kanuka secrets init
   ```

2. **Create Your Encryption Keys**:

   ```bash
   kanuka secrets create
   ```

3. **Encrypt Your Secrets**:

   ```bash
   kanuka secrets encrypt
   ```

4. **Register a Team Member**:

   ```bash
   kanuka secrets register --user username
   ```

## Commands

### Secrets Commands

- `kanuka secrets init`: Initialize a new secrets store
- `kanuka secrets create`: Create new encryption keys
- `kanuka secrets encrypt [files...]`: Encrypt .env files (all files if none specified)
- `kanuka secrets decrypt [files...]`: Decrypt .kanuka files (all files if none specified)
- `kanuka secrets register --user <email>`: Register a new user
- `kanuka secrets revoke --user <email>`: Revoke a user's access
- `kanuka secrets sync`: Rotate encryption key and re-encrypt all secrets
- `kanuka secrets rotate`: Rotate your personal keypair
- `kanuka secrets access`: List users with access to secrets
- `kanuka secrets status`: Show encryption status of secret files
- `kanuka secrets clean`: Remove orphaned keys and inconsistent state
- `kanuka secrets doctor`: Run health checks on the project
- `kanuka secrets log`: View audit log of operations
- `kanuka secrets export`: Create a backup archive of encrypted secrets
- `kanuka secrets import <archive>`: Restore secrets from a backup archive

### Configuration Commands

- `kanuka config list-devices`: List all devices in project
- `kanuka config set-default-device <name>`: Set your default device name for new projects
- `kanuka config set-project-device <name>`: Set your device name for an existing project

### General Commands

- `kanuka completion <shell>`: Generate autocompletion script for your shell
- `kanuka --help`: Show help information
- `kanuka <command> --help`: Show help for a specific command

## How It Works

Kānuka uses a hybrid encryption approach for secure secrets management:

1. A symmetric AES-256 key is used to encrypt your project secrets
2. Each user's RSA-2048 key pair is used to encrypt/decrypt the symmetric key
3. Public keys are stored in the project repository
4. Private keys are stored securely on each user's machine

This approach allows team members to securely share the same secrets without exposing sensitive information.

## Project Structure

Kānuka stores secrets-related files in a `.kanuka` folder at the root of your project:

```
project/
├── .env                  # Your secrets (should be in .gitignore)
├── .env.kanuka           # Your secrets, encrypted by Kānuka
└── .kanuka/
    ├── public_keys/
    │   ├── user_1.pub    # Public keys for each user
    │   └── user_2.pub
    └── secrets/
        ├── user_1.kanuka # Encrypted symmetric key for each user
        └── user_2.kanuka
```

### User Data Storage

User-specific private keys are stored in your system's data directory:

- Linux/macOS: `~/.local/share/kanuka/keys/`
- Windows: `%APPDATA%\kanuka\keys\`

## Building from Source

To build Kānuka from source:

```bash
# Clone the repository
git clone https://github.com/PolarWolf314/kanuka.git
cd kanuka

# Build the binary
go build -o kanuka

# Run the binary
./kanuka
```

## Running Tests

```bash
# Run all tests
go test ./test/...

# Run tests with verbose output
go test -v ./test/...

# Run specific command categories
go test ./test/integration/init/...
go test ./test/integration/create/...
go test ./test/integration/register/...
go test ./test/integration/encrypt/...
go test ./test/integration/decrypt/...
go test ./test/integration/revoke/...
go test ./test/integration/sync/...
go test ./test/integration/access/...
```

## Shell Autocompletion

Kānuka supports shell autocompletion for bash, zsh, fish, and PowerShell. Run `kanuka completion [shell]` to generate the appropriate completion script.

## Contributing

Contributions are welcome! Feel free to submit issues or pull requests.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
