---
title: Grove Getting Started
description: Quick start guide for Grove development environments
---

Get up and running with Grove development environments in just a few minutes.

## Prerequisites

Before using Grove, you'll need:

- **Nix**: The Nix package manager ([installation guide](https://nixos.org/download.html))
- **devenv**: Development environment tool ([installation guide](https://devenv.sh/getting-started/))
- **Kānuka**: Already installed if you're reading this!

## Quick Start

### 1. Initialize Grove Environment

Create a new development environment in your project:

```bash
kanuka grove init
```

This creates:
- `devenv.nix` - Your environment definition
- `devenv.yaml` - devenv configuration
- `kanuka.toml` - Kānuka's Grove configuration
- Updates `.gitignore` with appropriate entries

### 2. Add Packages

Add the tools and languages you need:

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

### 3. Enter Development Environment

Start using your environment:

```bash
kanuka grove enter
# or use the shorthand
kanuka dev
```

This drops you into a clean shell with all your packages available.

### 4. Verify Your Environment

Check what's available:

```bash
# Inside the grove shell
node --version
python --version
go version
```

## Common Workflows

### Adding Container Support

Enable container building:

```bash
kanuka grove init --containers
# or add to existing environment
kanuka grove container init
```

### Using Different Package Channels

```bash
# Use stable packages
kanuka grove add nodejs --channel stable

# Use specific channel
kanuka grove add python3 --channel nixpkgs-stable
```

### AWS Development

Enable AWS SSO authentication:

```bash
kanuka grove enter --auth
```

## Next Steps

- Learn about [development environments](/grove/development-environments/) in depth
- Explore [package management](/grove/package-management/) features
- Set up [container support](/grove/containers/) for deployment
- Configure [AWS integration](/grove/aws-integration/) for cloud development

## Troubleshooting

### Common Issues

**"devenv not found"**: Make sure devenv is installed and in your PATH.

**"Nix not found"**: Install Nix package manager first.

**Package not found**: Try searching with `kanuka grove search <package-name>`.

**Environment conflicts**: Grove environments are isolated, but ensure you're not in another development shell.