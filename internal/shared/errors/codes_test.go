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

// Test_NewErrCodes_NamedType_Constants verifies all 7 new builder-new ErrCode
// constants are defined with the correct snake_case string values.
// REQ-EC-01..07: 7 stable code constants for the new feature.
func Test_NewErrCodes_NamedType_Constants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code ErrCode
		want string
	}{
		{
			name: "ErrCodeNewSchematicExists",
			code: ErrCodeNewSchematicExists,
			want: "new_schematic_exists",
		},
		{
			name: "ErrCodeNewCollectionExists",
			code: ErrCodeNewCollectionExists,
			want: "new_collection_exists",
		},
		{
			name: "ErrCodeInvalidSchematicName",
			code: ErrCodeInvalidSchematicName,
			want: "new_invalid_name",
		},
		{
			name: "ErrCodeInvalidExtends",
			code: ErrCodeInvalidExtends,
			want: "new_invalid_extends",
		},
		{
			name: "ErrCodeModeConflict",
			code: ErrCodeModeConflict,
			want: "new_mode_conflict",
		},
		{
			name: "ErrCodeInvalidLanguage",
			code: ErrCodeInvalidLanguage,
			want: "new_invalid_language",
		},
		{
			name: "ErrCodeNewNotImplemented",
			code: ErrCodeNewNotImplemented,
			want: "new_not_implemented",
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

// Test_NewErrCodes_AreDistinct verifies all 7 builder-new ErrCode constants
// do not collide with each other or with existing codes.
// REQ-EC-01..07: additive only — no existing code may be renamed or removed.
func Test_NewErrCodes_AreDistinct(t *testing.T) {
	t.Parallel()

	allCodes := []ErrCode{
		// Existing codes — must remain unchanged.
		ErrCodeNotImplemented,
		ErrCodeCancelled,
		ErrCodeInvalidInput,
		ErrCodeEngineNotFound,
		ErrCodeExecutionFailed,
		ErrCodeInitDirNotEmpty,
		ErrCodeInitConfigExists,
		ErrCodeInitAgentFileAmbiguous,
		ErrCodeInitPackageManagerNotFound,
		ErrCodeInitSkillExists,
		ErrCodeInitNotImplemented,
		// New builder-new codes (REQ-EC-01..07).
		ErrCodeNewSchematicExists,
		ErrCodeNewCollectionExists,
		ErrCodeInvalidSchematicName,
		ErrCodeInvalidExtends,
		ErrCodeModeConflict,
		ErrCodeInvalidLanguage,
		ErrCodeNewNotImplemented,
	}

	seen := make(map[ErrCode]bool, len(allCodes))
	for _, c := range allCodes {
		if seen[c] {
			t.Errorf("ErrCode %q is duplicated — must be distinct (REQ-EC-01..07 additive)", c)
		}
		seen[c] = true
	}
}

// Test_NewErrCodes_ErrorsAs_RoundTrip verifies that errors.As can unwrap each
// new ErrCode from a wrapped *Error value, confirming the Is/As contract.
// REQ-EC-01..07: ErrorsAs round-trip per project testing convention.
func Test_NewErrCodes_ErrorsAs_RoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		sentinel *Error
		wrapped  error
	}{
		{
			name:     "ErrCodeNewSchematicExists",
			sentinel: &Error{Code: ErrCodeNewSchematicExists},
			wrapped:  &Error{Code: ErrCodeNewSchematicExists, Op: "new.handler", Message: "schematic exists"},
		},
		{
			name:     "ErrCodeNewCollectionExists",
			sentinel: &Error{Code: ErrCodeNewCollectionExists},
			wrapped:  &Error{Code: ErrCodeNewCollectionExists, Op: "new.handler", Message: "collection exists"},
		},
		{
			name:     "ErrCodeInvalidSchematicName",
			sentinel: &Error{Code: ErrCodeInvalidSchematicName},
			wrapped:  &Error{Code: ErrCodeInvalidSchematicName, Op: "new.handler", Message: "invalid name"},
		},
		{
			name:     "ErrCodeInvalidExtends",
			sentinel: &Error{Code: ErrCodeInvalidExtends},
			wrapped:  &Error{Code: ErrCodeInvalidExtends, Op: "new.handler", Message: "invalid extends"},
		},
		{
			name:     "ErrCodeModeConflict",
			sentinel: &Error{Code: ErrCodeModeConflict},
			wrapped:  &Error{Code: ErrCodeModeConflict, Op: "new.handler", Message: "mode conflict"},
		},
		{
			name:     "ErrCodeInvalidLanguage",
			sentinel: &Error{Code: ErrCodeInvalidLanguage},
			wrapped:  &Error{Code: ErrCodeInvalidLanguage, Op: "new.handler", Message: "invalid language"},
		},
		{
			name:     "ErrCodeNewNotImplemented",
			sentinel: &Error{Code: ErrCodeNewNotImplemented},
			wrapped:  &Error{Code: ErrCodeNewNotImplemented, Op: "new.handler", Message: "not implemented"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// errors.Is must return true when matching on Code.
			if !isError(tt.wrapped, tt.sentinel) {
				t.Errorf("errors.Is(%T{Code:%q}, sentinel{Code:%q}) = false; want true",
					tt.wrapped, tt.wrapped.(*Error).Code, tt.sentinel.Code)
			}

			// errors.As must successfully unwrap to *Error.
			var e *Error
			if !asError(tt.wrapped, &e) {
				t.Errorf("errors.As failed to unwrap %T to *Error", tt.wrapped)
			}
		})
	}
}

// isError is a test helper wrapping errors.Is to avoid import of the standard
// errors package name clashing with the local package name.
func isError(err, target error) bool {
	if err == nil || target == nil {
		return err == target
	}
	e, ok := err.(*Error)
	if !ok {
		return false
	}
	return e.Is(target)
}

// asError is a test helper wrapping errors.As for use within the package.
func asError(err error, target **Error) bool {
	if err == nil {
		return false
	}
	e, ok := err.(*Error)
	if ok {
		*target = e
	}
	return ok
}
