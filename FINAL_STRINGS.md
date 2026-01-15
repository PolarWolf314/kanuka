# Kānuka User-Facing Output Strings

This document catalogs all user-facing output strings in the Kānuka codebase, organized by file, string type, and whether the string is a FINAL output.

## String Types

- **Spinner Final Message**: Final message displayed by a spinner after completion
- **Println/Print**: Direct output to stdout/stderr
- **Error Message**: Error returned to user (wrapped in error)
- **JSON Output**: Machine-readable JSON output
- **Prompt**: Interactive user prompt
- **Table Output**: Tabular data display
- **Success Message**: Final success confirmation
- **Warning Message**: Warning information displayed to user

---

## cmd/secrets_register.go

### Spinner Final Messages (FINAL)

**Line 145** (context: User already has access, user continues)
```
✓ <email> has been granted access successfully!

Files created:
  Public key:    <path>
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 153** (context: Public key exists, kanuka file created)
```
✓ <email> has been granted access successfully!

Files created:
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 161** (context: Kanuka file exists, public key created)
```
✓ <email> has been granted access successfully!

Files created:
  Public key:    <path>

→ They now have access to decrypt the repository's secrets
```

**Line 173** (context: Both files already exist, update both)
```
✓ <email> access has been updated successfully!

Files updated:
  Public key:    <path>
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 190** (context: Public key would be created, kanuka updated)
```
✓ <email> has been granted access successfully!

Files updated:
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 219** (context: Neither file exists)
```
✓ <email> has been granted access successfully!

Files created:
  Public key:    <path>
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 237** (context: User already has access, update)
```
✓ <email> access has been updated successfully!

Files updated:
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 248** (context: Public key exists, update)
```
✓ <email> access has been updated successfully!

Files updated:
  Public key:    <path>

→ They now have access to decrypt the repository's secrets
```

**Line 272** (context: Kanuka file exists, public key updated)
```
✓ <email> access has been updated successfully!

Files updated:
  Public key:    <path>

→ They now have access to decrypt the repository's secrets
```

**Line 286** (context: Both files already exist, update public key)
```
✓ <email> access has been updated successfully!

Files updated:
  Public key:    <path>

→ They now have access to decrypt the repository's secrets
```

**Line 299** (context: User already has access, force skips confirmation)
```
✓ <email> has been granted access successfully!

Files created:
  Public key:    <path>
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 312** (context: User cancelled)
```
⚠ Registration cancelled.
```

**Line 329** (context: Custom file registration success)
```
✓ <display_name> has been granted access successfully!

Files created:
  Public key:    <path>
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 340** (context: Public key exists, update)
```
✓ <display_name> access has been updated successfully!

Files updated:
  Public key:    <path>

→ They now have access to decrypt the repository's secrets
```

**Line 391** (context: Neither file exists)
```
✓ <display_name> has been granted access successfully!

Files created:
  Public key:    <path>
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 481** (context: Kanuka file exists)
```
✓ <display_name> has been granted access successfully!

Files updated:
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 494** (context: Public key exists)
```
✓ <display_name> has been granted access successfully!

Files updated:
  Public key:    <path>

→ They now have access to decrypt the repository's secrets
```

**Line 503** (context: Both files already exist)
```
✓ <display_name> access has been updated successfully!

Files updated:
  Public key:    <path>
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 519** (context: Public key updated)
```
✓ <display_name> has been granted access successfully!

Files created:
  Public key:    <path>

→ They now have access to decrypt the repository's secrets
```

**Line 533** (context: Kanuka file updated)
```
✓ <display_name> access has been updated successfully!

Files created:
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 551** (context: Both files already exist, update public key)
```
✓ <display_name> access has been updated successfully!

Files updated:
  Public key:    <path>

→ They now have access to decrypt the repository's secrets
```

**Line 565** (context: Public key updated)
```
✓ <display_name> access has been updated successfully!

Files updated:
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 582** (context: User cancelled)
```
⚠ Registration cancelled.
```

**Line 659** (context: Both files already exist, update both)
```
✓ <display_name> access has been updated successfully!

Files updated:
  Public key:    <path>
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 685** (context: Public key exists)
```
✓ <display_name> has been granted access successfully!

Files updated:
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 698** (context: Both files already exist, update public key)
```
✓ <display_name> has been granted access successfully!

Files updated:
  Public key:    <path>
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 704** (context: Kanuka file updated)
```
✓ <display_name> access has been updated successfully!

Files updated:
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 720** (context: Public key updated)
```
✓ <display_name> has been granted access successfully!

Files created:
  Public key:    <path>

→ They now have access to decrypt the repository's secrets
```

**Line 731** (context: Kanuka file updated)
```
✓ <display_name> access has been updated successfully!

Files created:
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 744** (context: Both files already exist, update public key)
```
✓ <display_name> access has been updated successfully!

Files updated:
  Public key:    <path>
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 760** (context: Public key updated)
```
✓ <display_name> has been granted access successfully!

Files created:
  Public key:    <path>

→ They now have access to decrypt the repository's secrets
```

**Line 772** (context: Kanuka file updated)
```
✓ <display_name> access has been updated successfully!

Files created:
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 784** (context: User cancelled)
```
⚠ Registration cancelled.
```

**Line 804** (context: Custom file, both files exist, update public key)
```
✓ <display_name> has been granted access successfully!

Files updated:
  Public key:    <path>

→ They now have access to decrypt the repository's secrets
```

**Line 821** (context: Both files exist, update kanuka)
```
✓ <display_name> access has been updated successfully!

Files updated:
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

**Line 832** (context: Public key updated)
```
✓ <display_name> has been granted access successfully!

Files created:
  Public key:    <path>

→ They now have access to decrypt the repository's secrets
```

**Line 903** (context: Neither file exists)
```
✓ <display_name> has been granted access successfully!

