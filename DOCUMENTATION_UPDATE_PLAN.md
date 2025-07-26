# Kānuka Documentation Website Update Plan

## Overview

Kānuka has evolved from a secrets management tool into a comprehensive development environment and secrets management platform. The current documentation website only covers secrets management features and is missing all Grove (development environment) functionality, container management, channel management, and AWS SSO integration.

## Current State Analysis

### ✅ What's Currently Documented

- Secrets management (encrypt, decrypt, register, remove, purge)
- Basic installation and setup
- Secrets concepts and project structure
- CLI reference for secrets commands only

### ❌ What's Missing from Documentation

- **Grove development environment management** (completely absent)
- **Container management features** (build, sync, enter containers)
- **Channel management** (nixpkgs channels, pinning, updates)
- **AWS SSO integration** (authentication workflows)
- **Package management** (add, remove, search packages)
- **Development shell environments** (enter, status commands)
- **`kanuka dev` alias command**
- **Updated homepage reflecting full capabilities**
- **Complete CLI reference** (missing ~15+ Grove commands)

## Documentation Update Plan

### Phase 1: Foundation Updates

**Goal**: Update core structure and homepage to reflect Kānuka's full capabilities

#### 1.1 Homepage Updates (`docs/src/content/docs/index.mdx`)

- [ ] Update hero tagline from "secrets management" to comprehensive development tool
- [ ] Add Grove development environment cards alongside secrets cards
- [ ] Update feature highlights to include both Grove and secrets
- [ ] Add container and AWS integration highlights

#### 1.2 Introduction Updates (`docs/src/content/docs/introduction/kanuka.mdx`)

- [ ] Remove outdated note: "current scope only encompasses secrets management"
- [ ] Add Grove introduction and capabilities
- [ ] Update tool description to reflect dual nature (Grove + Secrets)
- [ ] Add development environment use cases

#### 1.3 Sidebar Structure Update (`docs/astro.config.mjs`)

- [ ] Reorganize sidebar to separate Grove and Secrets sections
- [ ] Add new Grove section with proper navigation
- [ ] Rename "Guides" to "Secrets Management" for clarity
- [ ] Update section ordering for better user flow

### Phase 2: Grove Documentation Creation

**Goal**: Create comprehensive documentation for Grove development environment features

#### 2.1 Create Grove Documentation Structure

```
docs/src/content/docs/grove/
├── introduction.mdx           # What is Grove? Core concepts
├── getting-started.md         # Quick start guide for Grove
├── development-environments.md # devenv.nix, kanuka.toml concepts
├── package-management.md      # add, remove, search, list commands
├── channels.md               # Channel management and nixpkgs
├── containers.md             # Container features and OCI builds
└── aws-integration.md        # AWS SSO authentication
```

#### 2.2 Grove Introduction (`grove/introduction.mdx`)

- [ ] Explain Grove's purpose and relationship to devenv/Nix
- [ ] Cover reproducible development environments concept
- [ ] Explain kanuka.toml and devenv.nix integration
- [ ] Show benefits over traditional development setups

#### 2.3 Grove Getting Started (`grove/getting-started.md`)

- [ ] Prerequisites (Nix installation, devenv setup)
- [ ] `kanuka grove init` walkthrough
- [ ] First package addition with `kanuka grove add`
- [ ] Entering development shell with `kanuka grove enter`
- [ ] Basic workflow examples

#### 2.4 Development Environments (`grove/development-environments.md`)

- [ ] Deep dive into devenv.nix structure
- [ ] kanuka.toml configuration options
- [ ] Environment isolation and reproducibility
- [ ] Project structure and file management
- [ ] Integration with existing projects

#### 2.5 Package Management (`grove/package-management.md`)

- [ ] `kanuka grove add` with examples and options
- [ ] `kanuka grove remove` for cleanup
- [ ] `kanuka grove search` for package discovery
- [ ] `kanuka grove list` for inventory
- [ ] Language vs package distinction
- [ ] Version pinning and channel selection

#### 2.6 Channel Management (`grove/channels.md`)

- [ ] Understanding nixpkgs channels
- [ ] `kanuka grove channel list/add/remove` commands
- [ ] Channel pinning with `kanuka grove channel pin`
- [ ] Updating channels with `kanuka grove channel update`
- [ ] Stable vs unstable channel strategies
- [ ] Custom channel configuration

#### 2.7 Container Features (`grove/containers.md`)

- [ ] Container initialization with `kanuka grove container init`
- [ ] Building OCI containers with `kanuka grove container build`
- [ ] Syncing to Docker daemon with `kanuka grove container sync`
- [ ] Interactive container access with `kanuka grove container enter`
- [ ] Container deployment strategies

#### 2.8 AWS Integration (`grove/aws-integration.md`)

