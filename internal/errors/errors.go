package errors

import "errors"

// Access errors indicate the user lacks permission or required keys.
var (
	// ErrNoAccess indicates the user does not have access to this project.
	ErrNoAccess = errors.New("user does not have access to this project")

	// ErrKeyNotFound indicates an encryption key could not be located.
	ErrKeyNotFound = errors.New("encryption key not found")

	// ErrPrivateKeyNotFound indicates the user's private key could not be located.
	ErrPrivateKeyNotFound = errors.New("private key not found")

	// ErrPublicKeyNotFound indicates a public key could not be located.
	ErrPublicKeyNotFound = errors.New("public key not found")
)

// Project state errors indicate issues with project configuration or initialization.
var (
	// ErrProjectNotInitialized indicates the project has not been set up with Kanuka.
	ErrProjectNotInitialized = errors.New("project has not been initialized")

	// ErrProjectAlreadyInitialized indicates the project has already been set up with Kanuka.
	ErrProjectAlreadyInitialized = errors.New("project has already been initialized")

	// ErrInvalidProjectConfig indicates the project configuration is malformed or corrupt.
	ErrInvalidProjectConfig = errors.New("project configuration is invalid")

	// ErrUserNotRegistered indicates the user is not registered with this project.
	ErrUserNotRegistered = errors.New("user is not registered with this project")
)

// Cryptographic errors indicate failures during encryption or decryption operations.
var (
	// ErrKeyDecryptFailed indicates the symmetric key could not be decrypted.
	ErrKeyDecryptFailed = errors.New("failed to decrypt symmetric key")

	// ErrEncryptFailed indicates file encryption failed.
	ErrEncryptFailed = errors.New("failed to encrypt file")

	// ErrDecryptFailed indicates file decryption failed.
	ErrDecryptFailed = errors.New("failed to decrypt file")

	// ErrInvalidKeyLength indicates the symmetric key has an unexpected length.
	ErrInvalidKeyLength = errors.New("invalid symmetric key length")

	// ErrInvalidPrivateKey indicates the private key is malformed or unsupported.
	ErrInvalidPrivateKey = errors.New("invalid or unsupported private key format")
)

// File errors indicate issues with file discovery or access.
var (
	// ErrNoFilesFound indicates no files matched the provided patterns.
	ErrNoFilesFound = errors.New("no matching files found")

	// ErrFileNotFound indicates a specific file could not be located.
	ErrFileNotFound = errors.New("file not found")

	// ErrInvalidFileType indicates the file is not of the expected type.
	ErrInvalidFileType = errors.New("invalid file type")

	// ErrInvalidArchive indicates the archive structure is invalid.
	ErrInvalidArchive = errors.New("invalid archive structure")
)

// User errors indicate issues with user-related operations.
var (
	// ErrUserNotFound indicates the specified user could not be found.
	ErrUserNotFound = errors.New("user not found")

	// ErrDeviceNotFound indicates the specified device could not be found.
	ErrDeviceNotFound = errors.New("device not found")

	// ErrSelfRevoke indicates a user attempted to revoke their own access.
	ErrSelfRevoke = errors.New("cannot revoke your own access")

	// ErrInvalidEmail indicates the email format is invalid.
	ErrInvalidEmail = errors.New("invalid email format")

	// ErrDeviceNameTaken indicates the device name is already in use.
	ErrDeviceNameTaken = errors.New("device name already in use")

	// ErrPublicKeyExists indicates a public key already exists for this user.
	ErrPublicKeyExists = errors.New("public key already exists")
)