Files created:
  Public key:    <path>
  Encrypted key: <path>

→ They now have access to decrypt the repository's secrets
```

### Spinner Final Messages - Errors (FINAL)

**Line 143-146**: Missing required flag
```
✗ Either --user, --file, or --pubkey must be specified.
Run kanuka secrets register --help to see the available commands
```

**Line 151-154**: --pubkey requires --user
```
✗ When using --pubkey, the --user flag is required.
Specify a user email with --user
```

**Line 159-162**: Invalid email format
```
✗ Invalid email format: <email>
→ Please provide a valid email address
```

**Line 167-174**: Empty public key
```
✗ Invalid public key format provided
Error: public key text cannot be empty
```

**Line 188-191**: Failed to read private key from stdin
```
✗ Failed to read private key from stdin
Error: <error message>
```

**Line 217-220**: Project not initialized
```
✗ Kānuka has not been initialized
→ Run kanuka secrets init instead
```

**Line 227-230**: Invalid public key format
```
✗ Invalid public key format provided
Error: <error message>
```

**Line 236-238**: User not found in project
```
✗ User <email> not found in project
They must first run: kanuka secrets create --email <email>
```

**Line 247-250**: Couldn't get kanuka key
```
✗ Couldn't get your Kānuka key from <path>

Are you sure you have access?

Error: <error message>
```

**Line 278-287**: Couldn't get private key
```
✗ Couldn't get your private key <error source>

Are you sure you have access?

Error: <error message>
```

**Line 294-300**: Failed to decrypt kanuka key
```
✗ Failed to decrypt your Kānuka key using your private key: 
    Kānuka key path: <path>
    Private key path: <path>

Are you sure you have access?

Error: <error message>
```

**Line 312-313**: User cancelled overwrite
```
⚠ Registration cancelled.
```

**Line 326-341**: Failed to save public key
```
✗ Failed to save public key to <path>
Error: <error message>
```

**Line 338-341**: Failed to register user
```
✗ Failed to register user with the provided public key
Error: <error message>
```

**Line 475-483**: TOML config error
```
✗ Failed to load project configuration.

→ The .kanuka/config.toml file is not valid TOML.
   <error message>

   To fix this issue:
   1. Restore the file from git: git checkout .kanuka/config.toml
   2. Or contact your project administrator for assistance
```

**Line 492-495**: Project not initialized
```
✗ Kānuka has not been initialized
→ Run kanuka secrets init instead
```

**Line 501-504**: User not found in project
```
✗ User <email> not found in project
They must first run: kanuka secrets create --email <email>
```

**Line 517-520**: Public key not found
```
✗ Public key for user <email> not found
They must first run: kanuka secrets create --email <email>
```

**Line 531-534**: Couldn't get kanuka key
```
✗ Couldn't get your Kānuka key from <path>

→ You don't have access to this project. Run kanuka secrets create to generate your keys
```

**Line 543-552**: Couldn't get private key
```
✗ Couldn't get your private key <error source>

→ You don't have access to this project. Run kanuka secrets create to generate your keys
```

**Line 561-567**: Failed to decrypt kanuka key
```
✗ Failed to decrypt your Kānuka key using your private key: 
    Kānuka key path: <path>
    Private key path: <path>

→ You don't have access to this project. Run kanuka secrets create to generate your keys
```

**Line 582-583**: User cancelled overwrite
```
⚠ Registration cancelled.
```

**Line 673-685**: Not initialized
```
✗ Kānuka has not been initialized
→ Run kanuka secrets init instead
```

**Line 697-699**: Invalid file path
```
✗ <path> is not a valid path to a public key file
```

**Line 709-721**: Invalid filename (not UUID)
```
✗ Public key file must be named <uuid>.pub

→ Rename your public key file to use UUID, or use --user and --pubkey flags instead

Example:
  mv /tmp/mykey.pub /tmp/550e8400-e29b-41d4-a716-4466554400000.pub
  kanuka secrets register --file /tmp/550e8400-e29b-41d4-a716-4466554400000.pub

Or:
  kanuka secrets register --user user@example.com --pubkey "$(cat /tmp/mykey.pub)"
```

**Line 728-732**: Failed to load public key
```
✗ Public key could not be loaded from <path>

Error: <error message>
```

**Line 739-745**: Couldn't get kanuka key
```
✗ Couldn't get your Kānuka key from <path>

Are you sure you have access?

Error: <error message>
```

**Line 761-772**: Failed to decrypt kanuka key
```
✗ Failed to decrypt your Kānuka key using your private key: 
    Kānuka key path: <path>
    Private key path: <path>

Are you sure you have access?

Error: <error message>
```

**Line 781-785**: UUID not found in project
```
✗ UUID <uuid> not found in project

→ To register a new user with this UUID, provide the --user flag:
   kanuka secrets register --file <customFilePath> --user <email>
```

**Line 804-805**: User cancelled overwrite
```
⚠ Registration cancelled.
```

**Line 819-821**: Failed to copy public key to project
```
✗ Failed to copy public key to <path>
Error: <error message>
```

**Line 831-833**: Failed to update project config
```
✗ Failed to update project config
Error: <error message>
```

### Dry-Run Output (FINAL)

**Line 911-927**: Dry-run preview
```
[dry-run] Would register <email>

Files that would be created:
  - <pubKeyPath>
  - <kanukaPath>

Prerequisites verified:
  ✓ User exists in project config
  ✓ Public key found at <pubKeyPath>
  ✓ Current user has access to decrypt symmetric key

No changes made. Run without --dry-run to execute.
```

**Line 934-946**: Dry-run preview (no pubkey to create)
```
[dry-run] Would register <display_name>

Files that would be created:
  - <kanukaPath>

