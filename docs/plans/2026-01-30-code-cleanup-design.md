# Code Documentation and Cleanup Design

## Overview

This document describes a comprehensive refactor to improve code readability, modularity, and documentation across the Kanuka codebase.

## Goals

- Readable, modular code with clear separation of concerns
- Package-level `doc.go` files plus thorough function documentation
- Remove "what" comments, keep only "why" comments
- Custom error types for programmatic handling instead of string matching

## New Packages

### internal/errors/

Sentinel errors for programmatic handling:

```go
var (
    ErrNoAccess              = errors.New("user does not have access to this project")
    ErrKeyNotFound           = errors.New("encryption key not found")
    ErrPrivateKeyNotFound    = errors.New("private key not found")
    ErrProjectNotInitialized = errors.New("project has not been initialized")
    ErrInvalidProjectConfig  = errors.New("project configuration is invalid")
    ErrKeyDecryptFailed      = errors.New("failed to decrypt symmetric key")
    ErrEncryptFailed         = errors.New("failed to encrypt file")
    ErrDecryptFailed         = errors.New("failed to decrypt file")
    ErrInvalidKeyLength      = errors.New("invalid symmetric key length")
    ErrNoFilesFound          = errors.New("no matching files found")
    ErrFileNotFound          = errors.New("file not found")
    ErrInvalidFileType       = errors.New("invalid file type")
)
```

### internal/workflows/

High-level command orchestration. Each workflow handles a single command's business logic, independent of CLI concerns.

Example workflow:

```go
func Encrypt(ctx context.Context, opts EncryptOptions) (*EncryptResult, error) {
    // Load configs, decrypt keys, encrypt files, audit log
}
```

## Package Structure

```
internal/
├── audit/           # Audit logging
├── configs/         # User and project configuration
├── errors/          # Custom error types
├── logging/         # Logging utilities
├── secrets/         # Low-level crypto operations
├── ui/              # Terminal UI helpers
├── utils/           # Filesystem, strings, system utilities
└── workflows/       # High-level command orchestration

cmd/
├── secrets.go                    # Root secrets command
├── secrets_encrypt.go            # Thin wrapper (~50 lines)
├── secrets_decrypt.go            # Thin wrapper
├── ... (other commands)
└── shared.go                     # Shared CLI helpers
```

## Documentation Style

### Package doc.go files

Each package gets a `doc.go` explaining purpose, key concepts, and usage:

```go
// Package workflows provides high-level orchestration for Kanuka commands.
//
// Workflows coordinate multiple operations across packages (configs, secrets,
// audit) to implement complete user-facing features. Each workflow handles
// a single command's business logic, independent of CLI concerns.
//
// # Encryption Workflow
//
// The encryption workflow coordinates:
//   - Loading user and project configuration
//   - Decrypting the symmetric key with the user's private key
//   - Encrypting environment files with the symmetric key
//   - Recording the operation in the audit log
//
// # Error Handling
//
// Workflows return typed errors from the errors package, allowing the CLI
// layer to provide appropriate user-facing messages without string matching.
package workflows
```

### Function documentation

```go
// EncryptFiles encrypts environment files using the project's symmetric key.
//
// It decrypts the user's copy of the symmetric key using their private key,
// then encrypts each input file with NaCl secretbox. The encrypted files are
// written alongside the originals with a .kanuka extension.
//
// Returns ErrNoAccess if the user doesn't have a key file for this project.
// Returns ErrKeyDecryptFailed if the private key cannot decrypt the symmetric key.
func EncryptFiles(ctx context.Context, opts EncryptOptions) (*EncryptResult, error)
```

## Thin Command Pattern

Commands in `cmd/` become thin wrappers:

```go
func runEncrypt(cmd *cobra.Command, args []string) error {
    spinner, cleanup := startSpinner("Encrypting...", verbose)
    defer cleanup()

    result, err := workflows.Encrypt(cmd.Context(), workflows.EncryptOptions{
        FilePatterns:    args,
        DryRun:          encryptDryRun,
        PrivateKeyStdin: encryptPrivateKeyStdin,
        Verbose:         verbose,
    })

    spinner.FinalMSG = formatEncryptResult(result, err)
    return nil
}
```

## Inline Comment Cleanup

Remove "what" comments:

```go
// Before
// 1. create sym key in memory
symKey, err := CreateSymmetricKey()

// After
symKey, err := CreateSymmetricKey()
```

Keep "why" comments:

```go
// NaCl secretbox prepends the nonce to the ciphertext
copy(decryptNonce[:], ciphertext[:24])

// #nosec G306 -- Decrypted .env files need to be user-editable
if err := os.WriteFile(outputPath, plaintext, 0644); err != nil {
```

## Implementation Phases

### Phase 1: Foundation

1. Create `internal/errors/` package with all error types
2. Create `internal/workflows/` package structure with `doc.go`
3. Add `doc.go` to existing packages (configs, secrets, utils, audit, ui)

### Phase 2: Refactor Commands

4. Extract `secrets encrypt` logic to `workflows/encrypt.go`
5. Extract `secrets decrypt` logic to `workflows/decrypt.go`
6. Extract `secrets init` logic to `workflows/init.go`
7. Extract `secrets register` logic to `workflows/register.go`
8. Extract `secrets revoke` logic to `workflows/revoke.go`
9. Extract `secrets rotate` logic to `workflows/rotate.go`
10. Extract remaining commands (status, clean, sync, access, etc.)

### Phase 3: Documentation Pass

11. Add thorough function docs to `internal/secrets/`
12. Add thorough function docs to `internal/configs/`
13. Add thorough function docs to `internal/utils/`
14. Add thorough function docs to `internal/workflows/`

### Phase 4: Cleanup

15. Remove inline "what" comments across all files
16. Run `golangci-lint` and fix any issues
17. Run full test suite to verify no regressions
