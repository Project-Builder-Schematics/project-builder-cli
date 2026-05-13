// Package errors — codes_test.go covers the 6 init-specific ErrCode constants
// introduced in the builder-init-end-to-end change.
//
// REQ coverage: REQ-EC-01, REQ-EC-04
package errors

import "testing"

// Test_InitErrCodes_NamedType_Constants verifies all 6 new init ErrCode
// constants are defined with the correct snake_case string values.
// REQ-EC-01: 6 stable code constants for the init feature.
// REQ-EC-04: constants are additive — existing codes unchanged.
func Test_InitErrCodes_NamedType_Constants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code ErrCode
		want string
	}{
		{
			name: "ErrCodeInitDirNotEmpty",
			code: ErrCodeInitDirNotEmpty,
			want: "init_dir_not_empty",
		},
		{
			name: "ErrCodeInitConfigExists",
			code: ErrCodeInitConfigExists,
			want: "init_config_exists",
		},
		{
			name: "ErrCodeInitAgentFileAmbiguous",
			code: ErrCodeInitAgentFileAmbiguous,
			want: "init_agent_file_ambiguous",
		},
		{
			name: "ErrCodeInitPackageManagerNotFound",
			code: ErrCodeInitPackageManagerNotFound,
			want: "init_package_manager_not_found",
		},
		{
			name: "ErrCodeInitSkillExists",
			code: ErrCodeInitSkillExists,
			want: "init_skill_exists",
		},
		{
			name: "ErrCodeInitNotImplemented",
			code: ErrCodeInitNotImplemented,
			want: "init_not_implemented",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if string(tt.code) != tt.want {
				t.Errorf("ErrCode %s: got %q, want %q", tt.name, string(tt.code), tt.want)
			}
		})
	}
}

// Test_InitErrCodes_AreDistinct verifies none of the 6 new init error codes
// collide with each other or with the 5 existing codes.
// REQ-EC-04: additive only.
func Test_InitErrCodes_AreDistinct(t *testing.T) {
	t.Parallel()

	allCodes := []ErrCode{
		// Existing codes — must remain unchanged.
		ErrCodeNotImplemented,
		ErrCodeCancelled,
		ErrCodeInvalidInput,
		ErrCodeEngineNotFound,
		ErrCodeExecutionFailed,
		// New init codes.
		ErrCodeInitDirNotEmpty,
		ErrCodeInitConfigExists,
		ErrCodeInitAgentFileAmbiguous,
		ErrCodeInitPackageManagerNotFound,
		ErrCodeInitSkillExists,
		ErrCodeInitNotImplemented,
	}

	seen := make(map[ErrCode]bool, len(allCodes))
	for _, c := range allCodes {
		if seen[c] {
			t.Errorf("ErrCode %q is duplicated — must be distinct (REQ-EC-04)", c)
		}
		seen[c] = true
	}
}
