---
title: Development Environments
description: Understanding Grove's development environment concepts
---

Grove creates reproducible development environments using the Nix ecosystem and devenv. This guide explains the core concepts and how they work together.

## Environment Files

Grove manages several files in your project:

### devenv.nix
The main environment definition file written in the Nix language:

```nix
{ pkgs, ... }: {
  # Packages managed by Kānuka Grove
  packages = [
    pkgs.nodejs_18
    pkgs.python3
    pkgs.git
  ];

  # Languages (also managed by Grove)
  languages.typescript.enable = true;
  
  # Services and additional configuration
  services.postgres.enable = true;
}
```

### kanuka.toml
Kānuka's configuration file tracking Grove-managed packages:

```toml
[grove]
packages = ["nodejs_18", "python3", "git"]
languages = ["typescript"]
```

### devenv.yaml
devenv configuration for inputs and channels:

```yaml
inputs:
  nixpkgs:
    url: github:NixOS/nixpkgs/nixpkgs-unstable
```

## Environment Isolation

Grove environments are completely isolated from your system:

- **No system dependencies**: Everything needed is declared in devenv.nix
- **Clean PATH**: Only declared packages are available
- **Reproducible**: Same environment on every machine
- **Project-specific**: Each project has its own environment

## Package Management

Grove distinguishes between two types of additions:

### Packages
Individual tools and applications:
```bash
kanuka grove add nodejs    # Adds Node.js runtime
kanuka grove add docker    # Adds Docker CLI
kanuka grove add awscli2   # Adds AWS CLI v2
```

### Languages
Programming language environments with additional tooling:
```bash
kanuka grove add typescript  # Enables TypeScript language support
kanuka grove add rust       # Enables Rust language environment
kanuka grove add go         # Enables Go language environment
```

## Channel Management

Channels determine which version of packages you get:

- **unstable**: Latest packages (default)
- **stable**: Stable, tested packages
- **custom**: Your own channel definitions

```bash
# Use stable channel for production-like environments
kanuka grove add nodejs --channel stable

# Use unstable for latest features
kanuka grove add python3 --channel unstable
```

## Environment Lifecycle

### Entering the Environment
```bash
kanuka grove enter
# or
kanuka dev
```

### Checking Status
```bash
kanuka grove status
```

### Updating Packages
```bash
kanuka grove channel update  # Update channel definitions
# Then rebuild environment by entering again
```

## Integration with Existing Projects

Grove works with any project structure:

1. **New projects**: Start with `kanuka grove init`
2. **Existing projects**: Add Grove to existing codebases
3. **Team adoption**: Share devenv.nix and kanuka.toml via git

## Best Practices

### Version Control
Always commit these files:
- `devenv.nix`
- `devenv.yaml` 
- `kanuka.toml`
- `.gitignore` (updated by Grove)

Never commit:
- `.devenv/` directory
- `devenv.lock` (unless you want to pin exact versions)

### Team Collaboration
1. One team member sets up Grove environment
2. Commits configuration files to git
3. Other team members run `kanuka grove enter`
4. Everyone has identical environment

### Environment Updates
- Use `kanuka grove add/remove` for package changes
- Update channels periodically with `kanuka grove channel update`
- Test environment changes before committing

## Advanced Features

### Container Integration
Build containers from your development environment:
```bash
kanuka grove container init
kanuka grove container build
```

### AWS Integration
Automatic AWS SSO authentication:
```bash
kanuka grove enter --auth
```

### Multiple Environments
Use named environments for different configurations:
```bash
kanuka grove enter --env production
kanuka grove enter --env testing
```

## Next Steps

- Learn about [package management](/grove/package-management/) in detail
- Explore [channel management](/grove/channels/) for version control
- Set up [container support](/grove/containers/) for deployment