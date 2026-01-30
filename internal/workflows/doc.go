// Package workflows provides high-level orchestration for Kanuka commands.
//
// Workflows coordinate multiple operations across packages (configs, secrets,
// audit) to implement complete user-facing features. Each workflow handles
// a single command's business logic, independent of CLI concerns like flag
// parsing, spinners, and output formatting.
//
// # Design Philosophy
//
// The cmd/ package should be a thin layer that:
//   - Parses command-line flags and arguments
//   - Calls the appropriate workflow function
//   - Formats the result for display
//
// Workflows handle everything else:
//   - Loading configuration (user and project)
//   - Validating prerequisites and permissions
//   - Performing the core operation
//   - Recording audit trail entries
//
// # Available Workflows
//
// Each command has a corresponding workflow:
//
//   - Encrypt: Encrypts .env files using the project's symmetric key
//   - Decrypt: Decrypts .kanuka files back to .env files
//   - Init: Initializes a new Kanuka project
//   - Register: Registers a new user with an existing project
//   - Revoke: Removes a user's access to a project
//   - Rotate: Rotates the project's symmetric key
//
// # Error Handling
//
// Workflows return typed errors from the internal/errors package, allowing
// the CLI layer to provide appropriate user-facing messages without string
// matching. Use errors.Is() to check for specific error conditions:
//
//	result, err := workflows.Encrypt(ctx, opts)
//	if errors.Is(err, kerrors.ErrProjectNotInitialized) {
//	    // Show user-friendly initialization message
//	}
//
// # Context Usage
//
// All workflow functions accept a context.Context as their first parameter.
// This enables cancellation, timeouts, and passing request-scoped values.
package workflows
