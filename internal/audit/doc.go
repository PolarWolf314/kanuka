// Package audit provides audit trail logging for Kanuka operations.
//
// Every significant operation (encrypt, decrypt, register, revoke, etc.)
// is recorded in a project-level audit log. This provides accountability
// and helps teams understand who accessed secrets and when.
//
// # Log Format
//
// The audit log is stored as JSON Lines (one JSON object per line) at:
//
//	.kanuka/audit.jsonl
//
// Each entry contains:
//   - Timestamp (RFC3339 with microseconds, UTC)
//   - User email and UUID
//   - Operation name
//   - Operation-specific details (files, target users, etc.)
//
// # Usage
//
// Create an entry with user info pre-populated:
//
//	entry := audit.LogWithUser("encrypt")
//	entry.Files = encryptedFiles
//	audit.Log(entry)
//
// # Failure Handling
//
// Audit logging is best-effort. If logging fails (permissions, disk full,
// etc.), the operation continues without error. Operations should never
// fail just because audit logging failed.
//
// # Reading Logs
//
// Use ReadEntries() to parse the audit log for display or analysis.
// Malformed entries are silently skipped to handle partial writes.
package audit
