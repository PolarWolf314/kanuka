# Tests

## Running Tests

```bash
# Run all tests
go test ./test/...

# Run specific categories
go test ./test/integration/init/...
go test ./test/integration/encrypt/...
go test ./test/integration/decrypt/...
```

## Test Categories

### Init Tests

- **Basic**: Project initialization functionality
- **Cross Platform**: Platform-specific behavior
- **Environment**: Environment variable handling
- **Filesystem Edge Cases**: File system edge cases
- **Input Validation**: Input validation scenarios
- **Permissions**: Permission-related tests
- **State Recovery**: Recovery from corrupted states

### Encrypt Tests

- **Integration**: Core encrypt functionality
- **Filesystem Edge Cases**: File system edge cases
- **Permissions**: Permission and access control
- **Project State**: Project state edge cases

### Decrypt Tests

- **Integration**: Core decrypt functionality
- **Content Validation**: Content validation and integrity
- **Cryptographic**: Cryptographic edge cases
- **Filesystem Edge Cases**: File system edge cases
- **Project State**: Project state edge cases

## Test Structure

- `test/integration/shared/` - Common test utilities
- `test/integration/init/` - Project initialization tests
- `test/integration/encrypt/` - Encryption command tests
- `test/integration/decrypt/` - Decryption command tests

