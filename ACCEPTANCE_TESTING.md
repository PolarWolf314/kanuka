# Kanuka Acceptance Testing

Manual acceptance testing document for the Kanuka CLI tool. Use this document to systematically test all CLI commands, find bugs, and document UX feedback.

## Test Environment Setup

### Prerequisites

1. **Go installed**: Verify with `go version`
2. **Build the CLI**:
   ```bash
   go build -o kanuka .
   ```
3. **Create test directories**: We recommend creating isolated test directories for each test scenario
   ```bash
   mkdir -p ~/kanuka-test
   cd ~/kanuka-test
   ```
4. **Clear any existing user config** (optional, for fresh-start testing):
   ```bash
   rm -rf ~/.config/kanuka ~/.local/share/kanuka
   ```

### Test Users

For multi-user testing, you'll need to simulate multiple users. You can:

1. Use different email addresses for the same machine
2. Use separate VMs or containers
3. Manually swap user config directories

---

## Section 1: Fresh Install / First Run

### TEST-001: First run with no configuration

**Command(s):**

```bash
./kanuka --version
./kanuka --help
```

**Preconditions:**

- No `~/.config/kanuka` directory exists
- No `~/.local/share/kanuka` directory exists

**Expected Result:**

- Version displays correctly
- Help displays all available commands
- No errors or warnings about missing configuration

**Notes:**

Pass

---

### TEST-002: Run secrets command without project initialization

**Command(s):**

```bash
./kanuka secrets encrypt
./kanuka secrets decrypt
./kanuka secrets access
./kanuka secrets status
```

**Preconditions:**

- Not in a Kanuka project directory (no `.kanuka` folder)

**Expected Result:**

- Clear error message indicating Kanuka is not initialized
- Suggestion to run `kanuka secrets init`

**Notes:**

`kanuka secrets encrypt` causes spinner to hang infinitely.
`kanuka secrets decrypt` causes spinner to hang infinitely.
`kanuka secrets access` says no user found (correct), but has hard-coded test-project (not correct).
`kanuka secrets stsatus` just hangs with no spinner.

---

### TEST-003: Run secrets create before init

**Command(s):**

```bash
./kanuka secrets create
```

**Preconditions:**

- Not in a Kanuka project directory

**Expected Result:**

- Error message indicating Kanuka has not been initialized
- Suggestion to run `kanuka secrets init` instead

**Notes:**

No error message. I ran with verbose to see the issue. Seems to be that keypair is created and attempts to copy to the kanuka directory fails because it doesn't exist:

```bash
aaron@Aarons-MacBook-Pro-2:acceptance_testing % kanuka secrets create -v
[info] Starting create command
[info] Running in verbose or debug mode: Creating Kānuka file...
[info] RSA key pair created successfully
Error: Failed to copy public key to project: failed to write key to project: open /Users/aaron/.kanuka/public_keys/ab26c005-c609-4e34-9277-6fd811700ad9.pub: no such file or directory
Usage:
  kanuka secrets create [flags]

Flags:
      --device-name string   custom device name (auto-generated from hostname if not specified)
  -e, --email string         your email address for identification
  -f, --force                force key creation
  -h, --help                 help for create

Global Flags:
  -d, --debug     enable debug output
  -v, --verbose   enable verbose output

Failed to copy public key to project: failed to write key to project: open /Users/aaron/.kanuka/public_keys/ab26c005-c609-4e34-9277-6fd811700ad9.pub: no such file or directory
```

---

## Section 2: User Configuration (`config` commands)

### TEST-010: Initialize user config interactively

**Command(s):**

```bash
./kanuka config init
```

**Preconditions:**

- No existing user config at `~/.config/kanuka/config.toml`

**Expected Result:**

- Prompts for email address
- Prompts for display name (optional)
- Prompts for default device name (defaults to hostname)
- Creates config file and shows summary
- User UUID is generated

**Notes:**

Pass

---

### TEST-011: Initialize user config with flags (non-interactive)

**Command(s):**

```bash
./kanuka config init --email test@example.com --name "Test User" --device test-laptop
```

**Preconditions:**

- No existing user config OR existing config to update

**Expected Result:**

- Config created/updated without prompts
- Values match provided flags
- Success message displayed

**Notes:**

Pass

---

### TEST-012: Config init with invalid email

**Command(s):**

```bash
./kanuka config init --email "not-an-email"
```

**Preconditions:**

- None

**Expected Result:**

- Error message about invalid email format
- No config changes made

**Notes:**

Pass

---

### TEST-013: Config init when already configured

**Command(s):**

```bash
./kanuka config init
```

**Preconditions:**

- User config already exists with email and UUID

**Expected Result:**

- Shows current configuration
- Suggests using flags to update
- Does not prompt for new values

**Notes:**

Pass

---

### TEST-014: Config show - user configuration

**Command(s):**

```bash
./kanuka config show
./kanuka config show --json
```

**Preconditions:**

- User config exists

**Expected Result:**

- Displays email, name, user ID, default device
- Shows list of projects with device names
- JSON format is valid and parseable

**Notes:**

Pass

---

### TEST-015: Config show - project configuration

**Command(s):**

```bash
./kanuka config show --project
./kanuka config show --project --json
```

**Preconditions:**

- In a Kanuka project directory

**Expected Result:**

- Displays project name, UUID
- Lists all users with their devices
- JSON format is valid and parseable

**Notes:**

Pass

---

### TEST-016: Config show --project outside project

**Command(s):**

```bash
cd /tmp
./kanuka config show --project
```

**Preconditions:**

- Not in a Kanuka project directory

**Expected Result:**

- Error: "Not in a Kanuka project directory"
- Suggestion to run `kanuka secrets init`

**Notes:**

Pass

---

### TEST-017: List devices in project

**Command(s):**

```bash
./kanuka config list-devices
./kanuka config list-devices --user alice@example.com
```

**Preconditions:**

- In a Kanuka project with registered users

**Expected Result:**

- Lists all devices grouped by email
- With `--user` flag, only shows that user's devices
- Shows device name, UUID prefix, and creation date

**Notes:**

Pass

---

### TEST-018: Set device name for current project

**Command(s):**

```bash
./kanuka config set-project-device new-device-name
```

**Preconditions:**

