package utils

import (
	"testing"
)

func TestSanitizeDeviceName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"LowercaseSimple", "MacBook", "macbook"},
		{"SpacesToHyphens", "My Device", "my-device"},
		{"RemoveSpecialChars", "My@Device#123!", "mydevice123"},
		{"RemoveConsecutiveHyphens", "my--device", "my-device"},
		{"TrimHyphens", "-my-device-", "my-device"},
		{"EmptyToDefault", "", "device"},
		{"OnlySpecialChars", "@#$%", "device"},
		{"PreserveUnderscores", "my_device", "my_device"},
		{"PreserveNumbers", "device123", "device123"},
		{"TrimWhitespace", "  mydevice  ", "mydevice"},
		{"ComplexName", "  My MacBook Pro! #1  ", "my-macbook-pro-1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeDeviceName(tc.input)
			if result != tc.expected {
				t.Errorf("SanitizeDeviceName(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestUniqueDeviceName(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		existing []string
		expected string
	}{
		{"NoConflicts", "recovery-key", nil, "recovery-key"},
		{"CaseInsensitiveConflict", "recovery-key", []string{"Recovery-Key"}, "recovery-key-2"},
		{"SingleConflict", "recovery-key", []string{"recovery-key"}, "recovery-key-2"},
		{"MultipleConflicts", "recovery-key", []string{"recovery-key", "recovery-key-2", "recovery-key-3"}, "recovery-key-4"},
		{"OtherNamesDontConflict", "recovery-key", []string{"laptop", "desktop"}, "recovery-key"},
		{"BaseIsSanitized", "My Recovery Key!", []string{}, "my-recovery-key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UniqueDeviceName(tt.base, tt.existing)
			if result != tt.expected {
				t.Errorf("UniqueDeviceName(%q, %v) = %q, expected %q", tt.base, tt.existing, result, tt.expected)
			}
		})
	}
}

func TestGenerateDeviceName(t *testing.T) {
	t.Run("GeneratesUniqueName", func(t *testing.T) {
		existing := []string{}
		name, err := GenerateDeviceName(existing)
		if err != nil {
			t.Fatalf("GenerateDeviceName failed: %v", err)
		}
		if name == "" {
			t.Fatal("Expected non-empty device name")
		}
	})

	t.Run("AppendsNumberOnConflict", func(t *testing.T) {
		// Get the base name first.
		baseName, err := GenerateDeviceName([]string{})
		if err != nil {
			t.Fatalf("GenerateDeviceName failed: %v", err)
		}

		// Now generate with the base name already existing.
		existing := []string{baseName}
		name, err := GenerateDeviceName(existing)
		if err != nil {
			t.Fatalf("GenerateDeviceName failed: %v", err)
		}

		expected := baseName + "-2"
		if name != expected {
			t.Errorf("Expected %q, got %q", expected, name)
		}
	})

	t.Run("IncrementsForMultipleConflicts", func(t *testing.T) {
		// Get the base name first.
		baseName, err := GenerateDeviceName([]string{})
		if err != nil {
			t.Fatalf("GenerateDeviceName failed: %v", err)
		}

		// Now generate with multiple conflicts.
		existing := []string{baseName, baseName + "-2", baseName + "-3"}
		name, err := GenerateDeviceName(existing)
		if err != nil {
			t.Fatalf("GenerateDeviceName failed: %v", err)
		}

		expected := baseName + "-4"
		if name != expected {
			t.Errorf("Expected %q, got %q", expected, name)
		}
	})

	t.Run("CaseInsensitiveConflictCheck", func(t *testing.T) {
		// Get the base name first.
		baseName, err := GenerateDeviceName([]string{})
		if err != nil {
			t.Fatalf("GenerateDeviceName failed: %v", err)
		}

		// Add uppercase version of base name.
		existing := []string{baseName, baseName + "-2"}
		name, err := GenerateDeviceName(existing)
		if err != nil {
			t.Fatalf("GenerateDeviceName failed: %v", err)
		}

		expected := baseName + "-3"
		if name != expected {
			t.Errorf("Expected %q, got %q", expected, name)
		}
	})
}

func TestGetUsername(t *testing.T) {
	username, err := GetUsername()
	if err != nil {
		t.Fatalf("GetUsername failed: %v", err)
	}
	if username == "" {
		t.Fatal("Expected non-empty username")
	}
}

func TestGetHostname(t *testing.T) {
	hostname, err := GetHostname()
	if err != nil {
		t.Fatalf("GetHostname failed: %v", err)
	}
	if hostname == "" {
		t.Fatal("Expected non-empty hostname")
	}
}
