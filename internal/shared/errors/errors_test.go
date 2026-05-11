// Package errors_test covers the structured error contracts.
//
// REQ coverage: structured-error.REQ-01.1, .01.2, .01.3, .02.1, .02.2, .02.3,
// .03.1, .03.2, .03.3; security.REQ-04.1, .04.2
package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"testing"
)

// Test_Error_FieldsTypedAndAccessible verifies all expected fields exist.
// structured-error.REQ-01.1
func Test_Error_FieldsTypedAndAccessible(t *testing.T) {
	cause := fmt.Errorf("upstream")
	e := &Error{
		Code:        ErrCodeNotImplemented,
		Op:          "init.handler",
		Path:        "/some/path",
		Message:     "not implemented",
		Details:     []string{"detail 1"},
		Suggestions: []string{"try X"},
		Cause:       cause,
	}

	if e.Code != ErrCodeNotImplemented {
		t.Errorf("Code: got %q, want %q", e.Code, ErrCodeNotImplemented)
	}

	if e.Op != "init.handler" {
		t.Errorf("Op: got %q, want %q", e.Op, "init.handler")
	}

	if e.Path != "/some/path" {
		t.Errorf("Path: got %q, want %q", e.Path, "/some/path")
	}

	if e.Message != "not implemented" {
		t.Errorf("Message: got %q, want %q", e.Message, "not implemented")
	}

	if len(e.Details) != 1 || e.Details[0] != "detail 1" {
		t.Errorf("Details: got %v", e.Details)
	}

	if len(e.Suggestions) != 1 || e.Suggestions[0] != "try X" {
		t.Errorf("Suggestions: got %v", e.Suggestions)
	}

	if e.Cause != cause {
		t.Errorf("Cause: got %v, want %v", e.Cause, cause)
	}
}

// Test_ErrCode_NamedType_Constants verifies ErrCode is a named type and all
// 5 constants exist with correct string values.
// structured-error.REQ-01.2
func Test_ErrCode_NamedType_Constants(t *testing.T) {
	tests := []struct {
		code ErrCode
		want string
	}{
		{ErrCodeNotImplemented, "not_implemented"},
		{ErrCodeCancelled, "cancelled"},
		{ErrCodeInvalidInput, "invalid_input"},
		{ErrCodeEngineNotFound, "engine_not_found"},
		{ErrCodeExecutionFailed, "execution_failed"},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			if string(tt.code) != tt.want {
				t.Errorf("ErrCode constant: got %q, want %q", string(tt.code), tt.want)
			}
		})
	}
}

// Test_Op_FormatRegex_AllHandlers verifies that Op values follow the
// `^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$` format invariant.
// structured-error.REQ-01.3
func Test_Op_FormatRegex_AllHandlers(t *testing.T) {
	re := regexp.MustCompile(OpRegex)

	// Op values from all 8 handler stubs + engine. These will be verified
	// against the actual handlers in the smoke test in cmd/builder; here
	// we validate the regex rejects invalid patterns.
	validOps := []string{
		"init.handler",
		"execute.handler",
		"add.handler",
		"info.handler",
		"sync.handler",
		"validate.handler",
		"remove.handler",
		"skill_update.handler",
		"engine.execute",
	}

	invalidOps := []string{
		"Init.handler",     // uppercase first char
		"init.Handler",     // uppercase after dot
		"init",             // no dot
		"init.",            // trailing dot only
		".handler",         // leading dot
		"INIT.HANDLER",     // all uppercase
		"init.handler.sub", // three segments
		"",                 // empty
	}

	for _, op := range validOps {
		t.Run("valid/"+op, func(t *testing.T) {
			if !re.MatchString(op) {
				t.Errorf("Op %q should match regex %q", op, OpRegex)
			}
		})
	}

	for _, op := range invalidOps {
		t.Run("invalid/"+op, func(t *testing.T) {
			if re.MatchString(op) {
				t.Errorf("Op %q should NOT match regex %q", op, OpRegex)
			}
		})
	}
}

// Test_ErrorsIs_MatchesByCode_ThroughFmtErrorf verifies errors.Is matches by
// Code even when wrapped via fmt.Errorf("%w").
// structured-error.REQ-02.1
func Test_ErrorsIs_MatchesByCode_ThroughFmtErrorf(t *testing.T) {
	original := &Error{
		Code:    ErrCodeNotImplemented,
		Op:      "init.handler",
		Message: "not implemented",
	}

	wrapped := fmt.Errorf("outer: %w", original)

	sentinel := &Error{Code: ErrCodeNotImplemented}
	if !errors.Is(wrapped, sentinel) {
		t.Errorf("errors.Is should return true for matching Code through fmt.Errorf wrapper")
	}

	// Different code should NOT match.
	other := &Error{Code: ErrCodeCancelled}
	if errors.Is(wrapped, other) {
		t.Errorf("errors.Is should return false for different Code")
	}
}

// Test_ErrorsAs_UnwrapsToError verifies errors.As can extract *Error.
// structured-error.REQ-02.2
func Test_ErrorsAs_UnwrapsToError(t *testing.T) {
	original := &Error{
		Code:    ErrCodeInvalidInput,
		Op:      "validate.handler",
		Message: "invalid input",
	}

	wrapped := fmt.Errorf("wrap: %w", original)

	var target *Error
	if !errors.As(wrapped, &target) {
		t.Fatal("errors.As should extract *Error from wrapped error")
	}

	if target.Code != ErrCodeInvalidInput {
		t.Errorf("extracted Code: got %q, want %q", target.Code, ErrCodeInvalidInput)
	}
}