Prerequisites verified:
  ✓ Public key loaded from <pubKeyPath>
  ✓ Current user has access to decrypt symmetric key

No changes made. Run without --dry-run to execute.
```

### Warning Prompt (FINAL)

**Line 69-76**: Confirm overwrite warning
```
⚠ Warning: <email> already has access to this project.
  Continuing will replace their existing key.
  If they generated a new keypair, this is expected.
  If not, they may lose access.

Do you want to continue? [y/N]:
```

---

## cmd/secrets_revoke.go

### Spinner Final Messages - Success (FINAL)

**Line 654-668**: Revocation complete
```
✓ Access for <display_name> has been revoked successfully!
→ Revoked: <file1>, <file2>
→ All secrets have been re-encrypted with a new key
⚠ Warning: <display_name> may still have access to old secrets from their local git history.
→ If necessary, rotate your actual secret values after this revocation.
```

### Spinner Final Messages - Errors (FINAL)

**Line 124-126**: --device requires --user
```
✗ The --device flag requires --user flag.
Run kanuka secrets revoke --help to see the available commands.
```

**Line 132-134**: Missing required flag
```
✗ Either --user or --file flag is required.
Run kanuka secrets revoke --help to see the available commands.
```

**Line 152-154**: Cannot specify both --user and --file
```
✗ Cannot specify both --user and --file flags.
Run kanuka secrets revoke --help to see the available commands.
```

**Line 159-162**: Invalid email format
```
✗ Invalid email format: <email>
→ Please provide a valid email address
```

**Line 171-173**: Project not initialized
```
✗ Kānuka has not been initialized
→ Run kanuka secrets init first
```

**Line 183-185**: Project not found
```
✗ Kānuka project not found
→ Run kanuka secrets init first
```

**Line 237-239**: User not found
```
✗ User <email> not found in this project
→ No devices found for this user
```

**Line 247-253**: Device not found
```
✗ Device <device_name> not found for user <email>
→ Available devices:
    - <device1>
    - <device2>
```

**Line 264-269**: Multiple devices warning + prompt
```
⚠ Warning: <email> has <count> devices:
  - <device1> (created: <date>)
  - <device2> (created: <date>)

This will revoke ALL devices for this user.

Proceed? [y/N]:
```

**Line 279-281**: User cancelled
```
⚠ Revocation cancelled
```

**Line 304-306**: No files found for user
```
✗ No files found for user <email>
```

**Line 333-346**: File check errors
```
✗ Failed to check user's public key file
Error: <error message>
```

```
✗ Failed to check user's kanuka key file
Error: <error message>
```

**Line 348-354**: User does not exist
```
✗ User <display_name> does not exist in this project
→ No files found for this user
```

**Line 378-397**: File check errors
```
✗ Failed to check public key file
Error: <error message>
```

```
✗ Failed to check kanuka key file
Error: <error message>
```

**Line 410-423**: Directory/file validation errors
```
✗ Failed to resolve file path: <error>
```

```
✗ File <path> does not exist
```

```
✗ File <path> is a directory, not a file
```

```
✗ Failed to resolve project secrets path: <error>
```

```
✗ File <path> is not in the project secrets directory
→ Expected directory: <path>
```

```
✗ File <path> is not a .kanuka file
```

**Line 438-454**: Project path/file errors
```
✗ Failed to resolve project secrets path: <error>
```

```
✗ Failed to check public key file
Error: <error message>
```

**Line 468-509**: Revocation errors
```
✗ Failed to completely revoke files for <display_name>
Error: <error1>
Error: <error2>
Warning: Some files were revoked successfully
```

**Line 589-593**: Config update error
```
✗ Files were revoked but failed to update project config: <error>
```

**Line 600-618**: Key rotation errors
```
✗ Files were revoked but failed to rotate key: <error>
```

```
✗ Files were revoked but failed to load private key <key source>: <error>
```

```
✗ Files were revoked but failed to re-encrypt secrets: <error>
```

**Line 634-668**: Final success message
```
✓ Access for <display_name> has been revoked successfully!
→ Revoked: <file1>, <file2>
→ All secrets have been re-encrypted with a new key
⚠ Warning: <display_name> may still have access to old secrets from their local git history.
→ If necessary, rotate your actual secret values after this revocation.
```

### Dry-Run Output (FINAL)

**Line 468-509**: Dry-run preview
```
[dry-run] Would revoke access for <display_name>

Files that would be deleted:
  - <file1>
  - <file2>
  - <file3>

Config changes:
  - Remove user <uuid> from project

Post-revocation actions:
  - Generate new encryption key
  - Re-encrypt symmetric key for <count> remaining user(s)
  - Re-encrypt <count> secret file(s) with new key

⚠ Warning: After revocation, <display_name> may still have access to old secrets from git history.

No changes made. Run without --dry-run to execute.
```

---

## cmd/secrets_encrypt.go

### Spinner Final Messages - Success (FINAL)

**Line 261-266**: Encryption complete
```
✓ Environment files encrypted successfully!
The following files were created: <formatted list>
→ You can now safely commit all .kanuka files to version control
Note: Encryption is non-deterministic for security reasons.
       Re-encrypting unchanged files will produce different output.
```

### Spinner Final Messages - Errors (FINAL)

**Line 86-88**: Project not initialized
```
✗ Kānuka has not been initialized
→ Run kanuka secrets init first
```

**Line 102-104**: File resolution error
```
✗ <error message>
```

**Line 118-121**: No environment files found
```
✗ No environment files found in <project path>
```

**Line 140-166**: Project config load error
```
✗ Failed to load project configuration.

→ The .kanuka/config.toml file is not valid TOML.
   <error message>

   To fix this issue:
   1. Restore the file from git: git checkout .kanuka/config.toml
   2. Or contact your project administrator for assistance