- In a Kanuka project directory

**Expected Result:**

- Updates device name in user config for this project
- Success message shows old and new name

**Notes:**

Success message shows, user config changes, but the project config doesn't change to reflect the new device name for the user.

---

### TEST-019: Set device name with invalid characters

**Command(s):**

```bash
./kanuka config set-project-device "my device name!"
```

**Preconditions:**

- In a Kanuka project directory

**Expected Result:**

- Error about invalid device name
- Explains valid format (alphanumeric, hyphens, underscores)

**Notes:**

Pass

---

### TEST-020: Rename device for another user

**Command(s):**

```bash
```

**Preconditions:**

- In a Kanuka project with the specified user
- User has one device (first command) or multiple devices (second command)

**Expected Result:**

- Device renamed in project config
- Success message shows old and new name

**Notes:**

Pass

---

## Section 3: Project Initialization

### TEST-030: Initialize project in empty folder

**Command(s):**

```bash
mkdir test-project && cd test-project
../kanuka secrets init
```

**Preconditions:**

- Empty directory
- User config may or may not exist

**Expected Result:**

- If no user config: prompts for email setup
- Prompts for project name (defaults to directory name)
- Creates `.kanuka` directory structure
- Creates RSA keypair
- Creates encrypted symmetric key
- Shows success message with next steps

**Notes:**

Init for the first time works. However, if you cancel the prompt and try init again, it will fail and say that the project is already initialised. The `.kanuka` folder is created too early and is created even if the init command fails or is cancelled early.

---

### TEST-031: Initialize project with --yes flag (non-interactive)

**Command(s):**

```bash
mkdir test-project && cd test-project
../kanuka secrets init --yes
```

**Preconditions:**

- User config exists with email and UUID

**Expected Result:**

- No prompts
- Uses directory name as project name
- Creates all necessary files
- Success message displayed

**Notes:**

Pass

---

### TEST-032: Initialize project with --yes but no user config

**Command(s):**

```bash
rm -rf ~/.config/kanuka
mkdir test-project && cd test-project
../kanuka secrets init --yes
```

**Preconditions:**

- No user config exists

**Expected Result:**

- Error: "User configuration is incomplete"
- Suggestion to run `kanuka config init` first
- No project files created

**Notes:**

Pass

---

### TEST-033: Initialize project with custom name

**Command(s):**

```bash
./kanuka secrets init --name "My Custom Project"
```

**Preconditions:**

- Empty directory, user config exists

**Expected Result:**

- Project created with specified name
- Project name stored in `.kanuka/config.toml`

**Notes:**

Pass

---

### TEST-034: Run init in already initialized project

**Command(s):**

```bash
./kanuka secrets init
```

**Preconditions:**

- Already in an initialized Kanuka project

**Expected Result:**

- Error: "Kanuka has already been initialized"
- Suggestion to run `kanuka secrets create` instead

**Notes:**

Pass

---

### TEST-035: Verbose and debug flags on init

**Command(s):**

```bash
./kanuka secrets init --verbose
./kanuka secrets init --debug
```

**Preconditions:**

- Clean test directory

**Expected Result:**

- `--verbose`: Shows [info] level messages
- `--debug`: Shows [debug] level messages with more detail
- All steps logged appropriately

**Notes:**

Pass

---

## Section 4: Encryption and Decryption

### TEST-040: Encrypt single .env file

**Command(s):**

```bash
echo "SECRET_KEY=abc123" > .env
./kanuka secrets encrypt .env
```

**Preconditions:**

- Initialized Kanuka project
- User has access (symmetric key exists)

**Expected Result:**

- Creates `.env.kanuka` file
- Original `.env` still exists
- Success message lists created file
- Reminder about non-deterministic encryption

**Notes:**

Pass

---

### TEST-041: Encrypt all .env files (no arguments)

**Command(s):**

```bash
echo "KEY1=val1" > .env
echo "KEY2=val2" > .env.local
echo "KEY3=val3" > config/.env.production
./kanuka secrets encrypt
```

**Preconditions:**

- Initialized project with multiple .env files in various locations

**Expected Result:**

- All .env files encrypted
- Corresponding .kanuka files created
- Files in subdirectories also processed

**Notes:**

Pass

---

### TEST-042: Encrypt with glob pattern

**Command(s):**

```bash
./kanuka secrets encrypt "services/*/.env"
```

**Preconditions:**

- Project with .env files in `services/api/`, `services/web/`, etc.

**Expected Result:**

- Only files matching glob pattern are encrypted
- Other .env files untouched

**Notes:**

This ended up encrypting everything:

```bash
aaron@Aarons-MacBook-Pro-2:acceptance_testing % kanuka secrets encrypt "services/*/.env"
✓ Environment files encrypted successfully!
The following files were created:
    - /Users/aaron/Developer/testing/acceptance_testing/.env.kanuka
    - /Users/aaron/Developer/testing/acceptance_testing/.env.local.kanuka
    - /Users/aaron/Developer/testing/acceptance_testing/config/.env.production.kanuka
    - /Users/aaron/Developer/testing/acceptance_testing/services/api/.env.kanuka
    - /Users/aaron/Developer/testing/acceptance_testing/services/web/.env.kanuka
→ You can now safely commit all .kanuka files to version control

Note: Encryption is non-deterministic for security reasons.
       Re-encrypting unchanged files will produce different output.
```

````

---

### TEST-043: Encrypt with --dry-run

**Command(s):**

