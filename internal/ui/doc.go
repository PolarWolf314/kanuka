// Package ui provides semantic text formatting for CLI output.
//
// This package defines formatters for different types of content (code,
// paths, errors, etc.) that render appropriately based on terminal
// capabilities. When colors are available, content is colorized. When
// NO_COLOR is set or the terminal doesn't support colors, text-based
// decorations (backticks, quotes) are used instead.
//
// # Semantic Formatters
//
// Use the appropriate formatter for the content type:
//
//	ui.Code.Sprint("kanuka secrets init")     // Commands and code
//	ui.Path.Sprint(".kanuka/config.toml")     // File paths
//	ui.Success.Sprint("✓")                     // Success indicators
//	ui.Error.Sprint("✗")                       // Error indicators
//	ui.Warning.Sprint("[dry-run]")             // Warnings
//	ui.Info.Sprint("→")                        // Informational hints
//	ui.Highlight.Sprint("user@example.com")   // User values
//	ui.Muted.Sprint("optional")               // De-emphasized text
//
// # Color Behavior
//
// Colors are disabled when:
//   - NO_COLOR environment variable is set (any value)
//   - Terminal doesn't support colors (TERM=dumb, not a TTY)
//
// When colors are disabled, formatters apply text decorations:
//   - Code: `backticks`
//   - Highlight: 'single quotes'
//   - Muted: (parentheses)
//   - Others: no decoration (self-evident from context)
package ui