```

**Line 160-166**: Cannot get kanuka file
```
✗ Failed to get your .kanuka file. Are you sure you have access?

→ You don't have access to this project. Ask someone with access to run:
   kanuka secrets register --user <your-email>
```

**Line 176-189**: Failed to read private key from stdin
```
✗ Failed to read private key from stdin
Error: <error message>
```

**Line 184-188**: Failed to parse private key from stdin
```
✗ Failed to parse private key from stdin
→ Ensure your private key is in valid format (PEM or OpenSSH)
```

**Line 197-202**: Cannot get private key file
```
✗ Failed to get your private key file. Are you sure you have access?

→ You don't have access to this project. Ask someone with access to run:
   kanuka secrets register --user <your-email>
```

**Line 221-227**: Failed to decrypt kanuka key
```
✗ Failed to decrypt your .kanuka file. Are you sure you have access?

→ Your encrypted key file appears to be corrupted.
   Try asking the project administrator to revoke and re-register your access.

Error: <error message>
```

**Line 239-245**: Encrypt files error
```
✗ Failed to encrypt to project's .env files. Are you sure you have access?

Error: <error message>
```

### Dry-Run Output (FINAL)

**Line 276-292**: Dry-run preview
```
[dry-run] Would encrypt <count> environment file(s)

Files that would be created:
  <env_path> → <kanuka_file>
  <env_path2> → <kanuka_file2>

No changes made. Run without --dry-run to execute.
```

---

## cmd/secrets_decrypt.go

### Spinner Final Messages - Success (FINAL)

**Line 258-262**: Decryption complete
```
✓ Environment files decrypted successfully!
The following files were created:<formatted list>
→ Your environment files are now ready to use
```

### Spinner Final Messages - Errors (FINAL)

**Line 84-87**: Project not initialized
```
✗ Kānuka has not been initialized
→ Run kanuka secrets init first
```

**Line 100-103**: File resolution error
```
✗ <error message>
```

**Line 116-119**: No encrypted files found
```
✗ No encrypted environment (.kanuka) files found in <project path>
```

**Line 138-144**: Project config load error
```
✗ Failed to load project configuration.

→ The .kanuka/config.toml file is not valid TOML.
   <error message>

   To fix this issue:
   1. Restore the file from git: git checkout .kanuka/config.toml
   2. Or contact your project administrator for assistance
```

**Line 158-163**: Cannot get kanuka file
```
✗ Failed to obtain your .kanuka file. Are you sure you have access?
Error: <error message>
```

**Line 172-183**: Failed to read private key from stdin
```
✗ Failed to read private key from stdin
Error: <error message>
```

**Line 180-183**: Failed to parse private key from stdin
```
✗ Failed to parse private key from stdin
Error: <error message>
```

**Line 192-196**: Cannot get private key file
```
✗ Failed to get your private key file. Are you sure you have access?
Error: <error message>
```

**Line 214-220**: Failed to decrypt kanuka key
```
✗ Failed to decrypt your .kanuka file. Are you sure you have access?

→ Your encrypted key file appears to be corrupted.
   Try asking the project administrator to revoke and re-register your access.

Error: <error message>
```

**Line 232-237**: Decrypt files error
```
✗ Failed to decrypt project's .kanuka files. Are you sure you have access?
Error: <error message>
```

### Dry-Run Output (FINAL)

**Line 271-302**: Dry-run preview
```
[dry-run] Would decrypt <count> encrypted file(s)

Files that would be created:
  <kanuka_file> → <env_path> (<status>)
  <kanuka_file2> → <env_path2> (status2)

⚠ Warning: <count> existing file(s) would be overwritten.

No changes made. Run without --dry-run to execute.
```

---

## cmd/config_init.go

### Println/Print Output (FINAL)

**Line 74**: Welcome message
```
Welcome to Kanuka! Let's set up your identity.
```

**Line 81-82**: Using email from flag (verbose)
```
[info] Using email from flag: <email>
```

**Line 97-98**: Email validated
```
[info] Email validated successfully
```

**Line 105-106**: Using display name from flag
```
[info] Using display name from flag: <name>
```

**Line 121-122**: Using device name from flag
```
[info] Using device name from flag: <name>
```

**Line 141-142**: Device name validated
```
[info] Device name validated: <name>
```

**Line 153-154**: Generated new user UUID
```
[info] Generated new user UUID: <uuid>
```

**Line 167-168**: User config saved
```
[info] User configuration saved to <path>/config.toml
```

**Line 172-181**: User config saved success
```
✓ User configuration saved to <path>/config.toml

Your settings:
  Email:   <email>
  Name:    <name>
  Device:  <device>
  User ID: <uuid>
```

**Line 245-256**: User config already exists
```
✓ User configuration already exists

Your settings:
  Email:   <email>
  Name:    <name>
  Device:  <device>
  User ID: <uuid>

→ Run with flags to update: kanuka config init --email new@email.com
```

**Line 272-273**: Invalid email format
```
✗ Invalid email format: <email>
```

**Line 287-288**: Invalid device name
```
✗ Invalid device name: <device>
```

**Line 312-323**: User config updated
```
✓ User configuration updated

Your settings:
  Email:   <email>
  Name:    <name>
  Device:  <device>
  User ID: <uuid>
```

### Println/Print Output - Error (FINAL)

**Line 329**: Config setup error
```
✗ <error message>
```

---

## cmd/secrets_init.go

### Spinner Final Messages - Success (FINAL)

**Line 251-257**: Init complete
```
✓ Kānuka initialized successfully!

→ Run kanuka secrets encrypt to encrypt your existing .env files

Tip: Working in a monorepo? You have two options:
  1. Keep this single .kanuka at the root and use selective encryption:
     kanuka secrets encrypt services/api/.env
  2. Initialize separate .kanuka stores in each service:
     cd services/api && kanuka secrets init
