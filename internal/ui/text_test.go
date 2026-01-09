package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestFormatterWithColor(t *testing.T) {
	// Ensure NO_COLOR is not set for this test.
	os.Unsetenv("NO_COLOR")
	// Force color output for testing.
	color.NoColor = false

	// Code formatter should not have backticks when color is enabled.
	result := Code.Sprint("kanuka secrets init")
	if strings.Contains(result, "`") {
		t.Errorf("Code.Sprint should not contain backticks when color is enabled, got: %s", result)
	}

	// Verify it contains ANSI escape codes (color output).
	if !strings.Contains(result, "\x1b[") {
		t.Errorf("Code.Sprint should contain ANSI escape codes when color is enabled, got: %s", result)
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

func TestFormatterSprintfWithColor(t *testing.T) {
	// Ensure NO_COLOR is not set for this test.
	os.Unsetenv("NO_COLOR")
	// Force color output for testing.
	color.NoColor = false

	result := Highlight.Sprintf("user: %s", "test@example.com")

	// Should not have quotes when color is enabled.
	if strings.HasPrefix(result, "'") || strings.HasSuffix(result, "'") {
		t.Errorf("Highlight.Sprintf should not have quotes when color is enabled, got: %s", result)
	}

	// Should contain the formatted text.
	if !strings.Contains(result, "user: test@example.com") {
		t.Errorf("Highlight.Sprintf should contain formatted text, got: %s", result)
	}
}

func TestNoColorFunction(t *testing.T) {
	// Test with NO_COLOR set.
	os.Setenv("NO_COLOR", "1")
	if !noColor() {
		t.Error("noColor() should return true when NO_COLOR is set")
	}
	os.Unsetenv("NO_COLOR")

	// Test with color.NoColor set.
	originalNoColor := color.NoColor
	color.NoColor = true
	if !noColor() {
		t.Error("noColor() should return true when color.NoColor is true")
	}
	color.NoColor = originalNoColor
}

func TestAllFormattersExist(t *testing.T) {
	// Verify all formatters are initialized and usable.
	formatters := []struct {
		name      string
		formatter Formatter
	}{
		{"Code", Code},
		{"Path", Path},
		{"Flag", Flag},
		{"Success", Success},
		{"Error", Error},
		{"Warning", Warning},
		{"Info", Info},
		{"Highlight", Highlight},
		{"Muted", Muted},
	}

	for _, f := range formatters {
		t.Run(f.name, func(t *testing.T) {
			if f.formatter.color == nil {
				t.Errorf("%s formatter has nil color", f.name)
			}
			// Test that Sprint doesn't panic.
			result := f.formatter.Sprint("test")
			if result == "" {
				t.Errorf("%s.Sprint returned empty string", f.name)
			}
		})
	}
}

func TestMultipleArguments(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	// Test Sprint with multiple arguments.
	result := Code.Sprint("kanuka", " ", "secrets")
	want := "`kanuka secrets`"
	if result != want {
		t.Errorf("Code.Sprint with multiple args = %q, want %q", result, want)
	}
}