- [ ] AWS SSO authentication with `--auth` flag
- [ ] Profile management and cleanup
- [ ] Session-based authentication workflow
- [ ] Integration with AWS CLI and SDKs
- [ ] Troubleshooting authentication issues

### Phase 3: CLI Reference Complete Rewrite

**Goal**: Provide comprehensive, up-to-date CLI documentation

#### 3.1 Update CLI Reference (`docs/src/content/docs/reference/references.md`)

- [ ] Add complete Grove command documentation
- [ ] Update secrets command documentation with current flags
- [ ] Add `kanuka dev` alias documentation
- [ ] Include all subcommands and their options
- [ ] Add practical examples for each command
- [ ] Document flag combinations and use cases

#### 3.2 Grove Commands to Document

```bash
# Core Grove commands
kanuka grove init [--containers]
kanuka grove add <package> [--channel] [--skip-validation]
kanuka grove remove <package>
kanuka grove list
kanuka grove search <query> [--program]
kanuka grove enter [--auth] [--env]
kanuka grove status

# Channel management
kanuka grove channel list
kanuka grove channel add <name> <url>
kanuka grove channel remove <name>
kanuka grove channel show <name>
kanuka grove channel pin <name> <commit>
kanuka grove channel update

# Container management
kanuka grove container init
kanuka grove container build
kanuka grove container sync
kanuka grove container enter

# Alias
kanuka dev [--auth] [--env]
```

### Phase 4: Getting Started & Configuration Updates

**Goal**: Update foundational documentation to reflect current capabilities

#### 4.1 Getting Started Updates

- [ ] Update `first-steps.md` to include Grove workflow
- [ ] Add Grove prerequisites to installation guide
- [ ] Create decision tree: when to use Grove vs Secrets vs both
- [ ] Add troubleshooting section for common setup issues

#### 4.2 Configuration Documentation (`docs/src/content/docs/configuration/configuration.mdx`)

- [ ] Replace placeholder content with actual configuration
- [ ] Document kanuka.toml structure and options
- [ ] Document devenv.yaml integration
- [ ] Document AWS SSO configuration
- [ ] Add configuration examples and best practices

#### 4.3 Concepts Updates

- [ ] Update structure.mdx to include Grove files
- [ ] Add Grove-specific concepts documentation
- [ ] Update existing concepts to show relationship between Grove and Secrets

### Phase 5: Content Polish & Integration

**Goal**: Ensure consistency, quality, and discoverability

#### 5.1 Content Review

- [ ] Ensure consistent terminology throughout
- [ ] Verify all internal links work correctly
- [ ] Check code examples are accurate and tested
- [ ] Ensure proper cross-referencing between sections

#### 5.2 Navigation & UX

- [ ] Test navigation flow for new users
- [ ] Ensure logical progression from introduction to advanced topics
- [ ] Add "Next Steps" sections to guide user journey
- [ ] Include relevant cross-links between Grove and Secrets features

#### 5.3 Visual Assets

- [ ] Update any screenshots or diagrams
- [ ] Ensure Grove features are represented in visual content
- [ ] Check that all image assets load correctly

## Implementation Strategy

### Recommended Order

1. **Start with Phase 1** - Foundation updates provide immediate value
2. **Focus on Phase 2.1-2.3** - Core Grove documentation for basic workflows
3. **Complete Phase 3** - CLI reference for comprehensive coverage
4. **Finish Phase 2.4-2.8** - Advanced Grove features
5. **Complete Phase 4-5** - Polish and integration

### Success Metrics

- [ ] All major Kānuka features are documented
- [ ] New users can successfully set up both Grove and Secrets
- [ ] CLI reference matches actual command functionality
- [ ] Documentation site reflects Kānuka's current capabilities
- [ ] Clear separation and integration between Grove and Secrets workflows

## Technical Notes

### File Structure Changes

```
docs/src/content/docs/
├── index.mdx                    # Updated homepage
├── introduction/
│   └── kanuka.mdx              # Updated introduction
├── getting-started/            # Updated with Grove
├── grove/                      # NEW: Complete Grove section
├── guides/                     # Renamed to focus on secrets
├── concepts/                   # Updated for both features
├── configuration/              # Actual configuration docs
└── reference/                  # Complete CLI reference
```

### Astro Configuration Updates

- Update sidebar structure in `astro.config.mjs`
- Ensure proper routing for new Grove section
- Maintain existing functionality for secrets documentation

## Next Steps

Choose starting point based on priorities:

1. **Quick wins**: Start with Phase 1 for immediate homepage improvements
2. **User impact**: Start with Phase 2.1-2.3 for core Grove documentation
3. **Completeness**: Start with Phase 3 for comprehensive CLI coverage