```

### Spinner Final Messages - Errors (FINAL)

**Line 50-53**: Already initialized
```
✗ Kānuka has already been initialized
→ Run kanuka secrets create instead
```

**Line 74-75**: User config incomplete
```
✗ User configuration is incomplete
→ Run kanuka config init first to set up your identity
```

**Line 81-82**: Running initial setup
```
⚠ User configuration not found.
Running initial setup...
Initializing project...
```

**Line 90-92**: Project not initialized
```
✗ Kānuka has not been initialized
→ Run kanuka secrets init first to create a project
```

### Println/Print Output

**Line 96**: Project name prompt
```
Project name [<default>]:
```

---

## cmd/secrets_create.go

### Spinner Final Messages - Success (FINAL)

**Line 285-291**: Keys created
```
✓ Keys created for <email> (device: <device>)
    created: <path>
    <deleted> deleted: <path> (if existed)

To gain access to secrets in this project:
  1. Commit your <path> file to your version control system
  2. Ask someone with permissions to grant you access with:
     kanuka secrets register --user <userEmail>
```

### Spinner Final Messages - Errors (FINAL)

**Line 91-94**: Project not initialized
```
✗ Kānuka has not been initialized
→ Run kanuka secrets init first to create a project
```

**Line 99-105**: Project not initialized
```
✗ Kānuka has not been initialized
→ Run kanuka secrets init first to create a project
```

**Line 142-145**: Invalid email format
```
✗ Invalid email format: <email>
→ Please provide a valid email address
```

**Line 178-181**: Device name already in use
```
✗ Device name <device_name> is already in use for <email>
→ Choose a different device name with --device-name
```

**Line 203-206**: Public key already exists
```
✗ <path>.pub already exists
To override, run: kanuka secrets create --force
```

**Line 211-213**: Using --force flag
```
⚠ Using --force flag will overwrite existing keys - ensure you have backups
```

### Println/Print Output (FINAL)

**Line 41**: Email prompt
```
Enter your email:
```

---

## cmd/secrets_status.go

### Spinner Final Messages - Success (FINAL)

**Line 153**: Status displayed
```
✓ Status displayed
```

### Spinner Final Messages - Errors (FINAL)

**Line 84-85**: Project settings error
```
✗ Failed to initialize project settings
```

**Line 95-96**: Not initialized
```
✗ Kānuka has not been initialized
→ Run kanuka secrets init first
```

**Line 108-118**: Project config error
```
✗ Failed to load project configuration.

→ The .kanuka/config.toml file is not valid TOML.
   <error message>

   To fix this issue:
   1. Restore the file from git: git checkout .kanuka/config.toml
   2. Or contact your project administrator for assistance
```

**Line 119**: Config error (non-TOML)
```
✗ Failed to load project configuration
```

### Spinner Final Messages - JSON Error (FINAL)

**Line 92-93**: JSON not initialized error
```
{"error": "Kanuka has not been initialized"}
```

**Line 105-106**: JSON config error
```
{"error": "Failed to load project configuration: config.toml is not valid TOML"}
```

**Line 148-149**: Output status error
```
✗ Failed to output status
```

### Println/Print Output

**Line 277-343**: Table output
```
Project: <project_name>

Secret files status:
  FILE                                                 STATUS
  <file1>                                    ✓ encrypted (up to date)
  <file2>                                    ⚠ stale (plaintext modified after encryption)
  <file3>                                    ✗ not encrypted
  <file4>                                    ◌ encrypted only (no plaintext)

Summary:
  <count> file(s) up to date
  <count> file(s) stale (run 'kanuka secrets encrypt' to update)
  <count> file(s) not encrypted (run 'kanuka secrets encrypt' to secure)
  <count> file(s) encrypted only (plaintext removed, this is normal)
```

---

## cmd/secrets_log.go

### Spinner Final Messages - Success (FINAL)

**Line 201**: Audit log displayed
```
✓ Audit log displayed
```

### Spinner Final Messages - Errors (FINAL)

**Line 79-80**: Project settings error
```
✗ Failed to initialize project settings
```

**Line 87-88**: Not initialized
```
✗ Kānuka has not been initialized
→ Run kanuka secrets init first
```

**Line 95-96**: No audit log
```
ℹ No audit log found. Operations will be logged after running any secrets command.
```

**Line 102-103**: Audit log read error
```
✗ Failed to read audit log
```

**Line 106-107**: Parse error
```
✗ Failed to read audit log
```

**Line 183-191**: Output log error
```
✗ Failed to output log
```

**Line 196-199**: Output log error
```
✗ Failed to output log
```

**Line 201**: Final success (already set)
```
✓ Audit log displayed
```

### Println/Print Output

**Line 118**: No log entries
```
No audit log entries found.
```

**Line 176**: No log entries matching filters
```
No audit log entries found matching the filters.
```

**Line 279-289**: Log table output
```
<datetime>              <user>                        <operation>  <details>
2024-01-15 10:30:05    alice@example.com            encrypt      file1.env, file2.env
2024-01-14 11:22:15    bob@example.com             register     alice@example.com
2024-01-10 15:45:30    carol@example.com           revoke       alice@example.com (macbook)
```

**Line 276-282**: Log oneline output
```
<date> <user> <operation> <details>
2024-01-15 alice@example.com encrypt 3 files
2024-01-14 bob@example.com register alice@example.com
```

---

## cmd/secrets_doctor.go

### Spinner Final Messages - Success (FINAL)

**Line 167-169**: Health checks passed
```
✓ Health checks completed
```

**Line 168**: Health checks with warnings
```
⚠ Health checks completed with warnings
```

**Line 170**: Health checks with errors
```
✗ Health checks completed with errors
```

### Spinner Final Messages - Errors (FINAL)

**Line 160-161**: Output doctor results error
```
✗ Failed to output doctor results
```

### Println/Print Output

**Line 680-716**: Doctor results table
```
Running health checks...

