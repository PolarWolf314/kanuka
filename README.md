# Kānuka

Kānuka is a powerful command-line tool written in Go for managing secrets in your development projects. It makes sharing secrets, creating development environments, deploying code, and testing unified under one robust and simple interface.

> **Note:** The current scope of the project only encompasses secrets management.

## What's with the name?

Kānuka (_Kunzea ericoides_) is a tree that is endemic to Aotearoa New Zealand. It is a robust species, critical to restoring wildlife destroyed by fire as it quickly propagates and regenerates the land. Its leaves have a characteristically soft touch, and it's one of few plants that can survive the heat of geothermal features.

It is fast, resilient, yet pleasant to touch. This is the vision of Kānuka.

## Features

- **Secure Secret Management**: Store and encrypt environment variables using industry-standard encryption (AES-256 and RSA-2048)
- **User-friendly Interface**: Simple commands for managing secrets across your team
- **Cross-platform Support**: Works on Linux, macOS, and Windows
- **Shell Autocompletion**: Supports bash, zsh, fish, and PowerShell

## Prerequisites

- Go 1.21 or later
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

1. **Check Installation**:

   ```bash
   kanuka
   ```

   You should see: `Welcome to Kānuka! Run 'kanuka --help' to see available commands.`

2. **Initialize a Project**:

   ```bash
   kanuka secrets init
   ```

3. **Create Your Secrets**:

   ```bash
   kanuka secrets create
   ```

4. **Encrypt Your Secrets**:

   ```bash
   kanuka secrets encrypt
   ```

5. **Register a Team Member**:
   ```bash
   kanuka secrets register --user username
   ```

## Commands

- `kanuka secrets init`: Initialize a new project
- `kanuka secrets create`: Create new encryption keys
- `kanuka secrets encrypt`: Encrypt .env files
- `kanuka secrets decrypt`: Decrypt .kanuka files
- `kanuka secrets register`: Register a new user
- `kanuka secrets remove`: Remove a user
- `kanuka secrets purge`: Purge all secrets

Use `kanuka --help` or `kanuka [command] --help` for more information.

## How It Works

Kānuka uses a hybrid encryption approach:

1. A symmetric AES-256 key is used to encrypt your project secrets
2. Each user's RSA-2048 key pair is used to encrypt/decrypt the symmetric key
3. Public keys are stored in the project repository
4. Private keys are stored securely on each user's machine

This approach allows team members to securely share the same secrets without exposing sensitive information.

## Project Structure

Kānuka stores project-specific files in a `.kanuka` folder at the root of your project:

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

## Documentation

For complete documentation, visit our [official documentation site](https://kanuka.guo.nz).

## Contributing

Contributions are welcome! Feel free to submit issues or pull requests.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

