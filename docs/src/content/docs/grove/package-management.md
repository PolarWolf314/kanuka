---
title: Managing Packages
description: A guide to adding, removing, and managing packages in your Grove environment.
---

One of the best things about Grove is how easy it is to add and manage the tools your project needs. You get access to thousands of packages from the Nix ecosystem without any of the usual dependency headaches.

## Adding packages

### The basics

Adding a package is as simple as:

```bash
kanuka grove add <package-name>
```

Here are some examples:

```bash
kanuka grove add nodejs     # Latest Node.js
kanuka grove add python3    # Python 3
kanuka grove add git        # Git version control
kanuka grove add docker     # Docker CLI
kanuka grove add awscli2    # AWS CLI v2
```

### Adding specific versions

Sometimes you need a specific version of a package. You can do that too:
```bash
kanuka grove add nodejs_18  # Node.js version 18
kanuka grove add python39   # Python 3.9
kanuka grove add go_1_19    # Go version 1.19
```

### Using different channels

You can also choose which channel (version source) to use:

```bash
kanuka grove add nodejs --channel stable      # From stable channel
kanuka grove add python3 --channel unstable   # From unstable channel
kanuka grove add rust --channel nixpkgs-stable # From specific channel
```

## Languages vs packages

Grove treats languages and packages a bit differently:

### What's the difference?
- **Languages**: Full development environments with extra tooling and setup
- **Packages**: Individual tools and applications

```bash
# Language environments (includes tooling, LSP, etc.)
kanuka grove add typescript  # TypeScript language support
kanuka grove add rust       # Rust development environment
kanuka grove add go         # Go development environment

# Individual packages
kanuka grove add nodejs     # Just the Node.js runtime
kanuka grove add cargo      # Just the Cargo tool
```

## Searching for Packages

### General Search
```bash
kanuka grove search <query>
```

Examples:
```bash
kanuka grove search node     # Find Node.js related packages
kanuka grove search python   # Find Python packages
kanuka grove search aws      # Find AWS tools
```

### Program-Specific Search
```bash
kanuka grove search --program <program-name>
```

This searches for packages that provide a specific executable.

## Removing Packages

### Remove Individual Packages
```bash
kanuka grove remove <package-name>
```

Examples:
```bash
kanuka grove remove nodejs
kanuka grove remove python3
kanuka grove remove docker
```

### Remove Languages
```bash
kanuka grove remove typescript
kanuka grove remove rust
```

## Listing Packages

### View All Packages
```bash
kanuka grove list
```

This shows:
- All Grove-managed packages
- All Grove-managed languages
- Current channel information
- Package status

## Package Validation

Grove validates packages against nixpkgs by default:

### Automatic Validation
- Packages are checked against the specified channel
- Invalid packages are rejected with suggestions
- Validation ensures packages exist and are available

### Skip Validation
For testing or custom packages:
```bash
kanuka grove add custom-package --skip-validation
```

## Channel-Specific Packages

### Understanding Channels
Different channels provide different package versions:

- **unstable**: Latest packages, frequent updates
- **stable**: Tested packages, less frequent updates
- **custom**: Your own channel definitions

### Channel Validation
- **Standard channels** (unstable, stable): Full validation
- **Custom channels**: Validation automatically skipped

```bash
# These are validated
kanuka grove add nodejs --channel unstable
kanuka grove add nodejs --channel stable

# This skips validation (custom channel)
kanuka grove add nodejs --channel my-custom-channel
```

## Package Conflicts

### Handling Existing Packages
If a package already exists:

1. **Grove-managed**: Use `kanuka grove remove` first
2. **Non-Grove managed**: Grove will ask for confirmation to replace

### Example Workflow
```bash
# If nodejs already exists and is Grove-managed
kanuka grove remove nodejs
kanuka grove add nodejs_18

# If nodejs exists but not Grove-managed
kanuka grove add nodejs  # Will prompt for confirmation
```

## Advanced Package Management

### Bulk Operations
```bash
# Add multiple packages at once
kanuka grove add nodejs python3 git docker

# Add with specific channels
kanuka grove add nodejs --channel stable
kanuka grove add python3 --channel unstable
```

### Package Information
```bash
# Search for package details
kanuka grove search nodejs

# Check current environment status
kanuka grove status
```

## Best Practices

### Package Selection
1. **Use specific versions** for production environments
2. **Use latest versions** for development and experimentation
3. **Document package choices** in your project README

### Channel Strategy
1. **Start with unstable** for latest features
2. **Move to stable** for production-like environments
3. **Pin channels** for reproducible builds

### Team Coordination
1. **Communicate package changes** to team members
2. **Test package additions** before committing
3. **Document package requirements** and reasoning

## Common Packages

### Development Tools
```bash
kanuka grove add git          # Version control
kanuka grove add curl         # HTTP client
kanuka grove add jq           # JSON processor
kanuka grove add tree         # Directory visualization
```

### Programming Languages
```bash
kanuka grove add nodejs       # JavaScript/Node.js
kanuka grove add python3      # Python
kanuka grove add go           # Go
kanuka grove add rust         # Rust
kanuka grove add java         # Java
```

### Cloud Tools
```bash
kanuka grove add awscli2      # AWS CLI
kanuka grove add kubectl      # Kubernetes CLI
kanuka grove add terraform    # Infrastructure as Code
kanuka grove add docker       # Container tools
```

### Databases
```bash
kanuka grove add postgresql   # PostgreSQL client
kanuka grove add mysql        # MySQL client
kanuka grove add redis        # Redis CLI
```

## Troubleshooting

### Package Not Found
```bash
# Search for similar packages
kanuka grove search <partial-name>

# Try different channels
kanuka grove add <package> --channel stable
```

### Package Conflicts
```bash
# Check what's currently installed
kanuka grove list

# Remove conflicting package first
kanuka grove remove <conflicting-package>
```

### Validation Errors
```bash
# Skip validation for testing
kanuka grove add <package> --skip-validation

# Try a different channel
kanuka grove add <package> --channel stable
```

## Next Steps

- Learn about [channel management](/grove/channels/) for version control
- Explore [container integration](/grove/containers/) for deployment
- Set up [AWS integration](/grove/aws-integration/) for cloud development