✓ Project configuration valid
✓ User configuration valid
⚠ Private key has insecure permissions (0644)
✓ All public keys have corresponding encrypted symmetric keys
⚠ .env patterns not found in .gitignore

Summary: 1 passed, 1 warning

Suggestions:
  → Run 'chmod 600 <path>' to fix permissions
  → Add to .gitignore: .env, .env.*, and !*.kanuka (to keep encrypted files)
```

---

## cmd/secrets_access.go

### Spinner Final Messages - Success (FINAL)

**Line 48**: Access displayed
```
✓ Access information displayed
```

### Spinner Final Messages - Errors (FINAL)

**Line 79-80**: Project settings error
```
✗ Failed to initialize project settings
```

**Line 90-91**: Not initialized
```
✗ Kanuka has not been initialized
→ Run kanuka secrets init first
```

**Line 103-114**: Project config error (TOML)
```
✗ Failed to load project configuration.

→ The .kanuka/config.toml file is not valid TOML.
   <error message>

   To fix this issue:
   1. Restore the file from git: git checkout .kanuka/config.toml
   2. Or contact your project administrator for assistance
```

**Line 114**: Config error (non-TOML)
```
✗ Failed to load project configuration
```

**Line 141-142**: Output access error
```
✗ Failed to output access information
```

### Spinner Final Messages - JSON Error (FINAL)

**Line 87-88**: JSON not initialized error
```
{"error": "Kanuka has not been initialized"}
```

**Line 100-101**: JSON config error
```
{"error": "Failed to load project configuration: config.toml is not valid TOML"}
```

### Println/Print Output

**Line 296-343**: Access table
```
Project: <project_name>

Users with access:
  UUID                                  EMAIL                         STATUS
  <uuid1>                                alice@example.com (macbook)       ✓ active
  <uuid2>                                bob@example.com                ✓ active
  <uuid3>                                <email3> (workstation)    ✓ active
  <uuid4>                                carol@example.com              ⚠ pending
  <uuid5>                                <uuid5>                    ✗ orphan

Legend:
  ✓ active  - User has public key and encrypted symmetric key
  ⚠ pending - User has public key but no encrypted symmetric key (run 'sync')
  ✗ orphan  - Encrypted symmetric key exists but no public key (inconsistent)

Total: 5 user(s) (3 active, 1 pending, 1 orphan)

Tip: Run 'kanuka secrets clean' to remove orphaned entries.
```

**Line 300**: No users found
```
No users found.
```

---

## cmd/secrets_sync.go

### Spinner Final Messages - Success (FINAL)

**Line 105-107**: No encrypted files
```
✓ No encrypted files found. Nothing to sync.
```

**Line 116-119**: Sync complete
```
✓ Secrets synced successfully
  Re-encrypted <count> secret file(s) for <count> user(s).
  New encryption key generated and distributed to all users.
```

### Spinner Final Messages - Errors (FINAL)

**Line 51-54**: Project not initialized
```
✗ Kanuka has not been initialized
→ Run kanuka secrets init first
```

**Line 71-73**: Failed to load private key
```
✗ Failed to load your private key. Are you sure you have access?
Error: <error message>
```

**Line 89-92**: Sync error
```
✗ Failed to sync secrets
Error: <error message>
```

### Println/Print Output (FINAL)

**Line 126-148**: Dry-run output
```
[dry-run] Would sync secrets:

  No encrypted files found. Nothing to sync.

No changes made.
```

```
[dry-run] Would sync secrets:

  - Decrypt <count> secret file(s)
  - Generate new encryption key
  - Re-encrypt for <count> user(s)

No changes made.
```

---

## cmd/secrets_clean.go

### Spinner Final Messages - Success (FINAL)

**Line 78**: No orphans found
```
✓ No orphaned entries found. Nothing to clean.
```

**Line 22**: Orphans removed
```
✓ Removed <count> orphaned file(s)
```

### Spinner Final Messages - Errors (FINAL)

**Line 58-59**: Project not initialized
```
✗ Kanuka has not been initialized
→ Run kanuka secrets init first
```

**Line 73-74**: Find orphans error
```
✗ Failed to find orphaned entries
```

### Println/Print Output

**Line 84-87**: Dry-run preview
```
[dry-run] Would remove <count> orphaned file(s):

  UUID                     FILE
  <uuid1>                 .kanuka/secrets/<file1>.kanuka
  <uuid2>                 .kanuka/secrets/<file2>.kanuka

No changes made.
```

```
Found <count> orphaned entry(ies):

  UUID                     FILE
  <uuid1>                 .kanuka/secrets/<file1>.kanuka
  <uuid2>                 .kanuka/secrets/<file2>.kanuka

This will permanently delete the orphaned files listed above.
These files cannot be recovered.

Do you want to continue? [y/N]:
```

**Line 104**: User aborted
```
Aborted.
```

---

## cmd/secrets_import.go

### Spinner Final Messages - Success (FINAL)

**Line 80-93**: Dry-run complete
```
ℹ Dry run - no changes made

Mode: Merge
Total files in archive: <count>
  Added: <count>
  Skipped (already exist): <count>
```

**Line 92-93**: Import complete
```
✓ Imported secrets from <archive_path>

Mode: Merge
Total files in archive: <count>
  Added: <count>
  Skipped (already exist): <count>

Note: You may need to run kanuka secrets decrypt to decrypt secrets.
```

**Line 92-93**: Import complete (replace mode)
```
✓ Imported secrets from <archive_path>

Mode: Replace
Total files in archive: <count>
  Extracted: <count>

