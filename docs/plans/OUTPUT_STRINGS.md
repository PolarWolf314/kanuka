# Output Strings Catalog

This document catalogs every user-facing output string in the Kanuka CLI, organized by file and categorized by the mechanism used to display it.

## Output Mechanisms

### 1. spinner.FinalMSG
Messages assigned to `spinner.FinalMSG` are printed via the cleanup function in `cmd/secrets_helper_methods.go`. The cleanup function calls `ui.EnsureNewline()` before printing, which ensures proper newline handling.

### 2. Direct Prints (fmt.Println, fmt.Printf, fmt.Print)
Direct print statements are used for:
- Multi-line instructions (like CI init next steps)
- Dry-run output
- Interactive prompts
- Confirmation dialogs
- Table output

### 3. utils.WriteToTTY
Used for sensitive output that must be displayed directly to the terminal (bypassing stdout/stderr).

### 4. fmt.Fprint/fmt.Fprintln to os.Stderr
Used for passphrase prompts and warning messages during key loading.

---

## cmd/secrets_init.go

### spinner.FinalMSG
| Line | Context | String Pattern |
|------|---------|----------------|
| 52 | Already initialized error | `formatInitError(kerrors.ErrProjectAlreadyInitialized)` |
| 72-74 | User config incomplete (--yes mode) | `ui.Error.Sprint("✗") + " User configuration is incomplete" + "\n" + ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka config init") + " first..."` |
| 108 | Workflow error | `formatInitError(err)` |
| 124-130 | Success message | `ui.Success.Sprint("✓") + " Kanuka initialized successfully!" + "\n\n" + ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets encrypt")...` |

### Direct Prints (fmt.Println/Printf/Print)
| Line | Context | String Pattern |
|------|---------|----------------|
| 78-79 | User config not found prompt | `ui.Warning.Sprint("⚠") + " User configuration not found.\n"` + `"Running initial setup..."` |
| 91 | Initializing project | `"Initializing project..."` |
| 161 | Project name prompt | `fmt.Printf("Project name [%s]: ", defaultProjectName)` |

### formatInitError() Messages (returns spinner.FinalMSG strings)
| Error | Message |
|-------|---------|
| ErrProjectAlreadyInitialized | `✗ Kanuka has already been initialized\n→ Run kanuka secrets create instead` |
| default | `✗ {err.Error()}` |

---

## cmd/secrets_create.go

### spinner.FinalMSG
| Line | Context | String Pattern |
|------|---------|----------------|
| 84, 125 | Error formatting | `formatCreateError(err, email)` |
| 139-144 | Success message | `ui.Success.Sprint("✓") + " Keys created for " + ui.Highlight.Sprint(result.Email) + " (device: " + ui.Highlight.Sprint(result.DeviceName) + ")"...` |

### Direct Prints (fmt.Println/Printf/Print)
| Line | Context | String Pattern |
|------|---------|----------------|
| 39 | Email prompt | `fmt.Print("Enter your email: ")` |

### formatCreateError() Messages
| Error | Message |
|-------|---------|
| ErrProjectNotInitialized | `✗ Kanuka has not been initialized\n→ Run kanuka secrets init first to create a project` |
| ErrInvalidProjectConfig | `✗ Failed to load project config: .kanuka/config.toml is not valid TOML\n\nTo fix this issue:...` |
| ErrInvalidEmail | `✗ Invalid email format: {email}\n→ Please provide a valid email address` |
| ErrDeviceNameTaken | `✗ Device name {deviceName} is already in use for {email}\n→ Choose a different device name with --device-name` |
| ErrPublicKeyExists | `✗ Public key already exists\nTo override, run: kanuka secrets create --force` |
| default | `✗ Failed to create keys\nError: {err.Error()}` |

---

## cmd/secrets_register.go

