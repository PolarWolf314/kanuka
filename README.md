# Kānuka

Kānuka is a powerful command-line tool written in Go for managing development environments and secrets in your projects. It provides a unified interface for package management using the Nix ecosystem, container management, and secure secrets handling.

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

### Grove (Development Environment Management)

- **Package Management**: Add, remove, and search for packages using the Nix ecosystem
- **Development Shell**: Enter reproducible development environments with `devenv.nix`
- **Channel Management**: Manage nixpkgs channels for different package versions
- **Container Support**: Build and manage OCI containers from your development environment
- **Environment Status**: Monitor your development environment configuration

### Secrets Management

- **Secure Secret Management**: Store and encrypt environment variables using industry-standard encryption (AES-256 and RSA-2048)
- **Team Collaboration**: Register and manage team member access to shared secrets
- **User-friendly Interface**: Simple commands for managing secrets across your team

### General

- **Cross-platform Support**: Works on Linux, macOS, and Windows
- **Shell Autocompletion**: Supports bash, zsh, fish, and PowerShell
- **Unified Interface**: Single tool for both development environment and secrets management

## Prerequisites

- Go 1.21 or later (for building from source)
- Git (for version control)
- Nix package manager (for Grove development environment features)
- Docker (optional, for container management features)

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

### Getting Started with Grove (Development Environment)

1. **Check Installation**:

   ```bash
   kanuka
   ```

2. **Initialize a Development Environment**:

   ```bash
   kanuka grove init
   ```

3. **Add Packages to Your Environment**:

   ```bash
   kanuka grove add nodejs python3
   ```

4. **Enter the Development Shell**:

   ```bash
   kanuka grove enter
   # or use the shorthand alias:
   kanuka dev
   ```

5. **Check Environment Status**:
   ```bash
   kanuka grove status
   ```

### Getting Started with Secrets Management

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

### Grove Commands (Development Environment)

- `kanuka grove init`: Initialize a development environment with devenv.nix
- `kanuka grove add <package>`: Add a package to the development environment
- `kanuka grove remove <package>`: Remove a package from the development environment
- `kanuka grove list`: Show all Kanuka-managed packages and languages
- `kanuka grove search <query>`: Search nixpkgs for packages
- `kanuka grove enter`: Enter the development shell environment
- `kanuka grove status`: Show development environment status
- `kanuka dev`: Shorthand alias for `kanuka grove enter`

#### Channel Management

- `kanuka grove channel list`: Show all configured nixpkgs channels
- `kanuka grove channel add <name> <url>`: Add a new nixpkgs channel
- `kanuka grove channel remove <name>`: Remove a nixpkgs channel
- `kanuka grove channel show <name>`: Show detailed information about a channel
- `kanuka grove channel pin <name> <commit>`: Pin a channel to a specific commit
- `kanuka grove channel update`: Update channels to their latest versions

#### Container Management

- `kanuka grove container init`: Initialize container support
- `kanuka grove container build`: Build OCI container from Grove environment
- `kanuka grove container sync`: Sync container from Nix store to Docker daemon
- `kanuka grove container enter`: Enter container interactively

### Secrets Commands

- `kanuka secrets init`: Initialize a new secrets store
- `kanuka secrets create`: Create new encryption keys
- `kanuka secrets encrypt`: Encrypt .env files
- `kanuka secrets decrypt`: Decrypt .kanuka files
- `kanuka secrets register --user <username>`: Register a new user
- `kanuka secrets remove --user <username>`: Remove a user
- `kanuka secrets purge`: Purge all secrets

### General Commands

- `kanuka completion <shell>`: Generate autocompletion script for your shell
- `kanuka --help`: Show help information
- `kanuka <command> --help`: Show help for a specific command

## How It Works

### Grove (Development Environment)

Kānuka Grove leverages the Nix ecosystem and devenv to provide reproducible development environments:

1. **Environment Definition**: Uses `devenv.nix` to declaratively define your development environment
2. **Package Management**: Integrates with nixpkgs to provide access to thousands of packages
3. **Reproducibility**: Ensures consistent environments across different machines and team members
4. **Container Generation**: Can build OCI containers from your development environment for deployment
5. **Channel Management**: Allows using different versions of nixpkgs for specific package requirements

### Secrets Management

Kānuka uses a hybrid encryption approach for secure secrets management:

1. A symmetric AES-256 key is used to encrypt your project secrets
2. Each user's RSA-2048 key pair is used to encrypt/decrypt the symmetric key
3. Public keys are stored in the project repository
4. Private keys are stored securely on each user's machine

This approach allows team members to securely share the same secrets without exposing sensitive information.

## Project Structure

### Grove Environment Files

When using Grove, Kānuka creates and manages these files in your project:

```
project/
├── devenv.nix            # Development environment configuration
├── devenv.lock           # Lock file for reproducible builds
├── .devenv/              # Generated environment files
└── .envrc                # Optional direnv integration
```

### Secrets Management Files

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
```

## Shell Autocompletion

Kānuka supports shell autocompletion for bash, zsh, fish, and PowerShell. Run `kanuka completion [shell]` to generate the appropriate completion script.

## Contributing

Contributions are welcome! Feel free to submit issues or pull requests.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
