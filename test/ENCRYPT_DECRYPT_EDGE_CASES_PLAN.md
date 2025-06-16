# Encrypt/Decrypt Edge Cases Implementation Plan

This document outlines the planned edge case tests for the encrypt and decrypt commands, filtered based on requirements and existing test coverage.

## 📊 **Existing Test Coverage Analysis**

### **Current Encrypt Tests:**

- ✅ `EncryptInEmptyFolder` - Try to encrypt when project not initialized
- ✅ `EncryptInInitializedFolderWithNoEnvFiles` - No .env files exist
- ✅ `EncryptInInitializedFolderWithOneEnvFile` - Basic encryption functionality
- ✅ `EncryptInInitializedFolderWithMultipleEnvFiles` - Multiple .env files
- ✅ `EncryptWithoutAccess` - Missing private key
- ✅ `EncryptFromSubfolderWithOneEnvFile` - Run from subfolder
- ✅ `EncryptFromSubfolderWithMultipleEnvFiles` - Run from subfolder with multiple files

### **Current Decrypt Tests:**

- ✅ `DecryptInEmptyFolder` - Try to decrypt when project not initialized
- ✅ `DecryptInInitializedFolderWithNoKanukaFiles` - No encrypted files exist
- ✅ `DecryptInInitializedFolderWithOneKanukaFile` - Basic decryption functionality
- ✅ `DecryptInInitializedFolderWithMultipleKanukaFiles` - Multiple encrypted files
- ✅ `DecryptWithoutAccess` - Missing private key
- ✅ `DecryptFromSubfolderWithOneKanukaFile` - Run from subfolder
- ✅ `DecryptFromSubfolderWithMultipleKanukaFiles` - Run from subfolder with multiple files

---

## 🎯 **NEW TESTS TO IMPLEMENT**

### **🔐 ENCRYPT Command Edge Cases**

#### **Category 1: File System Edge Cases**

