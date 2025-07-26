---
title: Getting Started with Grove
description: A guide to setting up your first development environment using Kānuka.
---

Ever struggled with "it works on my machine" problems? Grove solves this by giving you reproducible development environments that work the same everywhere.

## What you'll need

Before you can use Grove, you'll need a couple of things installed:

- **Nix**: The package manager that makes Grove possible ([get it here](https://nixos.org/download.html))
- **devenv**: The tool that Grove uses under the hood ([installation guide](https://devenv.sh/getting-started/))
- **Kānuka**: You've already got this if you're reading this!

## Setting up your first environment

### 1. Initialize Grove

To get started with Grove in your project, just run:

```bash
kanuka grove init
```

That's it! Kānuka will create everything you need:
- `devenv.nix` - where your environment is defined
- `devenv.yaml` - configuration for devenv
- `kanuka.toml` - Kānuka's own configuration file
- Updates your `.gitignore` so you don't commit the wrong files

### 2. Add the tools you need

Now you can add whatever packages your project needs:

```bash
# Add programming languages
kanuka grove add nodejs
kanuka grove add python3
kanuka grove add go

# Add development tools
kanuka grove add git
kanuka grove add docker
kanuka grove add awscli2
```

### 3. Enter your environment

To start using your new environment, run:

```bash
kanuka grove enter
# or use the shortcut
kanuka dev
```

This will drop you into a clean shell where all your packages are available and ready to use.

### 4. Check everything works

You can verify your environment is working by checking the versions:

```bash
# Inside your grove environment
node --version
python --version
go version
```

## What else can you do?

### Adding container support

If you want to build containers from your environment, you can enable that too:

```bash
kanuka grove init --containers
# or if you already have Grove set up
kanuka grove container init
```

### Using different package versions

Sometimes you want stable packages instead of the latest ones:

```bash
# Use stable packages for production-like environments
kanuka grove add nodejs --channel stable

# Use a specific channel
kanuka grove add python3 --channel nixpkgs-stable
```

### AWS development

If you're working with AWS, Grove can handle authentication for you:

```bash
kanuka grove enter --auth
```

## Next steps

To learn more about Grove, you can read about:

- [Development environments](/grove/development-environments/) and how they work
- [Package management](/grove/package-management/) for adding and removing tools
- [Container support](/grove/containers/) for building deployable containers
- [Channel management](/grove/channels/) for controlling package versions

## When things go wrong

### Common problems

**"devenv not found"**: Make sure you've installed devenv and it's in your PATH.

**"Nix not found"**: You'll need to install the Nix package manager first.

**Package not found**: Try searching for it with `kanuka grove search <package-name>`.

**Environment conflicts**: Grove environments are isolated, but make sure you're not already inside another development shell.