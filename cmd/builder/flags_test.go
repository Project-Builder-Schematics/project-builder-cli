// Package main — flags_test.go covers ThemeFlag (pflag.Value) unit tests.
//
// REQ-ID coverage:
//   - theme-profile-detection/REQ-05.1 — invalid --theme value exits non-zero with clear error
package main

import (
	"strings"
	"testing"
)

// Test_ThemeFlag_Set_Light verifies Set("light") succeeds and String() reflects the value.
func Test_ThemeFlag_Set_Light(t *testing.T) {
	t.Parallel()

	var f ThemeFlag
	if err := f.Set("light"); err != nil {
		t.Fatalf("Set(\"light\") returned unexpected error: %v", err)
	}
	if got := f.String(); got != "light" {
		t.Errorf("String() = %q; want %q", got, "light")
	}
}

// Test_ThemeFlag_Set_Dark verifies Set("dark") succeeds and String() reflects the value.
func Test_ThemeFlag_Set_Dark(t *testing.T) {
	t.Parallel()

	var f ThemeFlag
	if err := f.Set("dark"); err != nil {
		t.Fatalf("Set(\"dark\") returned unexpected error: %v", err)
	}
	if got := f.String(); got != "dark" {
		t.Errorf("String() = %q; want %q", got, "dark")
	}
}

// Test_ThemeFlag_Set_Auto verifies Set("auto") succeeds and String() reflects the value.
func Test_ThemeFlag_Set_Auto(t *testing.T) {
	t.Parallel()

	var f ThemeFlag
	if err := f.Set("auto"); err != nil {
		t.Fatalf("Set(\"auto\") returned unexpected error: %v", err)
	}
	if got := f.String(); got != "auto" {
		t.Errorf("String() = %q; want %q", got, "auto")
	}
}

// Test_ThemeFlag_Set_RejectsInvalidValue covers theme-profile-detection/REQ-05.1.
// Set with any value other than light/dark/auto must return a non-nil error whose
// message contains "invalid argument", the bad value, and lists the accepted values.
func Test_ThemeFlag_Set_RejectsInvalidValue(t *testing.T) {
	t.Parallel()

	var f ThemeFlag
	err := f.Set("neon")
	if err == nil {
		t.Fatal("Set(\"neon\") returned nil error; want non-nil (REQ-05.1)")
	}

	msg := err.Error()

	if !strings.Contains(msg, "invalid argument") {
		t.Errorf("error message %q does not contain \"invalid argument\"", msg)
	}
	if !strings.Contains(msg, "neon") {
		t.Errorf("error message %q does not contain the bad value %q", msg, "neon")
	}
	// Must list the accepted values.
	for _, accepted := range []string{"light", "dark", "auto"} {
		if !strings.Contains(msg, accepted) {
			t.Errorf("error message %q does not mention accepted value %q", msg, accepted)
		}
	}
}

// Test_ThemeFlag_Type verifies the stable pflag type identifier.
func Test_ThemeFlag_Type(t *testing.T) {
	t.Parallel()

	var f ThemeFlag
	if got := f.Type(); got != "theme" {
		t.Errorf("Type() = %q; want %q", got, "theme")
	}
}