1. **`EncryptWithEmptyEnvFile`** - Encrypt an empty .env file
2. **`EncryptWithReadOnlyEnvFile`** - .env file exists but is read-only
3. **`EncryptWithEnvFileAsDirectory`** - .env exists as a directory instead of file
4. **`EncryptWithEnvFileAsSymlink`** - .env is a symlink to another file
5. **`EncryptWithBrokenEnvSymlink`** - .env is a broken symlink
6. **`EncryptWithMultipleEnvFiles`** - Multiple .env files (.env, .env.local, .env.production) _(Enhanced version)_
7. **`EncryptWithVeryLargeEnvFile`** - Encrypt a very large .env file (MB+ size)
8. ~~**`EncryptWithBinaryDataInEnv`**~~ - _Excluded per requirements (test #9)_

#### **Category 3: Project State Edge Cases**

9. **`EncryptWithCorruptedKanukaDir`** - .kanuka directory is corrupted/incomplete
10. **`EncryptWithMissingPublicKey`** - Public key file is missing
11. **`EncryptWithMissingSymmetricKey`** - Symmetric key file is missing
12. **`EncryptWithCorruptedPublicKey`** - Public key file is corrupted
13. **`EncryptWithCorruptedSymmetricKey`** - Symmetric key file is corrupted
14. **`EncryptWithWrongKeyFormat`** - Key files have wrong format/content

#### **Category 4: Permission and Access Edge Cases**

15. **`EncryptWithReadOnlyKanukaDir`** - .kanuka directory is read-only
16. **`EncryptWithReadOnlySecretsDir`** - .kanuka/secrets directory is read-only
17. **`EncryptWithNoWritePermissionToProject`** - Can't write to project directory
18. ~~**`EncryptWithInsufficientDiskSpace`**~~ - _Excluded per requirements (test #29)_

---

### **🔓 DECRYPT Command Edge Cases**

#### **Category 1: File System Edge Cases**

19. **`DecryptWithCorruptedEncryptedFile`** - Encrypted file is corrupted
20. **`DecryptWithReadOnlyEncryptedFile`** - Encrypted file is read-only
21. **`DecryptWithEncryptedFileAsDirectory`** - Encrypted file path is a directory _(Enhanced version)_
22. **`DecryptWithMissingEncryptedFile`** - Specific encrypted file doesn't exist
23. **`DecryptWithVeryLargeEncryptedFile`** - Very large encrypted file
24. **`DecryptWithEmptyEncryptedFile`** - Empty encrypted file

#### **Category 2: Cryptographic Edge Cases**

25. **`DecryptWithWrongPrivateKey`** - Private key doesn't match
26. **`DecryptWithCorruptedPrivateKey`** - Private key file is corrupted
27. **`DecryptWithWrongKeyFormat`** - Private key has wrong format
28. **`DecryptWithTamperedEncryptedData`** - Encrypted data has been modified
29. **`DecryptWithWrongEncryptionAlgorithm`** - File encrypted with different algorithm

#### **Category 4: Project State Edge Cases**

30. **`DecryptWithCorruptedKanukaDir`** - .kanuka directory is corrupted
31. **`DecryptWithMissingUserKeys`** - User key files are missing

#### **Category 6: Content Validation Edge Cases**

32. **`DecryptAndValidateContent`** - Verify decrypted content matches original
33. **`DecryptWithInvalidEncryptedFormat`** - Encrypted file has wrong format

---

### **🔄 ENCRYPT/DECRYPT Integration Edge Cases**

#### **Category 8: Round-Trip Testing**

34. **`EncryptDecryptRoundTrip`** - Encrypt then decrypt, verify content matches
35. **`EncryptDecryptWithDifferentUsers`** - Multiple users encrypting/decrypting
36. **`EncryptDecryptWithKeyRotation`** - Test key rotation scenarios

#### **Category 9: Environment Variable Edge Cases**

37. **`EncryptDecryptWithEnvironmentOverrides`** - XDG_DATA_HOME, custom paths
38. **`EncryptDecryptWithInvalidEnvironment`** - Invalid environment settings

---

## 📋 **EXCLUDED TESTS** _(Per Requirements)_

### **Encrypt Exclusions:**

- ❌ Test #9: `EncryptWithBinaryDataInEnv` - Binary/non-text data in .env
- ❌ Test #29: `EncryptWithInsufficientDiskSpace` - Disk full during encryption
- ❌ All Category 2 (Content Format Edge Cases)
- ❌ All Category 5 (Concurrent Access Edge Cases)
- ❌ All Category 6 (Cross-Platform Edge Cases)
- ❌ All Category 7 (Recovery and Cleanup Edge Cases)

### **Decrypt Exclusions:**

- ❌ Test #44: `DecryptWithEncryptedFileAsDirectory` - Encrypted file path is a directory
- ❌ Test #61: `DecryptAfterKeyRotation` - Keys have been rotated/changed
- ❌ Test #67: `DecryptAndValidateContent` - Verify decrypted content matches original
- ❌ Test #68: `DecryptWithInvalidEncryptedFormat` - Encrypted file has wrong format
- ❌ Test #74: `EncryptDecryptStressTest` - Many encrypt/decrypt cycles
- ❌ All Category 3 (Output and Overwrite Edge Cases)
- ❌ All Category 5 (Multi-File and Selection Edge Cases)
- ❌ All Category 7 (Recovery and Cleanup Edge Cases)

---

## 🎯 **IMPLEMENTATION PRIORITY**

### **🔥 High Priority (Core User Scenarios)**

1. **File State Issues**: Empty files, corrupted files, missing files
2. **Permission Issues**: Read-only files and directories
3. **Key Problems**: Missing, corrupted, or wrong format keys
4. **Round-Trip Testing**: Basic encrypt/decrypt cycle validation

### **⚡ Medium Priority (Advanced Edge Cases)**

5. **Symlink Handling**: Symlinks and broken symlinks
6. **Large File Handling**: Very large .env files
7. **Environment Variables**: XDG_DATA_HOME and custom paths
8. **Multi-User Scenarios**: Different users encrypting/decrypting

### **🔍 Low Priority (Complex Scenarios)**

9. **Cryptographic Edge Cases**: Tampered data, wrong algorithms
10. **Key Rotation**: Advanced key management scenarios

---

## 📁 **PROPOSED FILE ORGANIZATION**

### **New Test Files to Create:**

```
test/integration/encrypt/
├── encrypt_integration_test.go          # Existing basic tests
├── encrypt_filesystem_edge_cases_test.go # Tests 1-8
├── encrypt_project_state_test.go         # Tests 9-14
└── encrypt_permissions_test.go           # Tests 15-17

test/integration/decrypt/
├── decrypt_integration_test.go           # Existing basic tests
├── decrypt_filesystem_edge_cases_test.go # Tests 19-24
├── decrypt_cryptographic_test.go         # Tests 25-29
├── decrypt_project_state_test.go         # Tests 30-31
└── decrypt_content_validation_test.go    # Tests 32-33

test/integration/roundtrip/
├── encrypt_decrypt_roundtrip_test.go     # Tests 34-36
└── encrypt_decrypt_environment_test.go   # Tests 37-38
```

---

## ✅ **READY FOR IMPLEMENTATION**

This plan covers **38 new edge case tests** across the encrypt and decrypt commands, focusing on:

- ✅ Real-world user scenarios
- ✅ Error condition handling
- ✅ Cross-platform compatibility
- ✅ Security and data integrity
- ✅ Performance with large files

**Total Test Coverage After Implementation:**

- **Encrypt**: 7 existing + 14 new = **21 test scenarios**
- **Decrypt**: 7 existing + 15 new = **22 test scenarios**
- **Round-Trip**: 0 existing + 9 new = **9 test scenarios**
- **Grand Total**: **52 comprehensive test scenarios**

The plan excludes tests that are either too complex for the current scope or don't provide significant value for typical user scenarios, as specified in the requirements.

