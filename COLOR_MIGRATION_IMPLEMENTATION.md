# Color Migration Implementation Plan

## Overview

This document outlines the migration from direct `color.XxxString()` calls to a semantic UI text formatting system that properly supports `NO_COLOR` environments.

## Problem Statement

### Current Issues

1. **Lack of Semantic Meaning**: Colors are hard-coded throughout the codebase without semantic context:
   - Yellow is used for commands (`kanuka secrets init`), file paths (`.kanuka/public_keys/`), flags (`--device-name`), warning symbols (`⚠`), and user input values
   - This makes it impossible to change how a specific type of content is displayed without auditing every usage

2. **Poor NO_COLOR Support**: While `fatih/color` respects `NO_COLOR`, the fallback is plain text with no visual distinction:
   - Commands like `kanuka secrets init` become indistinguishable from surrounding prose
   - Users lose important context about what they should copy/run

3. **Maintenance Burden**: With 366 direct color calls across the codebase, any styling change requires a large, error-prone refactor

### Current Usage Statistics

```
175 color.YellowString  - Commands, paths, flags, warnings, values
151 color.RedString     - Errors, failure indicators
107 color.CyanString    - Info hints, highlights, project names
 43 color.GreenString   - Success indicators
  4 color.WhiteString   - Neutral emphasis
  3 color.HiBlackString - Muted/de-emphasized text
```

## Proposed Solution

### Semantic Text Formatters

Create an `internal/ui` package with semantic formatters that:
1. Apply appropriate colors when colors are enabled
2. Apply text decorations (backticks, quotes, etc.) when `NO_COLOR` is set
3. Provide clear semantic meaning to each type of text

### Formatter Categories

| Formatter   | Color     | NO_COLOR Fallback | Use Case |
|-------------|-----------|-------------------|----------|
| `Code`      | Yellow    | \`backticks\`     | Commands to run, code snippets |
| `Path`      | Yellow    | none              | File/directory paths (self-evident) |
| `Flag`      | Yellow    | none              | CLI flags (-- prefix is enough) |
| `Success`   | Green     | none              | Success indicators (checkmarks) |
| `Error`     | Red       | none              | Error indicators, failure messages |
| `Warning`   | Yellow    | none              | Warning indicators |
| `Info`      | Cyan      | none              | Hints, tips, arrows |
| `Highlight` | Cyan      | 'single quotes'   | User values, project/device names |
| `Muted`     | HiBlack   | (parentheses)     | De-emphasized, secondary text |

### Example Transformation

**Before:**
```go
finalMessage := color.RedString("✗") + " Kānuka has not been initialized\n" +
    color.CyanString("→") + " Run " + color.YellowString("kanuka secrets init") + " instead"
```

**After:**
```go
finalMessage := ui.Error.Sprint("✗") + " Kānuka has not been initialized\n" +
    ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " instead"
```

**With NO_COLOR:**
```
✗ Kānuka has not been initialized
→ Run `kanuka secrets init` instead
```

---

## Tickets

---

### KAN-020: Create `internal/ui` Package with Semantic Text Formatters

**Priority**: High  
**Estimated Effort**: Small (1-2 hours)  
**Dependencies**: None

#### Context

The codebase currently uses `fatih/color` directly with 366 calls spread across command files. This makes it difficult to:
- Maintain consistent styling
- Support NO_COLOR environments with meaningful fallbacks
- Change how specific types of content are displayed

#### Rationale

A semantic UI package provides:
1. **Meaningful abstractions**: `ui.Code.Sprint()` vs `color.YellowString()`
2. **Centralized styling**: Change all command formatting in one place
3. **Proper NO_COLOR support**: Fallback to text decorations, not just plain text
4. **Type safety**: Compile-time errors if a formatter is misused

#### Implementation

Create `internal/ui/text.go`:

```go
// Package ui provides semantic text formatting for CLI output.
// It supports colored output when available and falls back to
// text decorations (backticks, quotes) when NO_COLOR is set.
package ui

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

// Formatter applies semantic formatting to text.
type Formatter struct {
	color  *color.Color
	prefix string
	suffix string
}

// Sprint formats the arguments and returns the resulting string.
func (f Formatter) Sprint(a ...interface{}) string {
	text := fmt.Sprint(a...)
	if noColor() {
		return f.prefix + text + f.suffix
	}
	return f.color.Sprint(text)
}