// Test_ErrorsIs_TraversalReachesInnerCause verifies errors.Is traverses
// multiple wrapping layers to find a matching *Error.
// structured-error.REQ-02.3
func Test_ErrorsIs_TraversalReachesInnerCause(t *testing.T) {
	inner := &Error{
		Code:    ErrCodeExecutionFailed,
		Op:      "engine.execute",
		Message: "execution failed",
	}
	middle := fmt.Errorf("middle: %w", inner)
	outer := fmt.Errorf("outer: %w", middle)

	sentinel := &Error{Code: ErrCodeExecutionFailed}
	if !errors.Is(outer, sentinel) {
		t.Error("errors.Is must traverse multi-level wrapping to find *Error by Code")
	}
}

// Test_MarshalJSON_OmitsCauseDetailsPath verifies MarshalJSON outputs only
// Code, Op, Message, Suggestions — and omits Cause, Details, Path.
// structured-error.REQ-03.1 / security.REQ-04.1
func Test_MarshalJSON_OmitsCauseDetailsPath(t *testing.T) {
	e := &Error{
		Code:        ErrCodeNotImplemented,
		Op:          "init.handler",
		Path:        "/secret/path",
		Message:     "not implemented",
		Details:     []string{"internal detail"},
		Suggestions: []string{"try something"},
		Cause:       fmt.Errorf("upstream secret error"),
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}

	jsonStr := string(data)

	// Must include allowed fields.
	if !contains(jsonStr, `"not_implemented"`) {
		t.Errorf("JSON should contain Code value; got: %s", jsonStr)
	}

	if !contains(jsonStr, `"init.handler"`) {
		t.Errorf("JSON should contain Op value; got: %s", jsonStr)
	}

	if !contains(jsonStr, `"not implemented"`) {
		t.Errorf("JSON should contain Message value; got: %s", jsonStr)
	}

	if !contains(jsonStr, `"try something"`) {
		t.Errorf("JSON should contain Suggestions value; got: %s", jsonStr)
	}

	// Must NOT include security-sensitive fields.
	if contains(jsonStr, `/secret/path`) {
		t.Errorf("JSON must NOT contain Path value; got: %s", jsonStr)
	}

	if contains(jsonStr, `internal detail`) {
		t.Errorf("JSON must NOT contain Details value; got: %s", jsonStr)
	}

	if contains(jsonStr, `upstream secret error`) {
		t.Errorf("JSON must NOT contain Cause text; got: %s", jsonStr)
	}

	// Also verify no stray field keys for the omitted fields.
	if contains(jsonStr, `"cause"`) || contains(jsonStr, `"Cause"`) {
		t.Errorf("JSON must NOT contain Cause key; got: %s", jsonStr)
	}

	if contains(jsonStr, `"details"`) || contains(jsonStr, `"Details"`) {
		t.Errorf("JSON must NOT contain Details key; got: %s", jsonStr)
	}

	if contains(jsonStr, `"path"`) || contains(jsonStr, `"Path"`) {
		t.Errorf("JSON must NOT contain Path key; got: %s", jsonStr)
	}
}

// Test_SafeMessage_OmitsPathAndCauseAndDetails verifies SafeMessage() returns
// only "Code: Message" without leaking Path, Cause, or Details.
// structured-error.REQ-03.2 / security.REQ-04.2
func Test_SafeMessage_OmitsPathAndCauseAndDetails(t *testing.T) {
	e := &Error{
		Code:    ErrCodeEngineNotFound,
		Op:      "execute.handler",
		Path:    "/private/path",
		Message: "engine not found",
		Details: []string{"detail leak"},
		Cause:   fmt.Errorf("cause leak"),
	}

	safe := e.SafeMessage()

	if contains(safe, "/private/path") {
		t.Errorf("SafeMessage must NOT leak Path; got: %q", safe)
	}

	if contains(safe, "detail leak") {
		t.Errorf("SafeMessage must NOT leak Details; got: %q", safe)
	}

	if contains(safe, "cause leak") {
		t.Errorf("SafeMessage must NOT leak Cause; got: %q", safe)
	}

	// Should still contain the safe parts.
	expected := string(ErrCodeEngineNotFound) + ": " + "engine not found"
	if safe != expected {
		t.Errorf("SafeMessage: got %q, want %q", safe, expected)
	}
}

// Test_Error_ReturnsCodeColonMessage_Exact verifies Error() returns EXACTLY
// "<ErrCode>: <Message>" with no additional content.
// structured-error.REQ-03.3
func Test_Error_ReturnsCodeColonMessage_Exact(t *testing.T) {
	tests := []struct {
		code    ErrCode
		message string
		want    string
	}{
		{ErrCodeNotImplemented, "not implemented", "not_implemented: not implemented"},
		{ErrCodeCancelled, "operation cancelled", "cancelled: operation cancelled"},
		{ErrCodeInvalidInput, "missing field", "invalid_input: missing field"},
		{ErrCodeEngineNotFound, "no engine configured", "engine_not_found: no engine configured"},
		{ErrCodeExecutionFailed, "schematic failed", "execution_failed: schematic failed"},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			e := &Error{
				Code:    tt.code,
				Op:      "init.handler",
				Path:    "/should/not/appear",
				Message: tt.message,
				Cause:   fmt.Errorf("should not appear"),
				Details: []string{"should not appear"},
			}

			got := e.Error()
			if got != tt.want {
				t.Errorf("Error(): got %q, want %q", got, tt.want)
			}
		})
	}
}

// contains is a simple substring check helper.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && indexStr(s, substr) >= 0)
}

// indexStr returns the index of substr in s, or -1 if not found.
func indexStr(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}

	if len(substr) > len(s) {
		return -1
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}

	return -1
}
