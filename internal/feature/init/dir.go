// Package initialise — dir.go contains directory canonicalisation and
// traversal-rejection helpers used by handler.RunE.
//
// REQ-DV-01: target directory MUST be absolute + cleaned before use.
// REQ-DV-02: relative paths with ".." that escape cwd MUST be rejected.
package initialise

import (
	"os"
	"path/filepath"
	"strings"

	errs "github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/errors"
)

// canonicaliseDir applies filepath.Abs + filepath.Clean to rawDir and
// rejects paths that resolve outside the current working directory via
// .. segments (REQ-DV-01, REQ-DV-02).
//
// Sibling directories (e.g. ../sibling) that are outside cwd are rejected
// because they could point to sensitive system directories.
func canonicaliseDir(rawDir string) (string, error) {
	abs, err := filepath.Abs(rawDir)
	if err != nil {
		return "", &errs.Error{
			Code:    errs.ErrCodeInvalidInput,
			Op:      "init.handler",
			Message: "could not resolve directory path",
			Cause:   err,
		}
	}
	clean := filepath.Clean(abs)

	// Reject paths outside cwd when the raw input contains ".." segments.
	// A clean absolute path that is genuinely inside cwd has cwd as a prefix.
	if strings.Contains(rawDir, "..") {
		cwd, cwdErr := os.Getwd()
		if cwdErr == nil {
			cwdClean := filepath.Clean(cwd)
			if !strings.HasPrefix(clean, cwdClean) {
				return "", &errs.Error{
					Code:    errs.ErrCodeInvalidInput,
					Op:      "init.handler",
					Message: "directory path resolves outside the current working directory (.. traversal rejected)",
					Suggestions: []string{
						"use an absolute path instead of a relative path with ..",
						"use a path relative to the current working directory without traversal",
					},
				}
			}
		}
	}

	return clean, nil
}