### spinner.FinalMSG
| Line | Context | String Pattern |
|------|---------|----------------|
| 97-99 | Missing required flags | `ui.Error.Sprint("✗") + " Either " + ui.Flag.Sprint("--user") + ", " + ui.Flag.Sprint("--file") + ", or " + ui.Flag.Sprint("--pubkey") + " must be specified."...` |
| 105-106 | Missing user with pubkey | `ui.Error.Sprint("✗") + " When using " + ui.Flag.Sprint("--pubkey") + ", the " + ui.Flag.Sprint("--user") + " flag is required."...` |
| 113-114 | Invalid email format | `ui.Error.Sprint("✗") + " Invalid email format: " + ui.Highlight.Sprint(registerUserEmail)...` |
| 121-122 | Empty pubkey | `ui.Error.Sprint("✗") + " Invalid public key format provided\n" + ui.Error.Sprint("Error: ") + "public key text cannot be empty"` |
| 133-134 | Failed to read stdin | `ui.Error.Sprint("✗") + " Failed to read private key from stdin\n" + ui.Error.Sprint("Error: ") + err.Error()` |
| 159 | Cancelled | `ui.Warning.Sprint("⚠") + " Registration cancelled."` |
| 179 | Error formatting | `formatRegisterError(err, registerUserEmail, customFilePath)` |
| 201 | Success formatting | `formatRegisterSuccess(result)` |

### Direct Prints (fmt.Println/Printf/Print)
| Line | Context | String Pattern |
|------|---------|----------------|
| 308-328 | Dry-run output | `fmt.Println(ui.Warning.Sprint("[dry-run]") + " Would register " + ui.Highlight.Sprint(result.DisplayName))...` |
| 335-338 | Overwrite confirmation | `fmt.Printf("\n%s Warning: %s already has access to this project.\n", ui.Warning.Sprint("⚠"), ui.Highlight.Sprint(userEmail))...` |
| 342 | Confirm prompt | `fmt.Print("Do you want to continue? [y/N]: ")` |

---

## cmd/secrets_encrypt.go

### spinner.FinalMSG
| Line | Context | String Pattern |
|------|---------|----------------|
| 86 | Failed to read stdin | `ui.Error.Sprint("✗") + " Failed to read private key from stdin: " + err.Error()` |
| 95 | Error formatting | `formatEncryptError(err, encryptPrivateKeyStdin)` |
| 107-111 | Success message | `ui.Success.Sprint("✓") + " Environment files encrypted successfully!"...` |
| 180 | Dry-run reset | `""` (empty, output is via direct prints) |

### Direct Prints (fmt.Println/Printf/Print)
| Line | Context | String Pattern |
|------|---------|----------------|
| 163-178 | Dry-run output | `fmt.Println(ui.Warning.Sprint("[dry-run]") + fmt.Sprintf(" Would encrypt %d environment file(s)", len(envFiles)))...` |

---

## cmd/secrets_decrypt.go

### spinner.FinalMSG
| Line | Context | String Pattern |
|------|---------|----------------|
| 86 | Failed to read stdin | `ui.Error.Sprint("✗") + " Failed to read private key from stdin: " + err.Error()` |
| 95 | Error formatting | `formatDecryptError(err, decryptPrivateKeyStdin)` |
| 111-113 | Success message | `ui.Success.Sprint("✓") + " Environment files decrypted successfully!"...` |
| 200 | Dry-run reset | `""` (empty, output is via direct prints) |

### Direct Prints (fmt.Println/Printf/Print)
| Line | Context | String Pattern |
|------|---------|----------------|
| 167-198 | Dry-run output | `fmt.Println(ui.Warning.Sprint("[dry-run]") + fmt.Sprintf(" Would decrypt %d encrypted file(s)", len(kanukaFiles)))...` |

---

## cmd/secrets_revoke.go

### spinner.FinalMSG
| Line | Context | String Pattern |
|------|---------|----------------|
| 113-114 | Device without user flag | `ui.Error.Sprint("✗") + " The " + ui.Flag.Sprint("--device") + " flag requires " + ui.Flag.Sprint("--user") + " flag."...` |
| 120-121 | Missing required flags | `ui.Error.Sprint("✗") + " Either " + ui.Flag.Sprint("--user") + " or " + ui.Flag.Sprint("--file") + " flag is required."...` |
| 127-128 | Conflicting flags | `ui.Error.Sprint("✗") + " Cannot specify both " + ui.Flag.Sprint("--user") + " and " + ui.Flag.Sprint("--file") + " flags."...` |
| 135-136 | Invalid email | `ui.Error.Sprint("✗") + " Invalid email format: " + ui.Highlight.Sprint(revokeUserEmail)...` |
| 172 | Cancelled | `ui.Warning.Sprint("⚠") + " Revocation cancelled."` |
| 194 | Error formatting | `formatRevokeError(err)` |
| 208-209 | Self-revoke warning | `formatRevokeSuccess(result) + "\n" + ui.Warning.Sprint("⚠") + " Note: You revoked your own access to this project"` |
| 220 | Success formatting | `formatRevokeSuccess(result)` |

