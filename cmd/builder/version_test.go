package main

import (
	"bytes"
	"regexp"
	"testing"
)

// TestVersion_MatchesRegex covers REQ-CVA-001, REQ-CVA-002.
// Version must match the 0.x semantic: ^v0\.\d+\.\d+$
func TestVersion_MatchesRegex(t *testing.T) {
	t.Parallel()

	pattern := regexp.MustCompile(`^v0\.\d+\.\d+$`)
	if !pattern.MatchString(Version) {
		t.Errorf("Version = %q; does not match 0.x pattern ^v0\\.\\d+\\.\\d+$", Version)
	}
}

// TestVersion_NotEmpty covers REQ-CVA-001.
// Version must not be empty.
func TestVersion_NotEmpty(t *testing.T) {
	t.Parallel()

	if Version == "" {
		t.Error("Version is empty string; expected a semver value like v0.0.0")
	}
}

// TestComposeApp_VersionWired covers REQ-CVA-003.
// After calling composeApp(Config{}), app.Root.Version must equal Version.
func TestComposeApp_VersionWired(t *testing.T) {
	t.Parallel()

	app, err := composeApp(Config{})
	if err != nil {
		t.Fatalf("composeApp returned error: %v", err)
	}

	if app.Root.Version != Version {
		t.Errorf("app.Root.Version = %q; want %q (const Version)", app.Root.Version, Version)
	}
}

// TestVersionFlag covers REQ-CVA-004.
// Running the root command with --version must produce output containing Version.
func TestVersionFlag(t *testing.T) {
	t.Parallel()

	app, err := composeApp(Config{})
	if err != nil {
		t.Fatalf("composeApp returned error: %v", err)
	}

	var buf bytes.Buffer
	app.Root.SetOut(&buf)
	app.Root.SetErr(&buf)
	app.Root.SetArgs([]string{"--version"})
	// Cobra's built-in --version prints to stdout and returns nil.
	_ = app.Root.Execute()

	out := buf.String()
	if !regexp.MustCompile(`v\d+\.\d+\.\d+`).MatchString(out) {
		t.Errorf("--version output %q does not contain a semver string", out)
	}
	if !bytes.Contains(buf.Bytes(), []byte(Version)) {
		t.Errorf("--version output %q does not contain Version %q", out, Version)
	}
}