// Sprintf formats according to a format specifier and returns the resulting string.
func (f Formatter) Sprintf(format string, a ...interface{}) string {
	text := fmt.Sprintf(format, a...)
	if noColor() {
		return f.prefix + text + f.suffix
	}
	return f.color.Sprint(text)
}

// noColor returns true if color output should be disabled.
func noColor() bool {
	// Check NO_COLOR environment variable (https://no-color.org/)
	if _, exists := os.LookupEnv("NO_COLOR"); exists {
		return true
	}
	// Also respect fatih/color's detection (terminal capability, TERM=dumb, etc.)
	return color.NoColor
}

// Semantic formatters for different types of CLI output.
var (
	// Code formats runnable commands or code snippets.
	// Yellow with color, `backticks` without.
	Code = Formatter{color.New(color.FgYellow), "`", "`"}

	// Path formats file or directory paths.
	// Yellow with color, no decoration without (paths are self-evident).
	Path = Formatter{color.New(color.FgYellow), "", ""}

	// Flag formats CLI flags like --verbose or --dry-run.
	// Yellow with color, no decoration without (-- prefix is sufficient).
	Flag = Formatter{color.New(color.FgYellow), "", ""}

	// Success formats success indicators and messages.
	// Green with color, unchanged without.
	Success = Formatter{color.New(color.FgGreen), "", ""}

	// Error formats error indicators and messages.
	// Red with color, unchanged without.
	Error = Formatter{color.New(color.FgRed), "", ""}

	// Warning formats warning indicators and messages.
	// Yellow with color, unchanged without.
	Warning = Formatter{color.New(color.FgYellow), "", ""}

	// Info formats informational hints, tips, and directional indicators.
	// Cyan with color, unchanged without.
	Info = Formatter{color.New(color.FgCyan), "", ""}

	// Highlight formats emphasized user values like emails, project names, device names.
	// Cyan with color, 'single quotes' without.
	Highlight = Formatter{color.New(color.FgCyan), "'", "'"}

	// Muted formats de-emphasized or secondary text.
	// Gray with color, (parentheses) without.
	Muted = Formatter{color.New(color.FgHiBlack), "(", ")"}
)
```

#### Steps

1. Create `internal/ui/` directory
2. Create `internal/ui/text.go` with the implementation above
3. Create `internal/ui/text_test.go` with unit tests (see testing requirements)
4. Run `golangci-lint run ./internal/ui/...`
5. Run `go test -v ./internal/ui/...`

#### Testing Requirements

Create `internal/ui/text_test.go`:

```go
package ui

import (
	"os"
	"strings"
	"testing"
)

func TestFormatterWithColor(t *testing.T) {
	// Ensure NO_COLOR is not set for this test.
	os.Unsetenv("NO_COLOR")

	// Code formatter should not have backticks when color is enabled.
	result := Code.Sprint("kanuka secrets init")
	if strings.Contains(result, "`") {
		t.Errorf("Code.Sprint should not contain backticks when color is enabled, got: %s", result)
	}
}

