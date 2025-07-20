# Kanuka Grove Feature Plan

## Overview

`kanuka grove` is a new command suite that provides an enhanced development environment experience using devenv.nix and the devenv ecosystem. It simplifies package management and shell environment setup while maintaining deliberate authentication workflows.

## Core Requirements

1. **NEW command called `kanuka grove`** with alias `kanuka dev`
2. **Uses hybrid devenv.yaml + devenv.nix** for optimal development shell management
3. **Enhanced experience** for package discovery, channel management, and authentication
4. **Declarative nixpkgs pinning** through devenv.yaml input management
5. **Multi-channel support** for mixing stable and unstable packages
6. **Automatic stable channel detection** - always uses latest stable release

## Command Structure

### Primary Commands
```bash
kanuka grove init                    # Initialize devenv.nix + kanuka.toml
kanuka grove add <package>           # Add package (nix-style: nodejs_18)
kanuka grove remove <package>        # Remove package with confirmation
kanuka grove list                    # Show managed packages
kanuka grove search <term>           # Search nixpkgs (enhanced with nix-search-cli)
kanuka grove search --name <pkg>      # Search by exact package name
kanuka grove search --program <bin>   # Search by binary/program name
kanuka grove enter                   # Enter clean shell (no auth)
kanuka grove enter --auth            # Enter clean shell with AWS SSO
kanuka grove enter --env <name>      # Enter shell with named environment
kanuka grove status                  # Show environment status
kanuka dev                          # Alias for 'kanuka grove enter'
```

### Channel Management Commands
```bash
kanuka grove channel list            # List all configured channels
kanuka grove channel add <name> <url> # Add new nixpkgs channel
kanuka grove channel remove <name>   # Remove channel
kanuka grove channel pin <name> <commit> # Pin channel to specific commit
kanuka grove channel update <name>   # Update channel to latest
kanuka grove channel show <name>     # Show channel details
```

### Example Workflow
```bash
kanuka grove init                    # Setup project
kanuka grove add nodejs_18           # Add Node.js v18
kanuka grove add pnpm                # Add pnpm package manager
kanuka grove enter                   # Enter shell with tools available
# Fast development with no auth prompts

kanuka grove enter --auth            # When authentication is needed
kanuka grove enter --env production  # Use specific environment credentials
```

## Key Design Decisions

### 1. Package Management
- **Nix-style versioning**: `nodejs_18`, `typescript_5_3_2`
- **Enhanced search integration**: Leverage `github.com/peterldowns/nix-search-cli` for superior package discovery
- **Multiple search modes**: Support name, program, version, and query-string based searches
- **Version pinning support**: Essential for reproducible environments
- **Conflict resolution**: Ask for confirmation before replacing existing packages

### 2. Configuration Management
- **Pure devenv.nix interface**: No duplicate package databases
- **Project identification**: `kanuka.toml` at project root
- **Kanuka-managed markers**: Clear separation in devenv.nix
- **Extensible design**: `kanuka.toml` ready for future Kanuka features

### 3. Authentication Strategy
- **Deliberate opt-in**: Fast shell entry by default, auth on demand
- **AWS SSO priority**: Primary authentication method for MVP
- **Environment-specific auth**: Named environments for different contexts
- **User-global storage**: Per-project environments in user data directory

### 4. Error Handling & UX
- **Fail-fast approach**: Clear errors, no silent failures
- **Kanuka voice/style**: Minimal emojis (✓/✗), colors, professional tone
- **Helpful guidance**: Clear next-step suggestions
- **Conflict detection**: Warn before modifying existing packages

## Technical Implementation

### File Structure
```
project/
├── kanuka.toml                     # Project config (new, minimal)
├── devenv.yaml                     # Input management & nixpkgs pinning
├── devenv.nix                      # Package management with Kanuka markers
└── .kanuka/                        # Existing secrets structure

~/.local/share/kanuka/
├── keys/                           # Existing secrets keys
└── grove/
    └── <project-id>/               # Grove environments per project
        ├── production.env
        └── staging.env
```

### Hybrid devenv.yaml + devenv.nix Integration

**devenv.yaml** (Input & Channel Management):
```yaml
inputs:
  nixpkgs:
    url: github:NixOS/nixpkgs/nixpkgs-unstable
  nixpkgs-stable:
    url: github:NixOS/nixpkgs/nixos-24.05  # Auto-detected latest stable
  nixpkgs-custom:
    url: github:NixOS/nixpkgs/abc123def456  # Pinned commit

allowUnfree: true
```

**devenv.nix** (Package Management):
```nix
{ pkgs, inputs, ... }: {
  packages = [
    # Existing user packages
    pkgs.git
    
    # Kanuka-managed packages - DO NOT EDIT MANUALLY
    pkgs.nodejs_18                                           # from unstable
    inputs.nixpkgs-stable.legacyPackages.${system}.python39  # from stable
    inputs.nixpkgs-custom.legacyPackages.${system}.terraform # from custom
    # End Kanuka-managed packages
  ];

  dotenv.enable = true;
  
  enterShell = ''
    echo "Welcome to your development environment!"
    echo "Managed by Kanuka Grove"
  '';
}
```

### Automatic Stable Channel Detection

