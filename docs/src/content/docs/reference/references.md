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

Provides encryption, decryption, registration, removal, initialization, and purging of secrets.

### `kanuka secrets`

```
Usage:
  kanuka secrets [command]

Available Commands:
  create      Creates and adds your public key, and gives instructions on how to gain access
  decrypt     Decrypts the .env.kanuka file back into .env using your Kānuka key
  encrypt     Encrypts the .env file into .env.kanuka using your Kānuka key
  init        Initializes the secrets store
  purge       Purges all secrets, including from the git history
  register    Registers a new user to be given access to the repository's secrets
  remove      Removes access to the secret store

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
  -h, --help      help for decrypt
  -v, --verbose   enable verbose output
```

### `kanuka secrets encrypt`

Encrypts the `.env` file into `.env.kanuka` using your Kānuka key.

```
Usage:
  kanuka secrets encrypt [flags]

Flags:
  -h, --help      help for encrypt
  -v, --verbose   enable verbose output
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

### `kanuka secrets purge`

Purges all secrets, including from the git history.

```
Usage:
  kanuka secrets purge [flags]

Flags:
  -h, --help   help for purge
```

### `kanuka secrets register`

Registers a new user to be given access to the repository's secrets.

```
Usage:
  kanuka secrets register [flags]

Flags:
  -f, --file string     the path to a custom public key — will add public key to the project
  -h, --help            help for register
      --pubkey string   OpenSSH or PEM public key content to be saved with the specified username
  -u, --user string     username to register for access
  -v, --verbose         enable verbose output
```

### `kanuka secrets remove`

Removes access to the secret store.

```
Usage:
  kanuka secrets remove [flags]

Flags:
  -h, --help   help for remove
```

## Grove Development Environment Commands

Provides package management and shell environment setup using the devenv ecosystem.

### `kanuka grove`

```
Usage:
  kanuka grove [command]

Available Commands:
  add         Add a package to the development environment
  channel     Manage nixpkgs channels for Grove environment
  container   Manage containers for Grove environments
  enter       Enter the development shell environment
  init        Initialize a development environment with devenv.nix
  list        Show all Kanuka-managed packages and languages
  remove      Remove a package from the development environment
  search      Search nixpkgs for packages
  status      Show development environment status

Flags:
      --debug     enable debug output
  -h, --help      help for grove
  -v, --verbose   enable verbose output
```

### `kanuka grove init`

Initialize a development environment with devenv.nix.

```
Usage:
  kanuka grove init [flags]

Flags:
      --containers   enable container support
  -h, --help         help for init
```

### `kanuka grove add`

Add a package to your development environment by modifying devenv.nix.

```
Usage:
  kanuka grove add <package>[@version] [flags]

Flags:
      --channel string    nixpkgs channel to use (unstable, stable, or any channel name from devenv.yaml)
                          Note: Custom channels automatically skip validation (default "unstable")
  -h, --help              help for add
      --skip-validation   skip nixpkgs validation (for testing)
```

### `kanuka grove remove`

Remove a package from your development environment by modifying devenv.nix.

```
Usage:
  kanuka grove remove <package> [flags]

Flags:
  -h, --help   help for remove
```

### `kanuka grove list`

Display all packages and languages currently managed by Kanuka in your development environment.

```
Usage:
  kanuka grove list [flags]

Flags:
  -h, --help             help for list
      --languages-only   show only languages
      --packages-only    show only packages
```

### `kanuka grove search`

Search nixpkgs for packages using multiple search modes.

```
Usage:
  kanuka grove search <term> [flags]

Flags:
  -d, --details           show detailed package information
  -h, --help              help for search
  -j, --json              output results in JSON format
  -m, --max-results int   maximum number of results to show (default 25)
      --name string       search by exact package name
      --program string    search by program/binary name
      --version string    search by version (future feature)
```

### `kanuka grove enter`

Enter the development shell environment using devenv with --clean flag.

```
Usage:
  kanuka grove enter [flags]

Flags:
      --auth         prompt for AWS SSO authentication (session-only)
      --env string   use named environment configuration
  -h, --help         help for enter
```

### `kanuka grove status`

Display comprehensive status information about your development environment.

```
Usage:
  kanuka grove status [flags]

Flags:
      --compact   show compact status summary
  -h, --help      help for status
```

### `kanuka grove channel`

Manage nixpkgs channels including listing, adding, removing, pinning, and updating channels.

```
Usage:
  kanuka grove channel [command]

Available Commands:
  add         Add a new nixpkgs channel
  list        Show all configured nixpkgs channels
  pin         Pin a channel to a specific commit
  remove      Remove a nixpkgs channel from Grove environment
  show        Show detailed information about a specific channel
  update      Update channels to their latest versions

Flags:
  -h, --help   help for channel
```

### `kanuka grove container`

Build and manage OCI containers from your Grove development environment.

```
Usage:
  kanuka grove container [command]

Available Commands:
  build       Build OCI container from Grove environment
  enter       Enter container interactively for testing
  init        Initialize container support for Grove environment
  sync        Sync container from Nix store to Docker daemon

Flags:
  -h, --help   help for container
```

### `kanuka dev`

Alias for `kanuka grove enter` - Enter the development shell environment.

```
Usage:
  kanuka dev [flags]

Flags:
      --auth         prompt for AWS SSO authentication (session-only)
      --env string   use named environment configuration
  -h, --help         help for dev
```

## Shell Completion Setup

Use `kanuka completion [shell]` to generate completion scripts for your preferred shell:

- **Bash**: `kanuka completion bash`
- **Zsh**: `kanuka completion zsh`
- **Fish**: `kanuka completion fish`
- **PowerShell**: `kanuka completion powershell`

Refer to each sub-command's help for details on how to use the generated script.