func TestFormatterWithNoColor(t *testing.T) {
	// Set NO_COLOR for this test.
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	tests := []struct {
		name      string
		formatter Formatter
		input     string
		want      string
	}{
		{"Code adds backticks", Code, "kanuka secrets init", "`kanuka secrets init`"},
		{"Path has no decoration", Path, ".env.local", ".env.local"},
		{"Flag has no decoration", Flag, "--dry-run", "--dry-run"},
		{"Success has no decoration", Success, "✓", "✓"},
		{"Error has no decoration", Error, "✗", "✗"},
		{"Warning has no decoration", Warning, "⚠", "⚠"},
		{"Info has no decoration", Info, "→", "→"},
		{"Highlight adds quotes", Highlight, "testuser@example.com", "'testuser@example.com'"},
		{"Muted adds parentheses", Muted, "unknown", "(unknown)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.formatter.Sprint(tt.input)
			if got != tt.want {
				t.Errorf("%s.Sprint(%q) = %q, want %q", tt.name, tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatterSprintf(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	result := Code.Sprintf("kanuka secrets %s", "encrypt")
	want := "`kanuka secrets encrypt`"
	if result != want {
		t.Errorf("Code.Sprintf() = %q, want %q", result, want)
	}
}
```

#### Acceptance Criteria

- [ ] `internal/ui/text.go` exists with all formatters defined
- [ ] `internal/ui/text_test.go` exists with comprehensive tests
- [ ] All tests pass: `go test -v ./internal/ui/...`
- [ ] Linter passes: `golangci-lint run ./internal/ui/...`
- [ ] Package can be imported: `import "github.com/PolarWolf314/kanuka/internal/ui"`

---

### KAN-021: Migrate `secrets_clean.go` to Semantic UI Formatters (Pilot)

**Priority**: High  
**Estimated Effort**: Small (30 minutes)  
**Dependencies**: KAN-020

#### Context

`secrets_clean.go` is one of the smallest command files with only 4 color usages. It serves as an ideal pilot for the migration pattern before tackling larger files.

#### Rationale

Starting with a small file allows us to:
1. Validate the approach works in practice
2. Establish migration patterns for other files
3. Catch any issues with the ui package early
4. Create a reference implementation for other migrations

#### Current Implementation

```go
fmt.Println(color.RedString("✗") + " Kanuka has not been initialized")
fmt.Println(color.CyanString("→") + " Run " + color.YellowString("kanuka secrets init") + " first")
// ...
fmt.Println(color.GreenString("✓") + " No orphaned entries found. Nothing to clean.")
// ...
fmt.Printf("%s Removed %d orphaned file(s)\n", color.GreenString("✓"), len(orphans))
```

#### New Implementation

```go
import "github.com/PolarWolf314/kanuka/internal/ui"

// ...

fmt.Println(ui.Error.Sprint("✗") + " Kanuka has not been initialized")
fmt.Println(ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first")
// ...
fmt.Println(ui.Success.Sprint("✓") + " No orphaned entries found. Nothing to clean.")
// ...
fmt.Printf("%s Removed %d orphaned file(s)\n", ui.Success.Sprint("✓"), len(orphans))
```

#### Steps

1. Open `cmd/secrets_clean.go`
2. Add import: `"github.com/PolarWolf314/kanuka/internal/ui"`
3. Replace each color usage with the appropriate semantic formatter:
   - `color.RedString("✗")` → `ui.Error.Sprint("✗")`
   - `color.CyanString("→")` → `ui.Info.Sprint("→")`
   - `color.YellowString("kanuka secrets init")` → `ui.Code.Sprint("kanuka secrets init")`
   - `color.GreenString("✓")` → `ui.Success.Sprint("✓")`
4. Remove the `"github.com/fatih/color"` import if no longer used
5. Run `go build ./...`
6. Run `golangci-lint run ./cmd/secrets_clean.go`
7. Test manually with and without `NO_COLOR=1`

#### Testing Requirements

Manual testing:

```bash
# Test with colors
kanuka secrets clean

# Test without colors
NO_COLOR=1 kanuka secrets clean

# Verify in uninitialized directory
cd /tmp && mkdir test-clean && cd test-clean
kanuka secrets clean
NO_COLOR=1 kanuka secrets clean
```

Expected NO_COLOR output:
```
✗ Kanuka has not been initialized
→ Run `kanuka secrets init` first
```

#### Acceptance Criteria

- [ ] `cmd/secrets_clean.go` uses `internal/ui` instead of `fatih/color`
- [ ] No direct `color.XxxString()` calls remain in the file
- [ ] Build succeeds: `go build ./...`
- [ ] Linter passes: `golangci-lint run ./cmd/secrets_clean.go`
- [ ] Output looks correct with colors enabled
- [ ] Output looks correct with `NO_COLOR=1` (commands in backticks)

---

### KAN-022: Migrate `secrets_log.go` to Semantic UI Formatters

**Priority**: Medium  
**Estimated Effort**: Small (30 minutes)  
**Dependencies**: KAN-020

#### Context

`secrets_log.go` has only 2 color usages, making it a quick migration.

#### Current Implementation

```go
fmt.Println(color.RedString("✗") + " Kānuka has not been initialized")
fmt.Println(color.CyanString("→") + " Run " + color.YellowString("kanuka secrets init") + " first")
```

#### New Implementation

```go
fmt.Println(ui.Error.Sprint("✗") + " Kānuka has not been initialized")
fmt.Println(ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first")
```

#### Steps

1. Add import: `"github.com/PolarWolf314/kanuka/internal/ui"`
2. Replace color usages with semantic formatters
3. Remove unused `color` import
4. Run build and linter

#### Testing Requirements

```bash
kanuka secrets log
NO_COLOR=1 kanuka secrets log
```

#### Acceptance Criteria

- [ ] No direct `color.XxxString()` calls remain
- [ ] Build and linter pass
- [ ] Output correct with and without NO_COLOR

---

### KAN-023: Migrate `secrets_access.go` to Semantic UI Formatters

**Priority**: Medium  
**Estimated Effort**: Small (45 minutes)  
**Dependencies**: KAN-020

#### Context

`secrets_access.go` has 10 color usages including status indicators and legend text.

#### Current Implementation

```go
fmt.Println(color.RedString("✗") + " Kanuka has not been initialized")
fmt.Println(color.CyanString("→") + " Run " + color.YellowString("kanuka secrets init") + " first")
fmt.Printf("Project: %s\n", color.CyanString(result.ProjectName))
displayEmail = color.HiBlackString("(unknown)")
statusStr = color.GreenString("✓") + " active"
statusStr = color.YellowString("⚠") + " pending"
statusStr = color.RedString("✗") + " orphan"
fmt.Printf("  %s active  - ...\n", color.GreenString("✓"))
fmt.Printf("  %s pending - ...\n", color.YellowString("⚠"))
fmt.Printf("  %s orphan  - ...\n", color.RedString("✗"))
```

#### New Implementation

```go
fmt.Println(ui.Error.Sprint("✗") + " Kanuka has not been initialized")
fmt.Println(ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first")
fmt.Printf("Project: %s\n", ui.Highlight.Sprint(result.ProjectName))
displayEmail = ui.Muted.Sprint("unknown")
statusStr = ui.Success.Sprint("✓") + " active"
statusStr = ui.Warning.Sprint("⚠") + " pending"
statusStr = ui.Error.Sprint("✗") + " orphan"
fmt.Printf("  %s active  - ...\n", ui.Success.Sprint("✓"))
fmt.Printf("  %s pending - ...\n", ui.Warning.Sprint("⚠"))
fmt.Printf("  %s orphan  - ...\n", ui.Error.Sprint("✗"))
```

#### Semantic Mapping

| Current | New | Rationale |
|---------|-----|-----------|
| `color.RedString("✗")` | `ui.Error.Sprint("✗")` | Error/failure indicator |
| `color.CyanString("→")` | `ui.Info.Sprint("→")` | Informational hint |
| `color.YellowString("kanuka ...")` | `ui.Code.Sprint("kanuka ...")` | Runnable command |
| `color.CyanString(projectName)` | `ui.Highlight.Sprint(projectName)` | Emphasized user value |
| `color.HiBlackString("(unknown)")` | `ui.Muted.Sprint("unknown")` | De-emphasized text |
| `color.GreenString("✓")` | `ui.Success.Sprint("✓")` | Success indicator |
| `color.YellowString("⚠")` | `ui.Warning.Sprint("⚠")` | Warning indicator |

#### Steps

1. Add import: `"github.com/PolarWolf314/kanuka/internal/ui"`
2. Replace each color usage per the mapping above
3. Remove unused `color` import
4. Run build and linter
5. Test with `kanuka secrets access`

#### Testing Requirements

```bash
# In an initialized project with users
kanuka secrets access
NO_COLOR=1 kanuka secrets access
```

#### Acceptance Criteria

- [ ] No direct `color.XxxString()` calls remain
- [ ] Build and linter pass
- [ ] Status legend displays correctly
- [ ] Project name highlighted/quoted appropriately
- [ ] "(unknown)" shows as muted or "(unknown)" with NO_COLOR

---

### KAN-024: Migrate `secrets_init.go` to Semantic UI Formatters

**Priority**: Medium  
**Estimated Effort**: Medium (1 hour)  
**Dependencies**: KAN-020

#### Context

`secrets_init.go` has 12 color usages including success messages, hints, and command suggestions.

#### Current Implementation (excerpt)

```go
finalMessage := color.RedString("✗") + " Kānuka has already been initialized\n" +
    color.CyanString("→") + " Run " + color.YellowString("kanuka secrets create") + " instead"

spinner.FinalMSG = color.RedString("✗") + " User configuration is incomplete\n" +
    color.CyanString("→") + " Run " + color.YellowString("kanuka config init") + " first to set up your identity"

fmt.Println(color.YellowString("⚠") + " User configuration not found.\n")

finalMessage := color.GreenString("✓") + " Kānuka initialized successfully!\n\n" +
    color.CyanString("→") + " Run " + color.YellowString("kanuka secrets encrypt") + " to encrypt your existing .env files\n\n" +
    color.CyanString("Tip:") + " Working in a monorepo? You have two options:\n" +
    "     " + color.YellowString("kanuka secrets encrypt services/api/.env") + "\n" +
    "     " + color.YellowString("cd services/api && kanuka secrets init")
```

#### New Implementation (excerpt)

```go
finalMessage := ui.Error.Sprint("✗") + " Kānuka has already been initialized\n" +
    ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets create") + " instead"

spinner.FinalMSG = ui.Error.Sprint("✗") + " User configuration is incomplete\n" +
    ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka config init") + " first to set up your identity"

fmt.Println(ui.Warning.Sprint("⚠") + " User configuration not found.\n")

finalMessage := ui.Success.Sprint("✓") + " Kānuka initialized successfully!\n\n" +
    ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets encrypt") + " to encrypt your existing .env files\n\n" +
    ui.Info.Sprint("Tip:") + " Working in a monorepo? You have two options:\n" +
    "     " + ui.Code.Sprint("kanuka secrets encrypt services/api/.env") + "\n" +
    "     " + ui.Code.Sprint("cd services/api && kanuka secrets init")
```

#### Steps

1. Add import: `"github.com/PolarWolf314/kanuka/internal/ui"`
2. Replace each color usage with appropriate semantic formatter
3. Remove unused `color` import
4. Run build and linter
5. Test all scenarios (new init, already initialized, no user config)

#### Testing Requirements

```bash
# Test already initialized
cd existing-project && kanuka secrets init
NO_COLOR=1 kanuka secrets init

# Test new init
cd /tmp && mkdir new-proj && cd new-proj
kanuka secrets init --yes
NO_COLOR=1 kanuka secrets init --yes
```

#### Acceptance Criteria

- [ ] No direct `color.XxxString()` calls remain
- [ ] Build and linter pass
- [ ] All command suggestions appear in backticks with NO_COLOR
- [ ] Monorepo tip commands are properly formatted

---

### KAN-025: Migrate `secrets_create.go` to Semantic UI Formatters

**Priority**: Medium  
**Estimated Effort**: Medium (1 hour)  
**Dependencies**: KAN-020

#### Context

`secrets_create.go` has 15 color usages including success messages with file paths and multi-step instructions.

#### Semantic Mapping

| Content Type | Current | New |
|--------------|---------|-----|
| Error indicator | `color.RedString("✗")` | `ui.Error.Sprint("✗")` |
| Success indicator | `color.GreenString("✓")` | `ui.Success.Sprint("✓")` |
| Commands | `color.YellowString("kanuka ...")` | `ui.Code.Sprint("kanuka ...")` |
| File paths | `color.YellowString(".kanuka/...")` | `ui.Path.Sprint(".kanuka/...")` |
| Flags | `color.YellowString("--device-name")` | `ui.Flag.Sprint("--device-name")` |
| User email | `color.YellowString(userEmail)` | `ui.Highlight.Sprint(userEmail)` |
| Device name | `color.CyanString(deviceName)` | `ui.Highlight.Sprint(deviceName)` |
| Hints | `color.CyanString("→")` | `ui.Info.Sprint("→")` |
| Instructions | `color.WhiteString("Commit your...")` | Plain text (remove color) |

#### Steps

1. Add import: `"github.com/PolarWolf314/kanuka/internal/ui"`
2. Carefully map each color usage to semantic formatters
3. For `color.WhiteString()` calls, evaluate if they need any formatting or can be plain text
4. Remove unused `color` import
5. Run build and linter
6. Test create flow

#### Testing Requirements

```bash
# In initialized project
kanuka secrets create --force
NO_COLOR=1 kanuka secrets create --force

# Verify file paths and emails are formatted appropriately
```

#### Acceptance Criteria

- [ ] No direct `color.XxxString()` calls remain
- [ ] File paths are distinguishable from prose
- [ ] User emails and device names are highlighted
- [ ] Commands appear in backticks with NO_COLOR

---

### KAN-026: Migrate `secrets_encrypt.go` to Semantic UI Formatters

**Priority**: Medium  
**Estimated Effort**: Medium (1 hour)  
**Dependencies**: KAN-020

#### Context

`secrets_encrypt.go` has 15 color usages including dry-run output and file path listings.

#### Key Mappings

| Content | Current | New |
|---------|---------|-----|
| `.kanuka` file references | `color.YellowString(".kanuka")` | `ui.Path.Sprint(".kanuka")` |
| `.env` file references | `color.YellowString(".env")` | `ui.Path.Sprint(".env")` |
| File paths in dry-run | `color.CyanString(relPath)` | `ui.Path.Sprint(relPath)` |
| Output files in dry-run | `color.GreenString(kanukaFile)` | `ui.Success.Sprint(kanukaFile)` |
| Dry-run label | `color.YellowString("[dry-run]")` | `ui.Warning.Sprint("[dry-run]")` |
| Note text | `color.YellowString("Note:")` | `ui.Info.Sprint("Note:")` |

#### Steps

1. Add import and replace color usages
2. Pay special attention to the dry-run output formatting
3. Test with actual files

#### Testing Requirements

```bash
kanuka secrets encrypt --dry-run
NO_COLOR=1 kanuka secrets encrypt --dry-run

kanuka secrets encrypt
NO_COLOR=1 kanuka secrets encrypt
```

#### Acceptance Criteria

- [ ] Dry-run output clearly shows source → destination
- [ ] File paths are formatted appropriately
- [ ] Note about non-deterministic encryption is visible

---

### KAN-027: Migrate `secrets_decrypt.go` to Semantic UI Formatters

**Priority**: Medium  
**Estimated Effort**: Medium (1 hour)  
**Dependencies**: KAN-020

#### Context

`secrets_decrypt.go` has similar structure to encrypt with ~15 color usages.

#### Steps

Follow the same pattern as KAN-026.

#### Acceptance Criteria

- [ ] Mirrors the formatting approach of encrypt
- [ ] Dry-run output is clear
- [ ] NO_COLOR output is usable

---

### KAN-028: Migrate `secrets_register.go` to Semantic UI Formatters

**Priority**: Medium  
**Estimated Effort**: Medium (1 hour)  
**Dependencies**: KAN-020

#### Context

`secrets_register.go` has ~20 color usages including user emails, dry-run output, and multi-step success messages.

#### Key Considerations

- User emails should use `ui.Highlight`
- Dry-run sections should use `ui.Warning` for the label
- File paths should use `ui.Path`

#### Acceptance Criteria

- [ ] User emails are quoted with NO_COLOR
- [ ] Dry-run output is clear
- [ ] Success message with next steps is readable

---

### KAN-029: Migrate `secrets_revoke.go` to Semantic UI Formatters

**Priority**: Medium  
**Estimated Effort**: Medium (1 hour)  
**Dependencies**: KAN-020

#### Context

`secrets_revoke.go` has ~25 color usages with complex conditional formatting for device names and dry-run output.

#### Acceptance Criteria

- [ ] Device names and emails are properly highlighted
- [ ] Dry-run clearly shows what would be revoked
- [ ] NO_COLOR output is usable

---

### KAN-030: Migrate Remaining Command Files to Semantic UI Formatters

**Priority**: Medium  
**Estimated Effort**: Large (3-4 hours)  
**Dependencies**: KAN-020

#### Context

Remaining files to migrate:

| File | Approximate Usages |
|------|-------------------|
| `secrets_export.go` | 8 |
| `secrets_import.go` | 10 |
| `secrets_rotate.go` | 12 |
| `secrets_status.go` | 15 |
| `secrets_doctor.go` | 8 |
| `secrets_sync.go` | 10 |
| `config_init.go` | 10 |
| `config_show.go` | 8 |
| `config_list_devices.go` | 6 |
| `config_rename_device.go` | 8 |
| `config_set_device_name.go` | 8 |

#### Steps

1. Work through each file systematically
2. Apply consistent semantic mappings
3. Test each command after migration
4. Remove all direct `color` imports from cmd/ when complete

#### Acceptance Criteria

- [ ] No `color.XxxString()` calls remain in `cmd/` directory
- [ ] All commands work correctly with colors
- [ ] All commands work correctly with NO_COLOR
- [ ] `grep -r "color\." cmd/` returns no results (except potentially for import cleanup)

---

### KAN-031: Update Integration Tests for NO_COLOR Output

**Priority**: Medium  
**Estimated Effort**: Medium (2 hours)  
**Dependencies**: KAN-030

#### Context

Some integration tests check for specific output strings. After migration, the NO_COLOR output will change (e.g., commands will have backticks).

#### Rationale

Tests should verify that:
1. Semantic meaning is preserved
2. NO_COLOR fallbacks work correctly
3. Output remains parseable/useful

#### Steps

1. Run all integration tests to identify failures
2. Update assertions that check for colored output
3. Consider adding explicit NO_COLOR test cases for critical flows
4. Ensure tests don't rely on ANSI escape codes

#### Testing Requirements

```bash
# Run all tests with NO_COLOR
NO_COLOR=1 go test -v ./test/integration/...
```

#### Acceptance Criteria

- [ ] All integration tests pass with colors enabled
- [ ] All integration tests pass with NO_COLOR=1
- [ ] No tests depend on specific ANSI escape sequences

---

### KAN-032: Add UI Package Documentation and Usage Guide

**Priority**: Low  
**Estimated Effort**: Small (30 minutes)  
**Dependencies**: KAN-030

#### Context

Future contributors need to understand how to use the UI package correctly.

#### Steps

1. Add comprehensive package documentation to `internal/ui/text.go`
2. Add a `README.md` in `internal/ui/` with:
   - Quick reference table of all formatters
   - Examples of correct usage
   - Guidelines for choosing the right formatter
3. Update `AGENTS.md` with UI formatting guidelines

#### Content for `internal/ui/README.md`

```markdown
# UI Package

Semantic text formatting for Kanuka CLI output.

## Quick Reference

| Formatter | Use For | Color | NO_COLOR |
|-----------|---------|-------|----------|
| `ui.Code` | Commands to run | Yellow | \`backticks\` |
| `ui.Path` | File/directory paths | Yellow | plain |
| `ui.Flag` | CLI flags | Yellow | plain |
| `ui.Success` | Success indicators (✓) | Green | plain |
| `ui.Error` | Error indicators (✗) | Red | plain |
| `ui.Warning` | Warning indicators (⚠) | Yellow | plain |
| `ui.Info` | Hints and tips (→) | Cyan | plain |
| `ui.Highlight` | User values, names | Cyan | 'quotes' |
| `ui.Muted` | De-emphasized text | Gray | (parens) |

## Usage

```go
import "github.com/PolarWolf314/kanuka/internal/ui"

// Format a command
ui.Code.Sprint("kanuka secrets init")

// Format with values
ui.Code.Sprintf("kanuka secrets register --user %s", email)

// Compose a message
msg := ui.Error.Sprint("✗") + " Something went wrong\n" +
    ui.Info.Sprint("→") + " Try " + ui.Code.Sprint("kanuka secrets doctor")
```

## Guidelines

1. **Commands** → Always use `ui.Code` for anything the user should copy/run
2. **File paths** → Use `ui.Path` for paths, but `ui.Code` if it's part of a command
3. **User input** → Use `ui.Highlight` for emails, project names, device names
4. **Indicators** → Match the indicator type: Success/Error/Warning
5. **Don't over-format** → Plain text is fine for most prose
```

#### Acceptance Criteria

- [ ] Package has godoc comments
- [ ] README exists with quick reference
- [ ] AGENTS.md updated with formatting guidelines

---

## Migration Order

Recommended order based on dependencies and complexity:

1. **KAN-020**: Create UI package (required first)
2. **KAN-021**: Pilot with `secrets_clean.go` (validate approach)
3. **KAN-022-029**: Migrate individual command files (can be parallelized)
4. **KAN-030**: Migrate remaining files
5. **KAN-031**: Update integration tests
6. **KAN-032**: Add documentation

## Success Metrics

After complete migration:

1. `grep -r "fatih/color" cmd/` returns no results
2. All tests pass with and without NO_COLOR
3. Commands in output are visually distinct (colored or backticked)
4. Single source of truth for all CLI text styling

## Rollback Plan

If issues arise:
1. The `fatih/color` package remains a dependency (used by ui package)
2. Individual files can be reverted independently
3. The ui package can coexist with direct color calls during migration