Note: You may need to run kanuka secrets decrypt to decrypt secrets.
```

### Spinner Final Messages - Errors (FINAL)

**Line 95-97**: Cannot use both flags
```
✗ Cannot use both --merge and --replace flags.

→ Use --merge to add new files while keeping existing files,
   or use --replace to delete existing files and use only of backup.
```

**Line 21-24**: Archive not found
```
archive file not found: <archive_path>
```

**Line 19-27**: Invalid archive
```
✗ Invalid archive file: <archive_path>

→ The file is not a valid gzip archive. Ensure it was created with:
   kanuka secrets export
```

**Line 429-435**: Invalid archive
```
✗ Invalid archive file: <archive_path>

→ The file is not a valid gzip archive. Ensure it was created with:
   kanuka secrets export
```

### Println/Print Output

**Line 300-304**: Import mode prompt
```
Found existing .kanuka directory. How do you want to proceed?
  [m] Merge - Add new files, keep existing
  [r] Replace - Delete existing, use backup
  [c] Cancel
Choice:
```

**Line 58**: Import cancelled
```
⚠ Import cancelled
```

---

## cmd/secrets_export.go

### Spinner Final Messages - Success (FINAL)

**Line 49-67**: Export complete
```
✓ Exported secrets to <output_path>

Archive contents:
  .kanuka/config.toml
  .kanuka/public_keys/ (<count> file(s))
  .kanuka/secrets/ (<count> user key(s))
  <count> encrypted secret file(s)

Note: This archive contains encrypted data only.
      Private keys are NOT included.
```

### Spinner Final Messages - Errors (FINAL)

**Line 83-86**: Project not initialized
```
✗ Kanuka has not been initialized
→ Run kanuka secrets init instead
```

**Line 91-94**: Config not found
```
✗ config.toml not found
→ Run kanuka secrets init to initialize a project
```

**Line 98-106**: Config validation error
```
✗ Failed to load project configuration.

→ <error message>

→ To fix this issue:
   1. Restore from git: git checkout .kanuka/config.toml
   2. Or contact your project administrator for assistance

Error: <error message>
```

**Line 29**: No files to export
```
⚠ No files found to export
```

---

## cmd/secrets_rotate.go

### Spinner Final Messages - Success (FINAL)

**Line 42-46**: Rotation complete
```
✓ Keypair rotated successfully

Your new public key has been added to the project.
Other users do not need to take any action.
→ Commit updated .kanuka/public_keys/<userUUID>.pub file
```

### Spinner Final Messages - Errors (FINAL)

**Line 100-103**: Project not initialized
```
✗ Kanuka has not been initialized
→ Run kanuka secrets init instead
```

**Line 129-132**: No access
```
✗ You don't have access to this project
→ Run kanuka secrets create and ask someone to register you
```

**Line 141-144**: Cannot load private key
```
✗ Couldn't load your private key from <path>

Error: <error message>
```

**Line 152-155**: Cannot get kanuka key
```
✗ Couldn't get your Kanuka key from <path>

Error: <error message>
```

**Line 161-164**: Failed to decrypt kanuka key
```
✗ Failed to decrypt your Kanuka key

Error: <error message>
```

**Line 71**: Rotation cancelled
```
⚠ Keypair rotation cancelled.
```

### Println/Print Output

**Line 42-43**: Rotation warning
```
⚠ Warning: This will generate a new keypair and replace your current one.
  Your old private key will no longer work for this project.

Do you want to continue? [y/N]:
```

---

## cmd/config_show.go

### Spinner Final Messages - Success (FINAL)

**Line 103-104**: User config displayed
```
✓ User configuration displayed
```

**Line 224-225**: User config displayed
```
✓ User configuration displayed
```

### Spinner Final Messages - Errors (FINAL)

**Line 70-72**: Failed to init user settings
```
✗ Failed to initialize user settings
```

**Line 77-79**: Failed to load user config
```
✗ Failed to load user configuration
```

**Line 84-90**: No user config
```
⚠ No user configuration found.

→ Run kanuka config init to set up your identity
```

**Line 100-102**: Output user config error
```
✗ Failed to output user configuration
```

**Line 108-109**: Output user config error
```
✗ Failed to output user configuration
```

### Spinner Final Messages - JSON Success (FINAL)

**Line 85**: No user config (JSON)
```
{}
```

**Line 203-204**: Project config displayed
```
✓ Project configuration displayed
```

**Line 216-217**: Output project config error
```
✗ Failed to output project configuration
```

**Line 224-225**: Project config displayed
```
✓ Project configuration displayed
```

### Spinner Final Messages - JSON Error (FINAL)

**Line 84-87**: Not in project (JSON)
```
{"error": "not in a project directory"}
```

**Line 101-102**: Config error (JSON)
```
{"error": "Failed to load project configuration: .kanuka/config.toml is not valid TOML"}
```

### Println/Print Output

**Line 27-37**: User config output
```
User Configuration (~/.config/kanuka/config.toml):

  Email:           <email>
  Name:            <name>
  User ID:         <uuid>
  Default Device: <device>

Projects:
  <uuid1> → <device1> (project1)
  <uuid2> → <device2> (project2)
```

**Line 40-62**: Project config output
```
Project Configuration (.kanuka/config.toml):

  Project ID:   <uuid>
  Project Name: <name>

Users:
  <email1> (uuid1)
     - <device1> (created: <date>)
     - <device2> (created: <date>)

  <email2> (uuid2)
     - <device3> (created: <date>)
```

**Line 88-90**: No user config
```
No user configuration found.

→ Run kanuka config init to set up your identity
```

---

## cmd/config_list_devices.go

### Spinner Final Messages - Success (FINAL)

**Line 142**: Devices listed
```
✓ Devices listed successfully
```

### Spinner Final Messages - Errors (FINAL)

**Line 50-52**: Failed to init project settings
```
✗ Failed to initialize project settings