### Direct Prints (fmt.Println/Printf/Print)
| Line | Context | String Pattern |
|------|---------|----------------|
| 157-162 | Multi-device warning | `fmt.Printf("\n%s Warning: %s has %d devices:\n", ui.Warning.Sprint("⚠"), revokeUserEmail, len(devices))...` |
| 165 | Confirm prompt | `fmt.Print("Proceed? [y/N]: ")` |
| 288-323 | Dry-run output | `fmt.Println(ui.Warning.Sprint("[dry-run]") + " Would revoke access for " + ui.Highlight.Sprint(result.DisplayName))...` |

---

## cmd/secrets_rotate.go

### spinner.FinalMSG
| Line | Context | String Pattern |
|------|---------|----------------|
| 89 | Cancelled | `ui.Warning.Sprint("⚠") + " Keypair rotation cancelled."` |
| 100 | Error formatting | `formatRotateError(err)` |
| 107-110 | Success message | `ui.Success.Sprint("✓") + " Keypair rotated successfully\n\n"...` |

### Direct Prints (fmt.Println/Printf/Print)
| Line | Context | String Pattern |
|------|---------|----------------|
| 37-39 | Confirmation prompt | `fmt.Printf("\n%s This will generate a new keypair and replace your current one.\n", ui.Warning.Sprint("Warning:"))...` |
| 42 | Confirm prompt | `fmt.Print("Do you want to continue? [y/N]: ")` |

---

## cmd/secrets_sync.go

### spinner.FinalMSG
| Line | Context | String Pattern |
|------|---------|----------------|
| 50 | Error formatting | `formatSyncError(err)` |
| 61 | Dry-run reset | `""` |
| 67 | No files found | `ui.Success.Sprint("✓") + " No encrypted files found. Nothing to sync."` |
| 71-73 | Success message | `ui.Success.Sprint("✓") + " Secrets synced successfully"...` |

### Direct Prints (fmt.Println/Printf/Print)
| Line | Context | String Pattern |
|------|---------|----------------|
| 117-139 | Dry-run output | `fmt.Println(ui.Warning.Sprint("[dry-run]") + " Would sync secrets:")...` |

---

## cmd/secrets_ci_init.go

### spinner.FinalMSG
| Line | Context | String Pattern |
|------|---------|----------------|
| 61 | Error formatting | `formatCIInitError(err)` |
| 74 | Reset before manual output | `""` |

### Direct Prints (fmt.Println/Printf/Print)
| Line | Context | String Pattern |
|------|---------|----------------|
| 79 | Failed to display key | `fmt.Println(ui.Error.Sprint("✗") + " Failed to display private key: " + err.Error())` |
| 150-180 | Success and next steps | `fmt.Println(ui.Success.Sprint("✓") + " CI user registered successfully!")...` |

### utils.WriteToTTY (Secure TTY Output)
| Line | Context | String Pattern |
|------|---------|----------------|
| 113-116 | Pre-key instructions | `ui.Warning.Sprint("IMPORTANT:") + " Copy the private key below and save it to GitHub Secrets.\n"...` |
| 123 | Private key PEM | `string(result.PrivateKeyPEM)` |
| 128-129 | Post-key prompt | `"Press " + ui.Highlight.Sprint("Enter") + " when you have copied the key..."` |

---

## cmd/secrets_status.go

### spinner.FinalMSG
| Line | Context | String Pattern |
|------|---------|----------------|
| 72 | Error formatting | `formatStatusError(err)` |
| 82 | Failed JSON output | `ui.Error.Sprint("✗") + " Failed to output status."` |
| 87 | Success | `ui.Success.Sprint("✓") + " Status displayed."` |

### Direct Prints (fmt.Println/Printf/Print)
| Line | Context | String Pattern |
|------|---------|----------------|
| 69 | JSON error output | `fmt.Printf('{"error": "%s"}'+"\n", formatStatusErrorJSON(err))` |
| 172-237 | Status table | `printStatusTable(result)` - extensive table output |

---

## cmd/secrets_log.go

### spinner.FinalMSG
| Line | Context | String Pattern |
|------|---------|----------------|
| 88 | Error formatting | `formatLogError(err)` |
| 100, 103, 110 | Reset for direct output | `""` |

