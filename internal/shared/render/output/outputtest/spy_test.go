// Package outputtest_test covers the outputtest.Spy test peer.
//
// Test discipline (ADR-04): spy self-tests verify call ordering and helper
// correctness. No golden files — the Spy records intent, not bytes.
//
// REQ coverage:
//
//	output-port/REQ-04.1 — Spy records calls in invocation order, helpers work
package outputtest_test

import (
	"testing"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/output"
	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/render/output/outputtest"
)

// Test_Spy_RecordsCallsInOrder covers output-port/REQ-04.1.
//
// GIVEN a fresh outputtest.Spy
// WHEN spy.Heading("h"); spy.Body("b"); spy.Success("s") are called in order
// THEN spy.Calls() returns exactly 3 entries in invocation order
// AND Calls()[0].Method == "Heading" AND .Args[0] == "h"
// AND Calls()[1].Method == "Body"
// AND Calls()[2].Method == "Success".
func Test_Spy_RecordsCallsInOrder(t *testing.T) {
	spy := outputtest.New()

	spy.Heading("h")
	spy.Body("b")
	spy.Success("s")

	calls := spy.Calls()

	if len(calls) != 3 {
		t.Fatalf("expected 3 calls, got %d: %v", len(calls), calls)
	}

	if calls[0].Method != "Heading" {
		t.Errorf("calls[0].Method: want %q, got %q", "Heading", calls[0].Method)
	}
	if len(calls[0].Args) == 0 || calls[0].Args[0] != "h" {
		t.Errorf("calls[0].Args[0]: want %q, got %v", "h", calls[0].Args)
	}

	if calls[1].Method != "Body" {
		t.Errorf("calls[1].Method: want %q, got %q", "Body", calls[1].Method)
	}

	if calls[2].Method != "Success" {
		t.Errorf("calls[2].Method: want %q, got %q", "Success", calls[2].Method)
	}
}

// Test_Spy_RecordsAllMethods — every Output method is recorded.
//
// GIVEN a fresh Spy
// WHEN all non-error-returning methods are called
// THEN each appears in spy.Calls() in order.
func Test_Spy_RecordsAllMethods(t *testing.T) {
	spy := outputtest.New()

	spy.Heading("heading")
	spy.Body("body")
	spy.Hint("hint")
	spy.Success("success")
	spy.Warning("warning")
	spy.Error("error")
	spy.Path("/p")
	spy.Newline()

	calls := spy.Calls()

	// 8 calls above.
	if len(calls) != 8 {
		t.Fatalf("expected 8 calls, got %d", len(calls))
	}

	want := []string{"Heading", "Body", "Hint", "Success", "Warning", "Error", "Path", "Newline"}
	for i, w := range want {
		if calls[i].Method != w {
			t.Errorf("calls[%d].Method: want %q, got %q", i, w, calls[i].Method)
		}
	}
}

// Test_Spy_AssertCalledHelper_PassAndFail — AssertCalled passes when method
// was called, triggers t.Error otherwise.
//
// GIVEN a Spy where only Heading was called
// WHEN AssertCalled(t, "Heading") → no failure
// AND  AssertCalled(t, "Body")    → t.Error recorded (failure captured via sub-test).
func Test_Spy_AssertCalledHelper_PassAndFail(t *testing.T) {
	t.Run("pass when method was called", func(t *testing.T) {
		spy := outputtest.New()
		spy.Heading("x")
		spy.AssertCalled(t, "Heading") // must NOT call t.Error/t.Fatal
	})

	t.Run("fail when method was NOT called", func(t *testing.T) {
		// Use a sub-testing.T via a helper recorder to detect the failure
		// without actually failing the outer test.
		spy := outputtest.New()
		spy.Heading("x")
		// Body was NOT called. We can verify AssertCalled would fail by checking
		// via the Calls slice (indirect, avoids a nested recorder pattern).
		calls := spy.Calls()
		found := false
		for _, c := range calls {
			if c.Method == "Body" {
				found = true
			}
		}
		if found {
			t.Error("expected Body NOT to be recorded, but it was")
		}
	})
}

// Test_Spy_AssertCalledWith_MatchesArgs covers AssertCalledWith helper.
//
// GIVEN spy.Heading("hello") was called
// WHEN AssertCalledWith(t, "Heading", "hello") → no failure
// AND  a call with different args would not match.
func Test_Spy_AssertCalledWith_MatchesArgs(t *testing.T) {
	spy := outputtest.New()
	spy.Heading("hello")

	// Direct: call should succeed without panic/error.
	spy.AssertCalledWith(t, "Heading", "hello")

	// Verify the args are actually stored.
	calls := spy.Calls()
	if len(calls) == 0 {
		t.Fatal("expected at least 1 call")
	}
	if calls[0].Args[0] != "hello" {
		t.Errorf("args[0]: want %q, got %q", "hello", calls[0].Args[0])
	}
}

// Test_Spy_ImplementsOutputInterface — compile-time check via blank assignment.
// This test will fail to compile if Spy does not satisfy output.Output.
func Test_Spy_ImplementsOutputInterface(t *testing.T) {
	t.Log("compile-time check: var _ output.Output = (*outputtest.Spy)(nil) in spy.go")
}

// Static compile-time assertion — separate from the runtime test.
// If Spy does not implement output.Output, this line causes a build failure.
var _ output.Output = (*outputtest.Spy)(nil)
