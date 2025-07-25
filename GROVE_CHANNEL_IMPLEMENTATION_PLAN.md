# Implementation Plan for `kanuka grove channel`

## Overview
Implement the complete channel management command suite to allow users to manage nixpkgs channels in their Grove environments. This will complete the Grove feature set by providing advanced channel management capabilities.

## Command Structure Design

### Main Channel Command
```go
// cmd/grove_channel.go
var groveChannelCmd = &cobra.Command{
    Use:   "channel",
    Short: "Manage nixpkgs channels for Grove environment",
    Long:  `Manage nixpkgs channels including adding, removing, pinning, and updating channels.`,
}
```

### Subcommands to Implement

1. **`kanuka grove channel list`** - Show all configured channels
2. **`kanuka grove channel add <name> <url>`** - Add new nixpkgs channel
3. **`kanuka grove channel remove <name>`** - Remove channel
4. **`kanuka grove channel pin <name> <commit>`** - Pin channel to specific commit
5. **`kanuka grove channel show <name>`** - Show channel details
6. **`kanuka grove channel update <name>`** - Update channel to latest

## Technical Implementation Strategy

### 1. Data Storage Approach
**Store channel configuration in `devenv.yaml`** (following devenv patterns):
```yaml
# devenv.yaml
inputs:
  nixpkgs:
    url: "github:NixOS/nixpkgs/nixos-unstable"
  nixpkgs-stable:
    url: "github:NixOS/nixpkgs/nixos-23.11"
  custom-channel:
    url: "github:MyOrg/nixpkgs/custom-branch"
```

### 2. Core Functions to Add to `internal/grove/channels.go`

```go
type ChannelConfig struct {
    Name   string `yaml:"name"`
    URL    string `yaml:"url"`
    Commit string `yaml:"commit,omitempty"`
}

// Channel management functions
func ListChannels() ([]ChannelConfig, error)
func AddChannel(name, url string) error
func RemoveChannel(name string) error
func PinChannel(name, commit string) error
func UpdateChannel(name string) error
func ShowChannel(name string) (*ChannelConfig, error)
func ValidateChannelURL(url string) error
```

### 3. devenv.yaml Integration
Extend existing `internal/grove/filesystem.go` functions:
```go
func UpdateDevenvYamlChannels(channels []ChannelConfig) error
func GetChannelsFromDevenvYaml() ([]ChannelConfig, error)
```

## Implementation Order & Priority

### Phase 1: Foundation (High Priority) ✅ IMPLEMENTING NOW
1. **`grove channel list`** - Essential for discovering current state
   - Read from `devenv.yaml` inputs section
   - Display in user-friendly format with status indicators
   - Show which channel is currently used for packages

### Phase 2: Basic Management (High Priority)  
2. **`grove channel add`** - Core functionality
   - Validate URL format (github:owner/repo/branch pattern)
   - Check if channel name already exists
   - Update `devenv.yaml` inputs section
   - Verify channel accessibility

3. **`grove channel remove`** - Essential for cleanup
   - Check if channel is in use by packages
   - Confirm removal with user
   - Update `devenv.yaml`

### Phase 3: Advanced Features (Medium Priority)
4. **`grove channel show`** - Detailed information
   - Show URL, current commit, last updated
   - List packages using this channel
   - Show channel health/accessibility

5. **`grove channel pin`** - Version control
   - Pin to specific commit hash
   - Validate commit exists
   - Update `devenv.yaml` with pinned reference

### Phase 4: Maintenance (Lower Priority)
6. **`grove channel update`** - Keep channels current
   - Fetch latest commit for unpinned channels
   - Update `devenv.yaml` references
   - Show what changed

## File Structure

```
cmd/
├── grove_channel.go              # Main channel command ✅
├── grove_channel_list.go         # List channels ✅
├── grove_channel_add.go          # Add new channel
├── grove_channel_remove.go       # Remove channel
├── grove_channel_show.go         # Show channel details
├── grove_channel_pin.go          # Pin channel to commit
└── grove_channel_update.go       # Update channel

internal/grove/
├── channels.go                   # Extended with new functions ✅
└── filesystem.go                 # Extended with devenv.yaml channel handling ✅
```

## User Experience Design

### Success Messages (Following Kanuka Style)
```bash
kanuka grove channel add stable-custom github:MyOrg/nixpkgs/stable
# ✓ Added channel 'stable-custom'
# → Channel: github:MyOrg/nixpkgs/stable  
# → Use: kanuka grove add nodejs --channel stable-custom
```

### Error Handling
```bash
kanuka grove channel add existing-name github:other/repo
# ✗ Channel 'existing-name' already exists
# → Use 'kanuka grove channel remove existing-name' first
# → Or use 'kanuka grove channel show existing-name' to see current config
```

### Integration with Existing Commands
- **`grove add --channel <name>`** - Use custom channels
- **`grove status`** - Show channel information in environment status
- **`grove list`** - Indicate which channel each package uses

## Validation & Safety

### Input Validation
- Channel names: alphanumeric + hyphens only
- URLs: Must follow `github:owner/repo/branch` or `github:owner/repo/commit-hash` format
- Commits: Validate hash format and existence

### Safety Checks
- Prevent removing channels that have packages depending on them
- Warn when adding channels that might conflict
- Validate channel accessibility before adding
- Backup `devenv.yaml` before modifications

## Testing Strategy

### Integration Tests
```go
// test/integration/channel/
channel_add_test.go           # Test adding channels
channel_remove_test.go        # Test removing channels  
channel_list_test.go          # Test listing channels
channel_pin_test.go           # Test pinning functionality
channel_integration_test.go   # Test with grove add --channel
```

### Test Scenarios
- Add/remove channels in clean environment
- Handle existing `devenv.yaml` with custom inputs
- Channel validation and error cases
- Integration with package management
- Concurrent access safety

## Dependencies & Requirements

### External Dependencies
- No new external dependencies required
- Leverage existing YAML parsing in grove module
- Use existing HTTP client for URL validation

### System Requirements
- Same as existing Grove commands
- Network access for channel validation
- Git access for commit validation (optional enhancement)

## Migration & Backward Compatibility

### Existing Projects
- Existing `devenv.yaml` files will continue to work
- New channel commands only add functionality
- Default behavior unchanged for `grove add` without `--channel`

### Default Channels
- `unstable` (current default) maps to `nixpkgs` input
- `stable` maps to auto-detected stable channel
- Custom channels use their explicit names

## Implementation Status

- [x] Phase 1: `grove channel list` command
- [ ] Phase 2: `grove channel add` command
- [ ] Phase 2: `grove channel remove` command
- [ ] Phase 3: `grove channel show` command
- [ ] Phase 3: `grove channel pin` command
- [ ] Phase 4: `grove channel update` command

This implementation plan provides a complete, user-friendly channel management system that integrates seamlessly with the existing Grove functionality while following Kanuka's design patterns and user experience guidelines.