### Direct Prints (fmt.Println/Printf/Print)
| Line | Context | String Pattern |
|------|---------|----------------|
| 101 | No entries | `fmt.Println("No audit log entries found.")` |
| 104 | No matching entries | `fmt.Println("No audit log entries found matching the filters.")` |
| 162 | JSON output | `fmt.Println(string(data))` |
| 170 | Oneline format | `fmt.Printf("%s %s %s %s\n", date, e.User, e.Operation, details)` |
| 178 | Default format | `fmt.Printf("%-19s  %-25s  %-10s  %s\n", datetime, e.User, e.Operation, details)` |

---

## cmd/secrets_import.go

### spinner.FinalMSG
| Line | Context | String Pattern |
|------|---------|----------------|
| 87 | Error formatting | `formatImportError(err, archivePath)` |
| 129 | Error formatting | `formatImportError(err, archivePath)` |
| 136-158 | Success/dry-run message | Complex multi-line message with mode, file counts, and notes |

### Direct Prints (fmt.Println/Printf/Print)
| Line | Context | String Pattern |
|------|---------|----------------|
| 74-78 | Conflicting flags error | `fmt.Print(finalMessage)` - `ui.Error.Sprint("✗") + " Cannot use both --merge and --replace flags."...` |
| 106 | Cancelled | `fmt.Println(ui.Warning.Sprint("⚠") + " Import cancelled")` |
| 205-208 | Import mode prompt | `fmt.Println("Found existing .kanuka directory. How do you want to proceed?")...` |
| 209 | Choice prompt | `fmt.Print("Choice: ")` |

---

## cmd/secrets_export.go

### spinner.FinalMSG
| Line | Context | String Pattern |
|------|---------|----------------|
| 67 | Error formatting | `formatExportError(err)` |
| 75 | Success formatting | `formatExportSuccess(result)` |

---

## cmd/secrets_doctor.go

### spinner.FinalMSG
| Line | Context | String Pattern |
|------|---------|----------------|
| 66 | Error | `ui.Error.Sprint("✗") + " Failed to run health checks: " + err.Error()` |
| 76, 81 | Reset for direct output | `""` |

### Direct Prints (fmt.Println/Printf/Print)
| Line | Context | String Pattern |
|------|---------|----------------|
| 84 | Errors summary | `fmt.Println(ui.Error.Sprint("✗") + " Health checks completed with errors")` |
| 86 | Warnings summary | `fmt.Println(ui.Warning.Sprint("⚠") + " Health checks completed with warnings")` |
| 88 | Success | `fmt.Println(ui.Success.Sprint("✓") + " Health checks completed")` |
| 110-146 | Health check results | `printDoctorResults(result)` - extensive table/summary output |

---

## cmd/secrets_clean.go

### spinner.FinalMSG
| Line | Context | String Pattern |
|------|---------|----------------|
| 61 | Error formatting | `formatCleanError(err)` |
| 69 | No orphans | `ui.Success.Sprint("✓") + " No orphaned entries found. Nothing to clean."` |
| 86 | Dry-run reset | `""` |
| 98 | Abort reset | `""` |
| 113 | Error formatting | `formatCleanError(err)` |
| 117 | Success | `ui.Success.Sprint("✓") + fmt.Sprintf(" Removed %d orphaned file(s)", result.RemovedCount)` |

### Direct Prints (fmt.Println/Printf/Print)
| Line | Context | String Pattern |
|------|---------|----------------|
| 76 | Dry-run header | `fmt.Printf("[dry-run] Would remove %d orphaned file(s):\n", len(previewResult.Orphans))` |
| 78 | Found orphans | `fmt.Printf("Found %d orphaned entry(ies):\n\n", len(previewResult.Orphans))` |
| 85 | Dry-run footer | `fmt.Println("\nNo changes made.")` |
| 92-94 | Confirmation text | `fmt.Println("This will permanently delete the orphaned files listed above.")...` |
| 97 | Abort | `fmt.Println("Aborted.")` |
| 154-158 | Orphan table | `printOrphanTable(orphans)` |
| 164 | Confirm prompt | `fmt.Print("Do you want to continue? [y/N]: ")` |

---

## cmd/secrets_access.go

### spinner.FinalMSG
| Line | Context | String Pattern |
|------|---------|----------------|
| 71 | Error formatting | `formatAccessError(err)` |
| 81 | Failed JSON output | `ui.Error.Sprint("✗") + " Failed to output access information."` |
| 88 | Success | `ui.Success.Sprint("✓") + " Access information displayed."` |

