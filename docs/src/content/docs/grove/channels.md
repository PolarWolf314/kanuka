---
title: Managing Channels
description: A guide to controlling package versions and sources in your Grove environment.
---

Channels are how Grove controls which versions of packages you get. Think of them as different "streams" of packages - some have the latest and greatest, others focus on stability.

## What are channels?

Channels are basically different versions of the package repository:

- **unstable**: The latest packages with all the newest features (this is the default)
- **stable**: Well-tested packages that change less often
- **custom**: Your own channel definitions for specific needs

## The channels Grove gives you

Grove comes with a couple of standard channels already set up:

### unstable (the default)
- Latest package versions
- Gets updated frequently
- Great for development and trying new things
- Sometimes things might break (that's the trade-off for being cutting-edge)

### stable
- Well-tested package versions
- Less frequent updates
- Better for production-like environments
- More predictable and reliable

## Channel Commands

### List Channels
```bash
kanuka grove channel list
```

Shows all configured channels with their URLs and current commits.

### Add Channel
```bash
kanuka grove channel add <name> <url>
```

Examples:
```bash
# Add a specific nixpkgs branch
kanuka grove channel add nixos-22.11 github:NixOS/nixpkgs/nixos-22.11

# Add a custom channel
kanuka grove channel add my-packages github:myorg/my-nixpkgs
```

### Remove Channel
```bash
kanuka grove channel remove <name>
```

Example:
```bash
kanuka grove channel remove my-old-channel
```

### Show Channel Details
```bash
kanuka grove channel show <name>
```

Displays detailed information about a specific channel.

### Pin Channel
```bash
kanuka grove channel pin <name> <commit>
```

Pin a channel to a specific commit for reproducibility:
```bash
kanuka grove channel pin unstable abc123def456
```

### Update Channels
```bash
kanuka grove channel update
```

Updates all channels to their latest versions.

## Using Channels

### Specify Channel for Packages
```bash
kanuka grove add <package> --channel <channel-name>
```

Examples:
```bash
# Use stable channel for production tools
kanuka grove add nodejs --channel stable

# Use unstable for latest features
kanuka grove add python3 --channel unstable

# Use custom channel
kanuka grove add my-tool --channel my-packages
```

### Channel Validation

Grove validates packages against channels:

- **Standard channels** (unstable, stable): Full validation
- **Custom channels**: Validation automatically skipped
- **Unknown packages**: Suggestions provided

## Channel Strategies

### Development Strategy
```bash
# Use unstable for most packages
kanuka grove add nodejs --channel unstable
kanuka grove add python3 --channel unstable

# Use stable for critical tools
kanuka grove add git --channel stable
```

### Production Strategy
```bash
# Use stable for all packages
kanuka grove add nodejs --channel stable
kanuka grove add python3 --channel stable

# Pin channels for reproducibility
kanuka grove channel pin stable abc123def456
```

### Mixed Strategy
```bash
# Stable base tools
kanuka grove add git --channel stable
kanuka grove add curl --channel stable

# Unstable development tools
kanuka grove add nodejs --channel unstable
kanuka grove add typescript --channel unstable
```

## Channel Configuration

Channels are configured in `devenv.yaml`:

```yaml
inputs:
  nixpkgs:
    url: github:NixOS/nixpkgs/nixpkgs-unstable
  nixpkgs-stable:
    url: github:NixOS/nixpkgs/nixos-22.11
  my-packages:
    url: github:myorg/my-nixpkgs
```

## Advanced Channel Management

### Custom Channels

Create your own channels for:
- Custom package versions
- Internal company packages
- Experimental packages
- Security-patched packages

```bash
# Add your custom channel
kanuka grove channel add company-packages github:company/nixpkgs

# Use packages from custom channel
kanuka grove add internal-tool --channel company-packages
```

### Channel Pinning for Reproducibility

Pin channels to ensure exact reproducibility:

```bash
# Pin to specific commit
kanuka grove channel pin unstable abc123def456789

# Pin stable channel
kanuka grove channel pin stable def456789abc123
```

### Channel Updates

Regular update workflow:

```bash
# Update all channels
kanuka grove channel update

# Check what changed
kanuka grove channel list

# Test environment still works
kanuka grove enter
```

## Channel Best Practices

### For Teams
1. **Document channel choices** in project README
2. **Pin channels** for reproducible builds
3. **Test updates** before applying to team
4. **Coordinate updates** across team members

### For Projects
1. **Start with unstable** for development
2. **Move to stable** for production preparation
3. **Pin before releases** for reproducibility
4. **Update regularly** but carefully

### For Security
1. **Monitor security updates** in channels
2. **Update promptly** for security patches
3. **Test thoroughly** after updates
4. **Document security requirements**

## Troubleshooting Channels

### Channel Not Found
```bash
# Check available channels
kanuka grove channel list

# Add missing channel
kanuka grove channel add <name> <url>
```

### Package Not in Channel
```bash
# Search in different channel
kanuka grove search <package>

# Try different channel
kanuka grove add <package> --channel stable
```

### Channel Update Issues
```bash
# Check channel status
kanuka grove channel show <name>

# Remove and re-add problematic channel
kanuka grove channel remove <name>
kanuka grove channel add <name> <url>
```

### Validation Errors
```bash
# Check if package exists in channel
kanuka grove search <package>

# Use different channel
kanuka grove add <package> --channel unstable

# Skip validation for custom packages
kanuka grove add <package> --skip-validation
```

## Common Channel Configurations

### Standard Development
```yaml
inputs:
  nixpkgs:
    url: github:NixOS/nixpkgs/nixpkgs-unstable
```

### Stable Production
```yaml
inputs:
  nixpkgs:
    url: github:NixOS/nixpkgs/nixos-22.11
```

### Mixed Environment
```yaml
inputs:
  nixpkgs:
    url: github:NixOS/nixpkgs/nixpkgs-unstable
  nixpkgs-stable:
    url: github:NixOS/nixpkgs/nixos-22.11
```

### Custom Packages
```yaml
inputs:
  nixpkgs:
    url: github:NixOS/nixpkgs/nixpkgs-unstable
  company-packages:
    url: github:company/nixpkgs
```

## Next Steps

- Learn about [container integration](/grove/containers/) for deployment
- Explore [AWS integration](/grove/aws-integration/) for cloud development
- Check out [package management](/grove/package-management/) for more details