Kanuka Grove automatically detects the latest stable NixOS channel using intelligent date-based logic:

- **Smart Detection**: Uses NixOS release schedule (May .05 and November .11 releases)
- **Current Date Logic**: Automatically determines the latest available stable release
- **Verification**: Confirms channel exists before using it
- **Future-Proof**: Will automatically use newer stable releases (e.g., nixos-25.05, nixos-25.11)
- **Fallback Protection**: Uses known good version (nixos-24.05) if detection fails
- **No Manual Updates**: Stays current without code changes for years

**Detection Logic:**
- **January-April**: Uses previous year's November release (e.g., nixos-23.11)
- **May-October**: Uses current year's May release (e.g., nixos-24.05)  
- **November-December**: Uses current year's November release (e.g., nixos-24.11)

### kanuka.toml Structure (Minimal MVP)
```toml
[project]
id = "generated-project-id"
name = "project-name"

[grove]
# Grove-specific configuration (future expansion)
```

### Command Integration
Following existing Kanuka patterns:
```go
// main.go
func main() {
    rootCmd.AddCommand(cmd.SecretsCmd)
    rootCmd.AddCommand(cmd.GroveCmd)  // New addition
}

// cmd/grove.go (new file)
var GroveCmd = &cobra.Command{
    Use:   "grove",
    Short: "Manage development environments using devenv.nix",
    Long:  `Provides package management and shell environment setup using the devenv ecosystem.`,
}
```

## Voice & Style Guidelines

Following existing Kanuka messaging patterns:

### Success Messages
```bash
kanuka grove add nodejs_18
# ✓ Added nodejs_18 to devenv.nix
# → Run 'kanuka grove enter' to start using Node.js
```

### Error Messages
```bash
kanuka grove add nonexistent
# ✗ Package 'nonexistent' not found in nixpkgs
# → Try 'kanuka grove search nonexistent' to find similar packages
# → Or search by program: 'kanuka grove search --program nonexistent'
```

### Authentication Flow
```bash
kanuka grove enter --auth
# ✓ Authenticated with AWS SSO (production)
# → Shell ready with authenticated AWS access
```

## MVP Success Criteria

1. ✅ User can `kanuka grove init` in any directory
2. ✅ User can add packages with versions: `kanuka grove add nodejs_18`
3. ✅ User can enter shell immediately: `kanuka grove enter`
4. ✅ User can authenticate when needed: `kanuka grove enter --auth`
5. ✅ Existing devenv.nix files are respected and enhanced
6. ✅ Clear, helpful error messages following Kanuka style
7. ✅ Seamless integration with existing Kanuka command structure
8. ✅ Version pinning works reliably
9. ✅ Conflict resolution prevents accidental overwrites

## Post-MVP Roadmap

### Phase 2: Enhanced Package Management
- Common tool aliases (node → nodejs_18)
- Advanced search patterns (`kanuka grove search --version '1.*' golang`)
- Dependency suggestions based on search results
- Bulk operations (`kanuka grove add nodejs_18 pnpm typescript_5_3_2`)
- Search by installed programs (`kanuka grove search --program python`)

### Phase 3: Advanced Authentication
- Multiple cloud providers (GCP, Azure)
- Environment variable management
- Credential file handling

### Phase 4: Team Features
- Package list sharing
- Environment templates
- Integration with Kanuka secrets

## Implementation Notes

### Architecture Decisions
- **Search Integration**: Use `github.com/peterldowns/nix-search-cli` instead of custom `nix search` implementation
  - Provides superior search capabilities (name, program, version, query-string modes)
  - Reduces maintenance burden by leveraging specialized, well-maintained tooling
  - Better error messages and search suggestions
  - Uses search.nixos.org API for faster, more comprehensive results

### Assumptions
- Users with complex existing devenv.nix setups likely won't use Kanuka Grove
- AWS SSO is the primary authentication need
- Simple, fast workflows are more valuable than comprehensive feature sets
- nix-search-cli provides better nixpkgs integration than custom implementation

### Future Integration Points
- Potential integration with Kanuka secrets for seamless environment + auth + secrets workflow
- `kanuka.toml` designed to support future Kanuka project-level features
- Environment storage pattern reusable for other Kanuka features

## Development Priority

### Must Have (MVP)
1. Hybrid devenv.yaml + devenv.nix initialization
2. Channel-aware package add/remove with --channel flag
3. Basic channel management (list, add, remove, pin)
4. Enhanced search command with multiple search modes
5. AWS SSO authentication
6. Declarative nixpkgs pinning through devenv.yaml
7. Conflict detection and resolution
8. Clear error messages and guidance with intelligent search suggestions

### Should Have (Post-MVP)
1. Advanced channel management (update, automatic detection)
2. Bulk package operations
3. Environment management commands
4. Advanced search features (version patterns, complex queries)
5. Performance optimizations and search result caching
6. Channel-specific package validation

### Could Have (Future)
1. Multiple cloud provider support
2. Dependency suggestions
3. Team sharing features
4. Integration with Kanuka secrets

---

This plan provides a focused, powerful MVP that solves core development environment workflow issues while maintaining Kanuka's professional style and leaving room for natural feature evolution.