→ Make sure you're in a Kānuka project directory
```

**Line 57-60**: Not in project
```
✗ Not in a Kānuka project directory

→ Run this command from within a Kānuka project
```

**Line 68-70**: No devices found
```
⚠ No devices found in this project
```

**Line 97-98**: User not found
```
✗ User <email> not found in this project
```

### Println/Print Output

**Line 14-16**: Devices in project
```
Devices in project <project_name>:

  <email1>
    - <device1> (UUID: <short_uuid>) - created: <date>
    - <device2> (UUID: <short_uuid2>) - created: <date2>

  <email2>
    - <device3> (UUID: <short_uuid3>) - created: <date3>
```

**Line 18**: Devices in this project
```
Devices in this project:

  <email1>
    - <device1> (UUID: <short_uuid>) - created: <date>
    - <device2> (UUID: <short_uuid2>) - created: <date2>
```

---

## cmd/config_set_default_device.go

### Spinner Final Messages - Success (FINAL)

**Line 65**: Device name set
```
✓ Default device name set to <device_name>
```

### Spinner Final Messages - Errors (FINAL)

**Line 39-42**: Invalid device name
```
✗ Invalid device name: <device_name>
→ Device name must be alphanumeric with hyphens and underscores only
```

**Line 51**: Device name already set
```
⚠ Default device name is already set to <device_name>
```

### Spinner Final Messages - Warning (FINAL)

**Line 51**: Device name already set
```
⚠ Default device name is already set to <device_name>
```

---

## cmd/config_set_project_device.go

### Spinner Final Messages - Success (FINAL)

**Line 90**: Device name updated from
```
✓ Device name updated from <old_name> to <device_name> for project <project_name>
```

**Line 90**: Device name set
```
✓ Device name set to <device_name>
```

### Spinner Final Messages - Errors (FINAL)

**Line 59-62**: Invalid device name
```
✗ Invalid device name: <device_name>
→ Device name must be alphanumeric with hyphens and underscores only
```

**Line 67-68**: Invalid device name (same)
```
✗ Invalid device name: <device_name>
→ Device name must be alphanumeric with hyphens and underscores only
```

**Line 74-75**: Device name already set
```
⚠ Device name is already set to <device_name> for this project
```

**Line 82-84**: Failed to init project settings
```
✗ Failed to initialize project settings: <error>

→ Use --project-uuid to specify a project
```

**Line 89-90**: Not in project
```
✗ Not in a Kānuka project directory

→ Use --project-uuid to specify a project
```

**Line 102-106**: Config TOML error
```
✗ Failed to load project configuration: .kanuka/config.toml is not valid TOML

To fix this issue:
  1. Restore the file from git: git checkout .kanuka/config.toml
  2. Or contact your project administrator for assistance
Details: <error>
```

**Line 102-106**: Config TOML error (repeated)
```
✗ Failed to load project configuration: .kanuka/config.toml is not valid TOML

To fix this issue:
  1. Restore the file from git: git checkout .kanuka/config.toml
  2. Or contact your project administrator for assistance
Details: <error>
```

**Line 103-105**: Could not determine project UUID
```
✗ Could not determine project UUID

→ Use --project-uuid to specify a project
```

**Line 81-82**: Device not found in config
```
Device not found in project config - only user config updated
```

### Spinner Final Messages - Warning (FINAL)

**Line 81-82**: Device not found
```
Device not found in project config - only user config updated
```

---

## main.go

### Println/Print Output (FINAL)

**Line 27**: Welcome message
```
Welcome to Kānuka! Run 'kanuka --help' to see available commands.
```

### Println/Print Output - Error (FINAL)

**Line 37**: Command execution error
```
<error message>
```

---

## internal/secrets/keys.go

### Stderr Println/Print Output

**Line 113**: Incorrect passphrase
```
✗ Incorrect passphrase. Please try again.
```

**Line 162**: Incorrect passphrase
```
✗ Incorrect passphrase. Please try again.
```

**Line 22**: Encrypted PKCS#8 keys not supported
```
encrypted PKCS#8 keys are not supported; please convert to OpenSSH format
```

**Line 29**: Cannot parse private key (wrong passphrase)
```
failed to decrypt private key after 3 attempts
```

**Line 71**: Cannot prompt for passphrase (no terminal)
```
private key is passphrase-protected but stdin is not a terminal; cannot prompt for passphrase
```

**Line 143**: Cannot prompt for passphrase (no TTY)
```
private key is passphrase-protected but no TTY available; cannot prompt for passphrase
```

**Line 171**: Failed to decrypt private key after 3 attempts
```
failed to decrypt private key after 3 attempts
```

---

## internal/secrets/files.go

### Error Messages (FINAL)

**Line 39**: No matching files found
```
no matching files found
```

**Line 65**: File not found
```
file not found: <pattern>
```

**Line 70**: Not .env file
```
file is not a .env file: <pattern>
```

**Line 73**: Not .kanuka file
```
file is not a .kanuka file: <pattern>
```

**Line 89**: Invalid glob pattern
```
invalid glob pattern <pattern>: <error>
```

---

## Summary

This document catalogs all user-facing output strings in the Kānuka CLI. The strings are categorized as:

- **FINAL**: The last message displayed to the user before the command exits
- **Spinner Final Message**: Displayed via spinner.FinalMSG when the spinner stops
- **Println/Print**: Direct output using fmt.Println or fmt.Printf
- **Error Message**: Wrapped in errors and displayed to the user
- **JSON Output**: Machine-readable JSON format output
- **Prompt**: Interactive user input prompts
- **Table Output**: Formatted tabular data display
- **Warning Message**: Warning information
- **Success Message**: Confirmation of successful operations

Most spinner messages marked with ✓, ✗, or ⚠ are FINAL strings that the user will see as the last output of the command.