### Direct Prints (fmt.Println/Printf/Print)
| Line | Context | String Pattern |
|------|---------|----------------|
| 68 | JSON error output | `fmt.Printf('{"error": "%s"}'+"\n", formatAccessErrorJSON(err))` |
| 170-251 | Access table | `printAccessTable(result)` - extensive table output with legend |

---

## cmd/config_init.go

### spinner.FinalMSG
None (does not use spinner pattern)

### Direct Prints (fmt.Println/Printf/Print)
| Line | Context | String Pattern |
|------|---------|----------------|
| 41-44 | Input prompts | `fmt.Printf("%s [%s]: ", prompt, defaultValue)` or `fmt.Printf("%s: ", prompt)` |
| 78 | Welcome message | `fmt.Println(ui.Info.Sprint("Welcome to Kanuka!") + " Let's set up your identity.\n")` |
| 161-171 | Success summary | `fmt.Println(ui.Success.Sprint("✓") + " User configuration saved to " + ui.Path.Sprint(...))...` |
| 235-246 | Already configured output | `fmt.Println(ui.Success.Sprint("✓") + " User configuration already exists\n")...` |
| 262, 277 | Validation errors | `fmt.Println(ui.Error.Sprint("✗") + " Invalid email format: " + ui.Highlight.Sprint(configInitEmail))` |
| 302-311 | Updated output | `fmt.Println(ui.Success.Sprint("✓") + " User configuration updated\n")...` |
| 319 | Error output | `fmt.Println(ui.Error.Sprint("✗") + " " + err.Error())` |

---

## cmd/config_show.go

### spinner.FinalMSG
| Line | Context | String Pattern |
|------|---------|----------------|
| 70 | Error initializing | `ui.Error.Sprint("✗") + " Failed to initialize user settings\n"` |
| 77 | Error loading | `ui.Error.Sprint("✗") + " Failed to load user configuration\n"` |
| 88 | Warning no config | `ui.Warning.Sprint("⚠") + " No user configuration found.\n"` |
| 100, 108 | Error output | `ui.Error.Sprint("✗") + " Failed to output user configuration\n"` |
| 103, 111 | Success | `ui.Success.Sprint("✓") + " User configuration displayed\n"` |
| 177, 187 | Error project settings | `ui.Error.Sprint("✗") + " Failed to check project settings\n"` / `" Not in a Kanuka project directory\n"` |
| 196, 203, 213, 221 | Project config messages | Various error/success messages |
| 216, 224 | Success | `ui.Success.Sprint("✓") + " Project configuration displayed\n"` |

### Direct Prints (fmt.Println/Printf/Print)
| Line | Context | String Pattern |
|------|---------|----------------|
| 85 | Empty JSON | `fmt.Println("{}")` |
| 89-90 | No config hint | `fmt.Println(ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka config init") + " to set up your identity")` |
| 121 | JSON output | `fmt.Println(string(output))` |
| 127-166 | User config text | `outputUserConfigText(config)` - formatted output |
| 184 | Project error JSON | `fmt.Println('{"error": "not in a project directory"}')` |
| 188-189 | Project hint | `fmt.Println(ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " to initialize a project")` |
| 234 | Project JSON | `fmt.Println(string(output))` |
| 239-298 | Project config text | `outputProjectConfigText(config)` - formatted output |

---

## cmd/config_list_devices.go

### spinner.FinalMSG
| Line | Context | String Pattern |
|------|---------|----------------|
| 51 | Error init | `ui.Error.Sprint("✗") + " Failed to initialize project settings\n"` |
| 58 | Not in project | `ui.Error.Sprint("✗") + " Not in a Kanuka project directory\n"` |
| 75 | No devices | `ui.Warning.Sprint("⚠") + " No devices found in this project\n"` |
| 97 | User not found | `ui.Error.Sprint("✗") + " User " + ui.Highlight.Sprint(listDevicesUserEmail) + " not found in this project\n"` |
| 142 | Success | `ui.Success.Sprint("✓") + " Devices listed successfully\n"` |

### Direct Prints (fmt.Println/Printf/Print)
| Line | Context | String Pattern |
|------|---------|----------------|
| 52 | Hint | `fmt.Println(ui.Info.Sprint("→") + " Make sure you're in a Kanuka project directory")` |
| 59 | Hint | `fmt.Println(ui.Info.Sprint("→") + " Run this command from within a Kanuka project")` |
| 114-140 | Devices table | Device listing with grouped emails |

