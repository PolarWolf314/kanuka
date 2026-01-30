// Package secrets provides cryptographic operations for Kanuka.
//
// This package handles the core encryption and decryption functionality,
// including RSA key pair management and symmetric key operations using
// NaCl secretbox.
//
// # Encryption Architecture
//
// Kanuka uses a hybrid encryption scheme:
//
//  1. A random 256-bit symmetric key encrypts the actual secret files
//  2. Each user's RSA public key encrypts a copy of the symmetric key
//  3. Users decrypt the symmetric key with their private key, then decrypt files
//
// This allows multiple users to access the same encrypted files without
// sharing private keys, and enables key rotation without re-distributing
// private keys.
//
// # Key Management
//
// RSA key pairs are generated during user registration:
//   - Private keys are stored in ~/.kanuka/keys/<project-uuid>/privkey
//   - Public keys are copied to the project's .kanuka/public_keys/ directory
//
// The symmetric key is stored encrypted for each user:
//   - Location: .kanuka/secrets/<user-uuid>.kanuka
//   - Encrypted with the user's RSA public key
//
// # File Operations
//
// Environment files (.env, .env.local, etc.) are encrypted in place:
//   - Original: .env
//   - Encrypted: .env.kanuka
//
// Encryption uses NaCl secretbox with a random 24-byte nonce prepended
// to the ciphertext. This means re-encrypting the same file produces
// different output (non-deterministic encryption).
//
// # Security Considerations
//
// Private keys should have 0600 permissions. The package warns when
// permissions are too permissive but does not enforce this to avoid
// breaking workflows.
//
// Symmetric keys are 32 bytes (256 bits) for AES-256 equivalent security.
// RSA keys are 4096 bits by default.
package secrets