```bash
./kanuka secrets encrypt --dry-run
````

**Preconditions:**

- Project with unencrypted .env files

**Expected Result:**

- Lists files that WOULD be encrypted
- Shows mapping of .env -> .kanuka
- No files actually created
- Message: "No changes made"

**Notes:**

Pass

---

### TEST-044: Encrypt non-existent file

**Command(s):**

```bash
./kanuka secrets encrypt nonexistent.env
```

**Preconditions:**

- File does not exist

**Expected Result:**

- Clear error message about file not found

**Notes:**

Pass

---

### TEST-045: Encrypt without access (no .kanuka key)

**Command(s):**

```bash
# Remove user's .kanuka file
rm .kanuka/secrets/*.kanuka
./kanuka secrets encrypt
```

**Preconditions:**

- Project exists but user's symmetric key file is missing

**Expected Result:**

- Error about not being able to get .kanuka file
- Suggestion about whether user has access

**Notes:**

Error appears, but shows Go errors instead of just a nicely handled error:

```bash
aaron@Aarons-MacBook-Pro-2:acceptance_testing % kanuka secrets encrypt
✗ Failed to get your .kanuka file. Are you sure you have access?
Error: failed to get user's project encrypted symmetric key: stat /Users/aaron/Developer/testing/acceptance_testing/.kanuka/secrets/beafe009-1cc0-44e3-83e2-2071304c5144.kanuka: no such file or directory
```

---

### TEST-046: Decrypt single .kanuka file

**Command(s):**

```bash
./kanuka secrets decrypt .env.kanuka
```

**Preconditions:**

- Encrypted .kanuka file exists
- User has access

**Expected Result:**

- Creates .env file from .env.kanuka
- Original .kanuka file still exists
- Success message lists created file
- Warning about .gitignore for .env files

**Notes:**

Decrypt command ignores file path, and decrypts everything:

```bash
aaron@Aarons-MacBook-Pro-2:acceptance_testing % kanuka secrets decrypt .env.kanuka
Warning: Decrypted .env files contain sensitive data - ensure they're in your .gitignore
✓ Environment files decrypted successfully!
The following files were created:
    - /Users/aaron/Developer/testing/acceptance_testing/.env
    - /Users/aaron/Developer/testing/acceptance_testing/.env.local
    - /Users/aaron/Developer/testing/acceptance_testing/config/.env.production
    - /Users/aaron/Developer/testing/acceptance_testing/services/api/.env
    - /Users/aaron/Developer/testing/acceptance_testing/services/web/.env
→ Your environment files are now ready to use
```

---

### TEST-047: Decrypt all .kanuka files (no arguments)

**Command(s):**

```bash
./kanuka secrets decrypt
```

**Preconditions:**

- Multiple .kanuka files in project

**Expected Result:**

- All .kanuka files decrypted
- Corresponding .env files created
- Warning about .gitignore

**Notes:**

Pass

---

### TEST-048: Decrypt with --dry-run

**Command(s):**

```bash
./kanuka secrets decrypt --dry-run
```

**Preconditions:**

- .kanuka files exist, some with existing .env files

**Expected Result:**

- Lists files that WOULD be decrypted
- Shows which would be overwritten vs new
- No files actually created/modified
- Message: "No changes made"

**Notes:**

Pass

---

### TEST-049: Decrypt with --private-key-stdin

**Command(s):**

```bash
cat ~/.local/share/kanuka/keys/<project-uuid>/private.pem | ./kanuka secrets decrypt --private-key-stdin
```

**Preconditions:**

- Private key file exists
- Encrypted .kanuka files exist

**Expected Result:**

- Decryption works using piped key
- Same result as normal decryption

**Notes:**

Pass

---

### TEST-050: Decrypt with passphrase-protected key via stdin

**Command(s):**

```bash
cat /path/to/passphrase-protected-key.pem | ./kanuka secrets decrypt --private-key-stdin
```

**Preconditions:**

- Private key is passphrase-protected
- Running in terminal (for TTY passphrase prompt)

**Expected Result:**

- Prompts for passphrase via /dev/tty
- Decryption succeeds after correct passphrase

**Notes:**

Not tested

---

### TEST-051: No .env files to encrypt

**Command(s):**

```bash
./kanuka secrets encrypt
```

**Preconditions:**

- Initialized project with no .env files

**Expected Result:**

- Error: "No environment files found"
- Lists the project path

**Notes:**

Pass

---

### TEST-052: No .kanuka files to decrypt

**Command(s):**

```bash
./kanuka secrets decrypt
```

**Preconditions:**

- Initialized project with no .kanuka files

**Expected Result:**

- Error: "No encrypted environment files found"

**Notes:**

Pass

---

## Section 5: User Registration and Access

### TEST-060: Create keys for new user (secrets create)

**Command(s):**

```bash
./kanuka secrets create --email newuser@example.com
```

**Preconditions:**

- Project already initialized by another user
- Running as a new user (different UUID)

**Expected Result:**

- RSA keypair generated and saved
- Public key copied to `.kanuka/public_keys/<uuid>.pub`
- User added to project config
- Instructions to commit public key and ask for registration

**Notes:**

Pass

---

### TEST-061: Create keys with custom device name

**Command(s):**

```bash
./kanuka secrets create --email user@example.com --device-name work-macbook
```

**Preconditions:**

- Project exists

**Expected Result:**

- Keys created with specified device name
- Device name stored in project config

**Notes:**

Pass. Key was created, and the project config was correct.

NOTE: user config also got updated with the new email (correct behaviour).

---

### TEST-062: Create keys when already have keys

**Command(s):**

```bash
./kanuka secrets create
```

**Preconditions:**

- User already has public key in project

**Expected Result:**

- Error: public key already exists
- Suggestion to use `--force` to override

**Notes:**

Pass

---

### TEST-063: Create keys with --force

**Command(s):**

```bash
./kanuka secrets create --force
```

**Preconditions:**

- User already has keys

**Expected Result:**

- Warning about overwriting existing keys
- New keys generated
- Old .kanuka key file deleted
- Instructions for next steps

**Notes:**

Pass

---

### TEST-064: Register user by email

**Command(s):**

```bash
./kanuka secrets register --user newuser@example.com
```

**Preconditions:**

- Current user has access
- Target user has run `secrets create` (public key exists)
- Target user does NOT yet have .kanuka file

**Expected Result:**

- Creates `<target-uuid>.kanuka` file in secrets directory
- Success message with target user's email
- Lists created files

**Notes:**

Registration passes, but it lists out the path of the public key as "Files created", when that isn't true. It should only list out the `.kanuka` file that got created, rather than that + the public key (which already exists).

---

### TEST-065: Register user with --dry-run

**Command(s):**

```bash
./kanuka secrets register --user newuser@example.com --dry-run
```

**Preconditions:**

- Target user has public key but no access yet

**Expected Result:**

- Shows prerequisites verified
- Lists files that would be created
- No files actually created
- Message: "No changes made"

**Notes:**

Pass

---

### TEST-066: Register user who doesn't exist in project

**Command(s):**

```bash
./kanuka secrets register --user nonexistent@example.com
```

**Preconditions:**

- User has not run `secrets create`

**Expected Result:**

- Error: user not found in project
- Suggestion for them to run `kanuka secrets create`

**Notes:**

Pass

---

### TEST-067: Register with public key file

**Command(s):**

```bash
./kanuka secrets register --file /path/to/alice.pub
```

**Preconditions:**

- Current user has access
- Valid .pub file exists

**Expected Result:**

- Creates .kanuka file for the user
- Uses UUID from filename
- Success message

**Notes:**

Passes, but the file isn't named properly. Here is the log output:

```bash
aaron@Aarons-MacBook-Pro-2:acceptance_testing % kanuka secrets register --file /Users/aaron/.local/share/kanuka/keys/af10d56c-a33b-41ab-9502-db6b6b2a2a29/pubkey.pub
✓ pubkey has been granted access successfully!

Files created:
  Public key:    /Users/aaron/.local/share/kanuka/keys/af10d56c-a33b-41ab-9502-db6b6b2a2a29/pubkey.pub (provided)
  Encrypted key: /Users/aaron/Developer/testing/acceptance_testing/.kanuka/secrets/pubkey.kanuka

→ They now have access to decrypt the repository's secrets
```

Side note, I'm not sure what the correct behaviour for this should be, because what if the path of the pubkey the user provides doesn't have a uuid? What do we do then?

Also, pubkey is not written.

Should we just keep functionality as is, and make it clear to the user that whoever is decrypting secrets with a custom provided pubkey will need to rename their pubkey to have uuid OR remember to always provide that exact pubkey every time?

Also this feature breaks our existing project config stuff because now we have a user with no uuid or device to match. What should we do about this?

---

### TEST-068: Register with public key text

**Command(s):**

```bash
./kanuka secrets register --user alice@example.com --pubkey "ssh-rsa AAAA..."
```

**Preconditions:**

- User exists in project config
- Valid public key text provided

**Expected Result:**

- Saves public key to .pub file
- Creates .kanuka file
- Success message

**Notes:**

Pass

---

### TEST-069: Register without required flags

**Command(s):**

```bash
./kanuka secrets register
```

**Preconditions:**

- None

**Expected Result:**

- Error about missing required flags
- Lists available options (--user, --file, --pubkey)

**Notes:**

Pass

---

### TEST-070: Register when you don't have access

**Command(s):**

```bash
# Remove your own .kanuka file first
rm .kanuka/secrets/<your-uuid>.kanuka
./kanuka secrets register --user alice@example.com
```

**Preconditions:**

- Current user doesn't have access

**Expected Result:**

- Error about not being able to get your Kanuka key
- "Are you sure you have access?"

**Notes:**

Pass, but it returns a Go RSA decryption error, rather than having been handled properly by us.

---

### TEST-071: Re-register user who already has access

**Command(s):**

```bash
./kanuka secrets register --user alice@example.com
```

**Preconditions:**

- User already has full access (.pub and .kanuka files)

**Expected Result:**

- Warning that user already has access
- Prompts for confirmation to continue
- Explains this will replace their existing key

**Notes:**

Pass

---

### TEST-072: Re-register with --force (skip confirmation)

**Command(s):**

```bash
./kanuka secrets register --user alice@example.com --force
```

**Preconditions:**

- User already has access

**Expected Result:**

- No confirmation prompt
- Key replaced
- Success message says "updated" instead of "granted"

**Notes:**

Pass

---

## Section 6: Access Revocation

### TEST-080: Revoke user access

**Command(s):**

```bash
./kanuka secrets revoke --user alice@example.com
```

**Preconditions:**

- User has access to the project
- Current user has access

**Expected Result:**

- If user has multiple devices: prompts for confirmation
- Removes .pub and .kanuka files
- Re-encrypts secrets for remaining users
- Warning about git history access
- Reminder to rotate actual secret values

**Notes:**

Pass

---

### TEST-081: Revoke with --yes (non-interactive)

**Command(s):**

```bash
./kanuka secrets revoke --user alice@example.com --yes
```

**Preconditions:**

- User has multiple devices

**Expected Result:**

- No confirmation prompt
- All devices revoked
- Secrets re-encrypted

**Notes:**

Pass

---

### TEST-082: Revoke specific device

**Command(s):**

```bash
./kanuka secrets revoke --user alice@example.com --device macbook
```

**Preconditions:**

- User has multiple devices

**Expected Result:**

- Only specified device revoked
- Other devices still have access
- Secrets re-encrypted

**Notes:**

Pass

---

### TEST-083: Revoke with --dry-run

**Command(s):**

```bash
./kanuka secrets revoke --user alice@example.com --dry-run
```

**Preconditions:**

- User exists with access

**Expected Result:**

- Shows files that would be deleted
- Shows config changes
- Shows re-encryption impact
- No actual changes made

**Notes:**

Pass

---

### TEST-084: Revoke by file path

**Command(s):**

```bash
./kanuka secrets revoke --file .kanuka/secrets/<uuid>.kanuka
```

**Preconditions:**

- Valid .kanuka file exists

**Expected Result:**

- Removes specified .kanuka file and corresponding .pub
- Removes user from config
- Re-encrypts secrets

**Notes:**

Pass

---

### TEST-085: Revoke non-existent user

**Command(s):**

```bash
./kanuka secrets revoke --user nonexistent@example.com
```

**Preconditions:**

- User not in project

**Expected Result:**

- Error: user not found
- Lists no devices found

**Notes:**

Pass

---

### TEST-086: Revoke without required flags

**Command(s):**

```bash
./kanuka secrets revoke
```

**Preconditions:**

- None

**Expected Result:**

- Error about missing required flags
- Shows available options (--user, --file)

**Notes:**

Pass

---

### TEST-087: Revoke with --device but no --user

**Command(s):**

```bash
./kanuka secrets revoke --device macbook
```

**Preconditions:**

- None

**Expected Result:**

- Error: --device requires --user flag

**Notes:**

Pass with rejection. Wrong error message:

```bash
aaron@Aarons-MacBook-Pro-2:acceptance_testing % kanuka secrets revoke --device device1
✗ Either --user or --file flag is required.
Run kanuka secrets revoke --help to see the available commands.
```

---

## Section 7: Key Rotation and Sync

### TEST-090: Sync secrets (rotate symmetric key)

**Command(s):**

```bash
./kanuka secrets sync
```

**Preconditions:**

- Initialized project with encrypted files
- Current user has access

**Expected Result:**

- Generates new symmetric key
- Re-encrypts all .kanuka secret files
- Re-encrypts symmetric key for all users
- Success message with count of files and users

**Notes:**

Pass

---

### TEST-091: Sync with --dry-run

**Command(s):**

```bash
./kanuka secrets sync --dry-run
```

**Preconditions:**

- Project with encrypted files

**Expected Result:**

- Shows what would happen (files to decrypt, new key, re-encrypt)
- No actual changes made

**Notes:**

Pass

---

### TEST-092: Sync with no encrypted files

**Command(s):**

```bash
./kanuka secrets sync
```

**Preconditions:**

- Project with no .kanuka files

**Expected Result:**

- Message: "No encrypted files found. Nothing to sync."

**Notes:**

Pass

---

### TEST-093: Rotate personal keypair

**Command(s):**

```bash
./kanuka secrets rotate
```

**Preconditions:**

- User has access to project

**Expected Result:**

- Confirmation prompt about replacing keypair
- Generates new RSA keypair
- Re-encrypts symmetric key with new public key
- Updates public key in project
- Saves new private key
- Instruction to commit updated .pub file

**Notes:**

Pass

---

### TEST-094: Rotate with --force (skip confirmation)

**Command(s):**

```bash
./kanuka secrets rotate --force
```

**Preconditions:**

- User has access

**Expected Result:**

- No confirmation prompt
- Keypair rotated successfully

**Notes:**

Pass

---

### TEST-095: Rotate when no access

**Command(s):**

```bash
./kanuka secrets rotate
```

**Preconditions:**

- User does not have .kanuka file

**Expected Result:**

- Error: "You don't have access to this project"
- Suggestion to run `secrets create`

**Notes:**

Pass

---

## Section 8: Status and Diagnostics

### TEST-100: Show access list

**Command(s):**

```bash
./kanuka secrets access
./kanuka secrets access --json
```

**Preconditions:**

- Initialized project with users

**Expected Result:**

- Lists all users with UUID, email, device name
- Shows status: active, pending, or orphan
- Legend explains each status
- Summary shows counts
- JSON is valid and parseable

**Notes:**

Pass

---

### TEST-101: Show encryption status

**Command(s):**

```bash
./kanuka secrets status
./kanuka secrets status --json
```

**Preconditions:**

- Project with mix of encrypted and unencrypted files

**Expected Result:**

- Lists all secret files with status
- Status: current, stale, unencrypted, encrypted_only
- Summary with counts and suggestions
- JSON is valid and parseable

**Notes:**

Pass

---

### TEST-102: Run doctor health checks

**Command(s):**

```bash
./kanuka secrets doctor
./kanuka secrets doctor --json
```

**Preconditions:**

- Initialized project

**Expected Result:**

- Runs all health checks
- Shows pass/warning/error for each
- Summary with counts
- Suggestions for issues found
- Exit code: 0 (pass), 1 (warnings), 2 (errors)

**Notes:**

Pass

---

### TEST-103: Doctor with missing private key

**Command(s):**

```bash
mv ~/.local/share/kanuka/keys/<project-uuid>/private.pem /tmp/
./kanuka secrets doctor
```

**Preconditions:**

- Private key moved away

**Expected Result:**

- Error on "Private key exists" check
- Suggestion to re-run init or register

**Notes:**

Pass

---

### TEST-104: Doctor with insecure private key permissions

**Command(s):**

```bash
chmod 644 ~/.local/share/kanuka/keys/<project-uuid>/private.pem
./kanuka secrets doctor
```

**Preconditions:**

- Private key has wrong permissions

**Expected Result:**

- Warning on "Private key permissions" check
- Suggestion to run chmod 600

**Notes:**

Pass

---

### TEST-105: Doctor with pending users (public key, no .kanuka)

**Command(s):**

```bash
# Create user with public key but don't register
./kanuka secrets doctor
```

**Preconditions:**

- User has .pub file but no .kanuka file

**Expected Result:**

- Warning on "Public key consistency" check
- Suggestion to run `secrets sync`

**Notes:**

Pass

---

### TEST-106: Doctor with orphaned .kanuka files

**Command(s):**

```bash
# Remove public key but leave .kanuka file
rm .kanuka/public_keys/<uuid>.pub
./kanuka secrets doctor
```

**Preconditions:**

- .kanuka file exists without corresponding .pub

**Expected Result:**

- Error on "Encrypted key consistency" check
- Suggestion to run `secrets clean`

**Notes:**

Pass

---

### TEST-107: Doctor with missing .gitignore patterns

**Command(s):**

```bash
rm .gitignore
./kanuka secrets doctor
```

**Preconditions:**

- No .gitignore or no .env patterns in it

**Expected Result:**

- Warning on "Gitignore configuration" check
- Suggestion to add .env patterns

**Notes:**

Pass

---

## Section 9: Cleanup and Maintenance

### TEST-110: Clean orphaned entries

**Command(s):**

```bash
./kanuka secrets clean
```

**Preconditions:**

- Orphaned .kanuka files exist (no corresponding .pub)

**Expected Result:**

- Lists orphaned files with UUID
- Prompts for confirmation
- Removes orphaned files
- Success message with count

**Notes:**

Pass

---

### TEST-111: Clean with --dry-run

**Command(s):**

```bash
./kanuka secrets clean --dry-run
```

**Preconditions:**

- Orphaned entries exist

**Expected Result:**

- Lists what would be removed
- No actual changes made

**Notes:**

Pass

---

### TEST-112: Clean with --force (skip confirmation)

**Command(s):**

```bash
./kanuka secrets clean --force
```

**Preconditions:**

- Orphaned entries exist

**Expected Result:**

- No confirmation prompt
- Files removed
- Success message

**Notes:**

Pass

---

### TEST-113: Clean with no orphans

**Command(s):**

```bash
./kanuka secrets clean
```

**Preconditions:**

- No orphaned entries

**Expected Result:**

- Message: "No orphaned entries found. Nothing to clean."

**Notes:**

Pass

---

## Section 10: Backup and Restore

### TEST-120: Export secrets to archive

**Command(s):**

```bash
./kanuka secrets export
./kanuka secrets export -o backup.tar.gz
```

**Preconditions:**

- Initialized project with encrypted files

**Expected Result:**

- Creates tar.gz archive with date-based name (or specified name)
- Archive contains config.toml, public_keys, secrets, .kanuka files
- Does NOT contain private keys or .env files
- Summary shows file counts

**Notes:**

Pass

---

### TEST-121: Export with verbose output

**Command(s):**

```bash
./kanuka secrets export --verbose
```

**Preconditions:**

- Initialized project

**Expected Result:**

- Shows detailed [info] messages about collection process

**Notes:**

Pass

---

### TEST-122: Import secrets - merge mode

**Command(s):**

```bash
./kanuka secrets import backup.tar.gz --merge
```

**Preconditions:**

- Valid backup archive
- Existing .kanuka directory (or empty)

**Expected Result:**

- Adds new files from archive
- Keeps existing files
- Summary shows added vs skipped counts

**Notes:**

Pass

---

### TEST-123: Import secrets - replace mode

**Command(s):**

```bash
./kanuka secrets import backup.tar.gz --replace
```

**Preconditions:**

- Valid backup archive
- Existing .kanuka directory

**Expected Result:**

- Deletes existing .kanuka directory
- Extracts all files from archive
- Summary shows file count

**Notes:**

Pass

---

### TEST-124: Import with interactive prompt

**Command(s):**

```bash
./kanuka secrets import backup.tar.gz
```

**Preconditions:**

- Existing .kanuka directory
- No --merge or --replace flag

**Expected Result:**

- Prompts: "Found existing .kanuka directory. How do you want to proceed?"
- Options: [m] Merge, [r] Replace, [c] Cancel

**Notes:**

Pass

---

### TEST-125: Import with --dry-run

**Command(s):**

```bash
./kanuka secrets import backup.tar.gz --dry-run
```

**Preconditions:**

- Valid archive

**Expected Result:**

- Shows what would be imported
- No actual changes made

**Notes:**

Pass

---

### TEST-126: Import invalid archive

**Command(s):**

```bash
echo "not a tar" > fake.tar.gz
./kanuka secrets import fake.tar.gz
```

**Preconditions:**

- Invalid archive file

**Expected Result:**

- Error: "failed to read archive" or similar
- No changes made

**Notes:**

Pass, but not a nice error message:

```bash
aaron@Aarons-MacBook-Pro-2:acceptance_testing % kanuka secrets import fake.tar.gz
❌ failed to read archive: failed to create gzip reader: gzip: invalid header
Error: failed to read archive: failed to create gzip reader: gzip: invalid header
Usage:
  kanuka secrets import <archive> [flags]

Flags:
      --dry-run   show what would be imported without making changes
  -h, --help      help for import
      --merge     merge with existing files (add new, keep existing)
      --replace   replace existing .kanuka directory with backup

Global Flags:
  -d, --debug     enable debug output
  -v, --verbose   enable verbose output

failed to read archive: failed to create gzip reader: gzip: invalid header
```

I'd prefer it if we had a nicer error message that we define instead of a Go error.

---

### TEST-127: Import archive missing config.toml

**Command(s):**

```bash
./kanuka secrets import incomplete-backup.tar.gz
```

**Preconditions:**

- Archive without .kanuka/config.toml

**Expected Result:**

- Error: "archive missing .kanuka/config.toml"

**Notes:**

Fail. The import succeeded, and created a blank config file.

---

### TEST-128: Import with both --merge and --replace

**Command(s):**

```bash
./kanuka secrets import backup.tar.gz --merge --replace
```

**Preconditions:**

- Valid archive

**Expected Result:**

- Error: "cannot use both --merge and --replace flags"

**Notes:**

Pass but not nice error message:

```bash
aaron@Aarons-MacBook-Pro-2:acceptance_testing % kanuka secrets import kanuka-secrets-2026-01-13.tar.gz --merge --replace
❌ cannot use both --merge and --replace flags
Error: cannot use both --merge and --replace flags
Usage:
  kanuka secrets import <archive> [flags]

Flags:
      --dry-run   show what would be imported without making changes
  -h, --help      help for import
      --merge     merge with existing files (add new, keep existing)
      --replace   replace existing .kanuka directory with backup

Global Flags:
  -d, --debug     enable debug output
  -v, --verbose   enable verbose output

cannot use both --merge and --replace flags
```

We should not use Go's error messages, but instead we should craft our own.

---

## Section 11: Audit Log

### TEST-130: View audit log

**Command(s):**

```bash
./kanuka secrets log
```

**Preconditions:**

- Some operations have been performed

**Expected Result:**

- Lists operations with timestamp, user, operation, details
- Formatted table output

**Notes:**

Pass

---

### TEST-131: View log with limit

**Command(s):**

```bash
./kanuka secrets log -n 5
./kanuka secrets log --reverse -n 5
```

**Preconditions:**

- More than 5 log entries

**Expected Result:**

- Shows only 5 entries
- Without --reverse: oldest 5
- With --reverse: newest 5 first

**Notes:**

Pass

---

### TEST-132: Filter log by user

**Command(s):**

```bash
./kanuka secrets log --user alice@example.com
```

**Preconditions:**

- Entries from multiple users

**Expected Result:**

- Only shows entries from specified user

**Notes:**

Pass

---

### TEST-133: Filter log by operation

**Command(s):**

```bash
./kanuka secrets log --operation encrypt,decrypt
```

**Preconditions:**

- Various operation types in log

**Expected Result:**

- Only shows encrypt and decrypt operations

**Notes:**

Pass

---

### TEST-134: Filter log by date

**Command(s):**

```bash
./kanuka secrets log --since 2024-01-01
./kanuka secrets log --until 2024-06-30
./kanuka secrets log --since 2024-01-01 --until 2024-06-30
```

**Preconditions:**

- Entries spanning date range

**Expected Result:**

- Shows only entries within date range

**Notes:**

Pass

---

### TEST-135: Log in oneline format

**Command(s):**

```bash
./kanuka secrets log --oneline
```

**Preconditions:**

- Log entries exist

**Expected Result:**

- Compact one-line format
- Date, user, operation, brief details

**Notes:**

Fail. I don't think this exported as one line.

```bash
aaron@Aarons-MacBook-Pro-2:acceptance_testing % kanuka secrets log --oneline
2026-01-13 aaron@guo.nz init acceptance_testing_2
2026-01-13 aaron2@guo.nz create aarons-macbook-pro-2local
2026-01-13 aaron@guo.nz register aaron2@guo.nz
2026-01-13 aaron@guo.nz register pubkey
2026-01-13 aaron@guo.nz register aaron2@guo.nz
2026-01-13 aaron@guo.nz encrypt 5 files
2026-01-13 aaron@guo.nz export kanuka-secrets-2026-01-13.tar.gz
2026-01-13 aaron@guo.nz export backup.tar.gz
2026-01-13 aaron@guo.nz export kanuka-secrets-2026-01-13.tar.gz
```

---

### TEST-136: Log in JSON format

**Command(s):**

```bash
./kanuka secrets log --json
```

**Preconditions:**

- Log entries exist

**Expected Result:**

- Valid JSON array output
- All entry fields included

**Notes:**

Pass

---

### TEST-137: Log with no entries

**Command(s):**

```bash
./kanuka secrets log
```

**Preconditions:**

- Fresh install, no operations performed

**Expected Result:**

- Message: "No audit log found. Operations will be logged after running any secrets command."

**Notes:**

Pass

---

## Section 12: CI/CD and Automation Scenarios (NOT TESTED)

### TEST-140: Full CI/CD decrypt flow

**Command(s):**

```bash
# Simulate CI environment
cat private_key.pem | ./kanuka secrets decrypt --private-key-stdin
```

**Preconditions:**

- Project with encrypted files
- Private key available as file or environment variable

**Expected Result:**

- Decrypts all files non-interactively
- No prompts
- Exit code 0 on success

**Notes:**

---

### TEST-141: CI/CD encrypt flow

**Command(s):**

```bash
cat private_key.pem | ./kanuka secrets encrypt --private-key-stdin
```

**Preconditions:**

- .env files exist
- Private key available

**Expected Result:**

- Encrypts all files non-interactively
- No prompts
- Exit code 0 on success

**Notes:**

---

### TEST-142: Register in CI with stdin key

**Command(s):**

```bash
cat private_key.pem | ./kanuka secrets register --user newuser@example.com --private-key-stdin
```

**Preconditions:**

- User has created keys
- Admin private key available

**Expected Result:**

- Registers user non-interactively

**Notes:**

---

### TEST-143: Revoke in CI with stdin key

**Command(s):**

```bash
cat private_key.pem | ./kanuka secrets revoke --user olduser@example.com --yes --private-key-stdin
```

**Preconditions:**

- User exists
- Admin private key available

**Expected Result:**

- Revokes access non-interactively
- Secrets re-encrypted

**Notes:**

---

## Section 13: Edge Cases and Error Handling

### TEST-150: Invalid email format in various commands

**Command(s):**

```bash
./kanuka secrets register --user "not-an-email"
./kanuka secrets revoke --user "bad@"
./kanuka secrets create --email "@incomplete.com"
```

**Preconditions:**

- None

**Expected Result:**

- All commands reject invalid emails
- Clear error message about format

**Notes:**

Pass. However, not sure if we should continue to call the flag --user or if we should be consistent and have it be --email fully.

---

### TEST-151: Operations on corrupted .kanuka file

**Command(s):**

```bash
echo "garbage" > .kanuka/secrets/<uuid>.kanuka
./kanuka secrets decrypt
```

**Preconditions:**

- Corrupted .kanuka file

**Expected Result:**

- Clear error about decryption failure
- Does not crash

**Notes:**

Not sure if we should show the user the Go error, but pass otherwise.

```bash
aaron@Aarons-MacBook-Pro-2:acceptance_testing % kanuka secrets decrypt
✗ Failed to decrypt your .kanuka file. Are you sure you have access?
Error: crypto/rsa: decryption error
```

---

### TEST-152: Operations with corrupted config.toml

**Command(s):**

```bash
echo "not valid toml [" > .kanuka/config.toml
./kanuka secrets status
```

**Preconditions:**

- Invalid TOML in config file

**Expected Result:**

- Error about parsing config
- Suggestion to check for syntax errors

**Notes:**

We get a Go error instead of a nice user friendly error:

```bash
aaron@Aarons-MacBook-Pro-2:acceptance_testing % kanuka secrets status
❌ failed to load project config: failed to load project config: toml: line 2: expected '.' or ']' to end table name, but got '\n' instead
Error: failed to load project config: failed to load project config: toml: line 2: expected '.' or ']' to end table name, but got '\n' instead
Usage:
  kanuka secrets status [flags]

Flags:
  -h, --help   help for status
      --json   output in JSON format

Global Flags:
  -d, --debug     enable debug output
  -v, --verbose   enable verbose output

failed to load project config: failed to load project config: toml: line 2: expected '.' or ']' to end table name, but got '\n' instead
```

---

### TEST-153: Spaces and special characters in paths

**Command(s):**

```bash
mkdir "Project With Spaces"
cd "Project With Spaces"
../kanuka secrets init
echo "KEY=val" > ".env with spaces"
../kanuka secrets encrypt
```

**Preconditions:**

- Directory and files with spaces

**Expected Result:**

- All operations work correctly
- Paths handled properly

**Notes:**

Pass

---

### TEST-154: Very deep directory nesting

**Command(s):**

```bash
mkdir -p a/b/c/d/e/f/g/h/i/j
echo "KEY=val" > a/b/c/d/e/f/g/h/i/j/.env
./kanuka secrets encrypt
./kanuka secrets status
```

**Preconditions:**

- Deeply nested .env file

**Expected Result:**

- File discovered and encrypted
- Status shows correct relative path

**Notes:**

Pass

---

### TEST-155: Large number of files

**Command(s):**

```bash
for i in {1..50}; do echo "KEY=$i" > ".env.$i"; done
./kanuka secrets encrypt
./kanuka secrets status
```

**Preconditions:**

- 50+ .env files

**Expected Result:**

- All files encrypted (may show performance warning)
- Status lists all files

**Notes:**

Pass

---

### TEST-156: Unicode in .env file content

**Command(s):**

```bash
echo "GREETING=Hello World" > .env
echo "EMOJI=" >> .env
./kanuka secrets encrypt
./kanuka secrets decrypt
cat .env
```

**Preconditions:**

- .env with unicode characters

**Expected Result:**

- Content preserved after encrypt/decrypt cycle

**Notes:**

Pass

---

### TEST-157: Binary content in file

**Command(s):**

```bash
dd if=/dev/urandom of=.env bs=1024 count=1
./kanuka secrets encrypt
./kanuka secrets decrypt
```

**Preconditions:**

- .env with binary content

**Expected Result:**

- Either handles gracefully or gives clear error
- Does not crash

**Notes:**

Pass

---

### TEST-158: Concurrent operations (race conditions)

**Command(s):**

```bash
# Run in parallel:
./kanuka secrets encrypt &
./kanuka secrets encrypt &
wait
```

**Preconditions:**

- Multiple .env files

**Expected Result:**

- Either succeeds or fails cleanly
- No corrupted files
- No deadlocks

**Notes:**

Pass, both worked.

---

### TEST-159: Disk full scenario (NOT TESTED)

**Command(s):**

```bash
# Simulate full disk (platform-specific)
./kanuka secrets encrypt
```

**Preconditions:**

- Very limited disk space

**Expected Result:**

- Clear error about disk space
- No partial/corrupted files

**Notes:**

---

### TEST-160: Read-only filesystem

**Command(s):**

```bash
chmod 555 .kanuka
./kanuka secrets encrypt
```

**Preconditions:**

- .kanuka directory is read-only

**Expected Result:**

- Clear permission error
- Suggestion to check permissions

**Notes:**

FAIL.

```bash
aaron@Aarons-MacBook-Pro-2:acceptance_testing % chmod 555 .kanuka

aaron@Aarons-MacBook-Pro-2:acceptance_testing % kanuka secrets encrypt
✓ Environment files encrypted successfully!
The following files were created:
    - /Users/aaron/Developer/testing/acceptance_testing/.env.kanuka
    - /Users/aaron/Developer/testing/acceptance_testing/.env.local.kanuka
    - /Users/aaron/Developer/testing/acceptance_testing/config/.env.production.kanuka
    - /Users/aaron/Developer/testing/acceptance_testing/services/api/.env.kanuka
    - /Users/aaron/Developer/testing/acceptance_testing/services/web/.env.kanuka
→ You can now safely commit all .kanuka files to version control

Note: Encryption is non-deterministic for security reasons.
       Re-encrypting unchanged files will produce different output.
```

---

## Section 14: Monorepo Scenarios

### TEST-170: Single .kanuka at repo root

**Command(s):**

```bash
./kanuka secrets init
echo "API_KEY=xxx" > services/api/.env
echo "WEB_KEY=yyy" > services/web/.env
./kanuka secrets encrypt
./kanuka secrets status
```

**Preconditions:**

- Monorepo structure with services subdirectories

**Expected Result:**

- All .env files encrypted from root
- Status shows all files with relative paths

**Notes:**

Pass

---

### TEST-171: Selective encryption in monorepo

**Command(s):**

```bash
./kanuka secrets encrypt services/api/.env
./kanuka secrets encrypt "services/*/.env"
```

**Preconditions:**

- Monorepo with multiple service .env files

**Expected Result:**

- Only specified files encrypted
- Other files untouched

**Notes:**

Pass

---

### TEST-172: Multiple .kanuka stores (per-service)

**Command(s):**

```bash
cd services/api && ../../kanuka secrets init
cd services/web && ../../kanuka secrets init
```

**Preconditions:**

- Monorepo structure

**Expected Result:**

- Each service has its own .kanuka directory
- Independent user lists and keys

**Notes:**

Pass

---

## Section 15: UX and Output Quality

### TEST-180: Color and formatting in terminal

**Command(s):**
Run various commands in a terminal with color support

**Expected Result:**

- Success messages in green
- Errors in red
- Warnings in yellow
- Code snippets highlighted
- Paths styled distinctly

**Notes:**

Pass

---

### TEST-181: Output in non-TTY (pipe)

**Command(s):**

```bash
./kanuka secrets status | cat
./kanuka secrets access | head
```

**Preconditions:**

- Piped output

**Expected Result:**

- Output readable without color codes
- No garbled escape sequences

**Notes:**

Pass

---

### TEST-182: Spinner behavior during long operations

**Command(s):**

```bash
./kanuka secrets encrypt  # with many files
./kanuka secrets sync     # with many users
```

**Preconditions:**

- Operation that takes a few seconds

**Expected Result:**

- Spinner shows activity
- Final message replaces spinner cleanly

**Notes:**

Pass

---

### TEST-183: Error message clarity

Review all error messages encountered during testing.

**Expected Result:**

- All errors explain WHAT went wrong
- Most errors suggest HOW to fix
- No cryptic error codes alone

**Notes:**

For the things that worked, pass.

---

### TEST-184: Help text completeness

**Command(s):**

```bash
./kanuka --help
./kanuka secrets --help
./kanuka secrets encrypt --help
./kanuka config --help
# ... for all commands
```

**Expected Result:**

- All commands have short and long descriptions
- Examples provided for complex commands
- All flags documented

**Notes:**

Pass

---

## Cleanup

After testing, clean up test artifacts:

```bash
# Remove test directories
rm -rf ~/kanuka-test

# Optionally restore original user config
# (if you backed it up before testing)
```

---

## Summary and Notes

Use this section to record overall findings:

### Bugs Found

| ID  | Severity | Description | Command | Status |
| --- | -------- | ----------- | ------- | ------ |
|     |          |             |         |        |

### UX Improvements

| ID  | Priority | Description | Current Behavior | Suggested Improvement |
| --- | -------- | ----------- | ---------------- | --------------------- |
|     |          |             |                  |                       |

### Documentation Gaps

| ID  | Description | Location |
| --- | ----------- | -------- |
|     |             |          |

### Tester Information

- **Tester:** Aaron Guo
- **Date:** 2026-01-13
- **Platform:** macOS
- **Kanuka Version:** 1.2.1
- **Go Version:** 1.24.5 darwin/arm64