---

## cmd/config_set_default_device.go

### spinner.FinalMSG
| Line | Context | String Pattern |
|------|---------|----------------|
| 39-40 | Invalid device | `ui.Error.Sprint("✗") + " Invalid device name: " + ui.Highlight.Sprint(deviceName) + "\n" + ui.Info.Sprint("→") + " Device name must be alphanumeric with hyphens and underscores only"` |
| 51 | Already set | `ui.Warning.Sprint("⚠") + " Default device name is already set to " + ui.Highlight.Sprint(deviceName)` |
| 65 | Success | `ui.Success.Sprint("✓") + " Default device name set to " + ui.Highlight.Sprint(deviceName)` |

---

## cmd/config_set_project_device.go

### spinner.FinalMSG
| Line | Context | String Pattern |
|------|---------|----------------|
| 59-60 | Invalid device | `ui.Error.Sprint("✗") + " Invalid device name: " + ui.Highlight.Sprint(deviceName) + "\n" + ui.Info.Sprint("→") + " Device name must be alphanumeric with hyphens and underscores only"` |
| 74-75 | Error with hint | `ui.Error.Sprint("✗") + " Failed to initialize project settings: " + err.Error() + "\n" + ui.Info.Sprint("→") + " Use " + ui.Flag.Sprint("--project-uuid") + " to specify a project"` |
| 81-82 | Not in project | `ui.Error.Sprint("✗") + " Not in a Kanuka project directory\n" + ui.Info.Sprint("→") + " Use " + ui.Flag.Sprint("--project-uuid") + " to specify a project"` |
| 103-104 | No UUID | `ui.Error.Sprint("✗") + " Could not determine project UUID\n" + ui.Info.Sprint("→") + " Use " + ui.Flag.Sprint("--project-uuid") + " to specify a project"` |
| 124 | Already set | `ui.Warning.Sprint("⚠") + " Device name is already set to " + ui.Highlight.Sprint(deviceName) + " for this project"` |
| 159-164 | TOML error | Multi-line error with fix instructions |
| 188-199 | Success | `ui.Success.Sprint("✓") + " Device name updated from ... to ..."` or `" Device name set to ..."` |

---

## internal/secrets/keys.go

### fmt.Fprintln to os.Stderr
| Line | Context | String Pattern |
|------|---------|----------------|
| 113 | Incorrect passphrase | `fmt.Fprintln(os.Stderr, ui.Warning.Sprint("✗")+" Incorrect passphrase. Please try again.")` |
| 162 | Incorrect passphrase (TTY) | `fmt.Fprintln(os.Stderr, ui.Warning.Sprint("✗")+" Incorrect passphrase. Please try again.")` |

---

## internal/utils/terminal.go

### fmt.Fprint to os.Stderr
| Line | Context | String Pattern |
|------|---------|----------------|
| 20 | Passphrase prompt | `fmt.Fprint(os.Stderr, prompt)` |
| 22 | Newline after input | `fmt.Fprintln(os.Stderr)` |
| 51 | Passphrase prompt (TTY) | `fmt.Fprint(os.Stderr, prompt)` |
| 53 | Newline after input | `fmt.Fprintln(os.Stderr)` |

---

## Summary Statistics

### By Mechanism
| Mechanism | Count (approx) |
|-----------|----------------|
| spinner.FinalMSG | 75+ |
| fmt.Println/Printf/Print | 120+ |
| utils.WriteToTTY | 3 |
| fmt.Fprint(ln) to stderr | 6 |

### By Category
| Category | Count (approx) |
|----------|----------------|
| Error messages | 50+ |
| Success messages | 20+ |
| Warning messages | 15+ |
| Prompts | 15+ |
| Dry-run output | 30+ |
| Table output | 25+ |
| Help/hint messages | 20+ |

### Key Patterns Observed
1. **spinner.FinalMSG** is used for the primary command result (success/error)
2. **Direct prints** are used for:
   - Interactive prompts (before spinner stops)
   - Dry-run previews (after spinner stops)
   - Tables and detailed output
   - Confirmation dialogs
3. **TTY output** is reserved for sensitive data (private keys in ci-init)
4. **Stderr** is used for passphrase prompts to not interfere with stdout piping
