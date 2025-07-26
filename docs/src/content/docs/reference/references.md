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

## Shell Completion Setup

Use `kanuka completion [shell]` to generate completion scripts for your preferred shell:

- **Bash**: `kanuka completion bash`
- **Zsh**: `kanuka completion zsh`
- **Fish**: `kanuka completion fish`
- **PowerShell**: `kanuka completion powershell`

Refer to each sub-command's help for details on how to use the generated script.
