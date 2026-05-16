// Package main — flags.go defines pflag.Value implementations for custom CLI flags.
package main

import (
	"fmt"
	"strings"
)

// ThemeFlag implements pflag.Value for the --theme flag.
// Accepted values: "light", "dark", "auto" (case-insensitive).
// Any other value causes Set to return a descriptive error (REQ-05.1).
type ThemeFlag string

const (
	// ThemeAuto is the default: resolve appearance from the environment.
	ThemeAuto ThemeFlag = "auto"

	// ThemeLight forces a light terminal appearance.
	ThemeLight ThemeFlag = "light"

	// ThemeDark forces a dark terminal appearance.
	ThemeDark ThemeFlag = "dark"
)

// String returns the current flag value as a string.
// Implements pflag.Value.
func (t *ThemeFlag) String() string {
	if t == nil || *t == "" {
		return string(ThemeAuto)
	}
	return string(*t)
}

// Set parses and validates the flag value.
// Returns a descriptive error for any value that is not light, dark, or auto.
// Implements pflag.Value (REQ-05.1).
func (t *ThemeFlag) Set(v string) error {
	switch strings.ToLower(v) {
	case "light", "dark", "auto":
		*t = ThemeFlag(strings.ToLower(v))
		return nil
	default:
		return fmt.Errorf(
			`invalid argument %q for "--theme" flag: must be light, dark, or auto`,
			v,
		)
	}
}

// Type returns the pflag type identifier displayed in --help output.
// Implements pflag.Value.
func (t *ThemeFlag) Type() string {
	return "theme"
}
