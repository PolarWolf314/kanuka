// Package configs manages user and project configuration for Kanuka.
//
// Configuration is stored in TOML format at two levels:
//
//   - User config: ~/.kanuka/config.toml (user identity, registered projects)
//   - Project config: .kanuka/config.toml (project settings, registered users)
//
// # User Configuration
//
// The user config stores:
//   - User identity (email, name, UUID)
//   - Default device name for new project registrations
//   - Map of registered projects with per-project device names
//
// The user UUID is auto-generated on first use and persists across all
// projects. This UUID identifies the user's encrypted key files.
//
// # Project Configuration
//
// The project config stores:
//   - Project identity (name, UUID)
//   - Map of registered users (UUID -> email)
//   - Map of registered devices (UUID -> device metadata)
//
// Device metadata includes the user's email, device name, and registration
// timestamp. A single email can have multiple devices (e.g., laptop, desktop).
//
// # Key Metadata
//
// Each project's keys are stored in ~/.kanuka/keys/<project-uuid>/ with
// a metadata.toml file tracking:
//   - Project name and path (for display purposes)
//   - Creation and last access timestamps
//
// # Settings
//
// Global settings are initialized at startup:
//   - UserKanukaSettings: paths to user config and keys directories
//   - ProjectKanukaSettings: current project's paths and identity
//
// Call InitProjectSettings() before accessing ProjectKanukaSettings.
// It walks up the directory tree to find the nearest .kanuka directory.
package configs
