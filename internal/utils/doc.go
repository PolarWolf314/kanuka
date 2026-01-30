// Package utils provides shared utility functions for the Kanuka application.
//
// This package contains general-purpose helpers used across multiple packages.
// Functions are organized into logical groups:
//
// # Filesystem Utilities
//
// Functions for working with the filesystem and project structure:
//   - FindProjectKanukaRoot: walks up directories to find .kanuka
//   - FormatPaths: formats file paths for human-readable output
//
// # System Utilities
//
// Functions for interacting with the operating system:
//   - GetUsername: returns the current system username
//   - GetHostname: returns the system hostname
//   - SanitizeDeviceName: normalizes device names for safe storage
//
// # Project Utilities
//
// Functions for working with Kanuka projects:
//   - GetProjectName: returns the current project's directory name
//
// # String Utilities
//
// Functions for string manipulation and formatting.
//
// # I/O Utilities
//
// Functions for reading from stdin and other I/O operations:
//   - ReadStdin: reads all data from standard input
//
// # Terminal Utilities
//
// Functions for terminal detection and interaction:
//   - IsTerminal: checks if a file descriptor is a terminal
package utils
