// Package errors provides typed error values for the Kanuka application.
//
// Using sentinel errors allows callers to handle specific error conditions
// programmatically with errors.Is() rather than string matching. This makes
// error handling more robust and refactoring-safe.
//
// # Error Categories
//
// Errors are grouped by category:
//
//   - Access errors: User lacks permission or keys (ErrNoAccess, ErrKeyNotFound)
//   - Project errors: Project state issues (ErrProjectNotInitialized)
//   - Crypto errors: Encryption/decryption failures (ErrKeyDecryptFailed)
//   - File errors: File system issues (ErrNoFilesFound, ErrFileNotFound)
//
// # Usage
//
// Return errors from internal packages:
//
//	if projectPath == "" {
//	    return nil, errors.ErrProjectNotInitialized
//	}
//
// Handle errors in the CLI layer:
//
//	result, err := workflows.Encrypt(ctx, opts)
//	if errors.Is(err, kerrors.ErrProjectNotInitialized) {
//	    // Show user-friendly message
//	}
//
// Wrap errors with additional context:
//
//	return fmt.Errorf("loading key for user %s: %w", userID, errors.ErrKeyNotFound)
package errors
