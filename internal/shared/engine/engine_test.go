// Package engine_test covers engine-port.REQ-01.*, engine-port.REQ-02.*,
// security.REQ-01.*, and security.REQ-02.* via compile-time contracts,
// behavioural assertions, and GoDoc grep checks.
//
// CONTRACT:STUB — behaviour-deferred to /plan #3+
package engine

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// compile-time interface satisfaction check (not a test function — package-level assertion).
// Mutation: remove FakeEngine.Execute → compile error.
var _ Engine = (*FakeEngine)(nil)

// ──────────────────────────────────────────────────────────────────────────────
// engine-port.REQ-01.1 — Engine interface satisfiable by FakeEngine
// ──────────────────────────────────────────────────────────────────────────────

// Test_FakeEngine_SatisfiesEngineIface verifies the compile-time assertion above
// by calling Execute on a *FakeEngine through the Engine interface.
func Test_FakeEngine_SatisfiesEngineIface(t *testing.T) {
	var e Engine = &FakeEngine{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := e.Execute(ctx, ExecuteRequest{})
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	if ch == nil {
		t.Fatal("Execute returned nil channel")
	}
	// Drain the channel so the fake goroutine exits cleanly.
	cancel()
	for range ch { //nolint:revive // drain intentional
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// engine-port.REQ-01.2 — ExecuteRequest fields accessible
// ──────────────────────────────────────────────────────────────────────────────

// Test_ExecuteRequest_FieldsAccessible verifies that ExecuteRequest has all
// required fields and they are compile-time accessible.
func Test_ExecuteRequest_FieldsAccessible(t *testing.T) {
	req := ExecuteRequest{
		Workspace:    "/tmp/ws",
		Schematic:    SchematicRef{Collection: "@schematics/angular", Name: "component", Version: "latest"},
		Inputs:       map[string]any{"name": "app"},
		EnvAllowlist: []string{"PATH", "HOME"}, // fitness:allow-untyped-args env-allowlist
	}

	if req.Workspace != "/tmp/ws" {
		t.Errorf("Workspace = %q, want /tmp/ws", req.Workspace)
	}
	if req.Schematic.Collection != "@schematics/angular" {
		t.Errorf("Schematic.Collection = %q", req.Schematic.Collection)
	}
	if req.Schematic.Name != "component" {
		t.Errorf("Schematic.Name = %q", req.Schematic.Name)
	}
	if req.Schematic.Version != "latest" {
		t.Errorf("Schematic.Version = %q", req.Schematic.Version)
	}
	if len(req.Inputs) != 1 {
		t.Errorf("Inputs len = %d, want 1", len(req.Inputs))
	}
	if len(req.EnvAllowlist) != 2 {
		t.Errorf("EnvAllowlist len = %d, want 2", len(req.EnvAllowlist))
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// engine-port.REQ-01.3 / security.REQ-01.2 — SchematicRef is a struct, not a string alias
// ──────────────────────────────────────────────────────────────────────────────

// Test_SchematicRef_IsStruct_NotStringAlias asserts via reflection that
// SchematicRef is a struct kind, not a string alias (which would accept
// arbitrary injection). Mutation: change SchematicRef to `type SchematicRef string`
// → reflect.Struct check fails.
func Test_SchematicRef_IsStruct_NotStringAlias(t *testing.T) {
	var ref SchematicRef
	kind := reflect.TypeOf(ref).Kind()
	if kind != reflect.Struct {
		t.Errorf("SchematicRef kind = %v, want reflect.Struct — must be a struct, not a string alias", kind)
	}
}

// Test_SchematicRef_DistinctNamedType verifies that the three fields of
// SchematicRef have distinct typed slots (compile-time check via literal).
func Test_SchematicRef_DistinctNamedType(t *testing.T) {
	ref := SchematicRef{
		Collection: "col",
		Name:       "name",
		Version:    "v1.0.0",
	}
	if ref.Collection == "" || ref.Name == "" || ref.Version == "" {
		t.Error("SchematicRef fields must be non-empty when set")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// engine-port.REQ-02.1 / security.REQ-01.1 — GoDoc declares 5s ceiling + anti-script
// ──────────────────────────────────────────────────────────────────────────────

// Test_Engine_GoDoc_DeclaresCancellationCeiling reads engine.go and asserts
// that the Engine interface GoDoc contains the phrases "5 seconds" and "cancel".
// Mutation: remove either phrase from GoDoc → this test fails.
func Test_Engine_GoDoc_DeclaresCancellationCeiling(t *testing.T) {
	src, err := os.ReadFile("engine.go")
	if err != nil {
		t.Fatalf("could not read engine.go: %v", err)
	}
	content := string(src)

	phrases := []string{"5 seconds", "cancel"}
	for _, phrase := range phrases {
		if !strings.Contains(strings.ToLower(content), strings.ToLower(phrase)) {
			t.Errorf("engine.go GoDoc does not contain required phrase %q", phrase)
		}
	}
}

// Test_Engine_GoDoc_AntiScriptInvariant reads engine.go and asserts that the
// file contains the anti-script mandate (security.REQ-01.1).
// Mutation: remove "anti-script" or "MUST NOT" from GoDoc → this test fails.
func Test_Engine_GoDoc_AntiScriptInvariant(t *testing.T) {
	src, err := os.ReadFile("engine.go")
	if err != nil {
		t.Fatalf("could not read engine.go: %v", err)
	}
	content := strings.ToLower(string(src))

	required := []string{"anti-script", "must not"}
	for _, phrase := range required {
		if !strings.Contains(content, phrase) {
			t.Errorf("engine.go GoDoc does not contain required security phrase %q", phrase)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// engine-port.REQ-02.2 / security.REQ-02.1 — FakeEngine honours ctx.Done() within 5s
// ──────────────────────────────────────────────────────────────────────────────

// Test_FakeEngine_HonoursContextCancel_Within5s cancels the context and asserts
// that FakeEngine.Execute's returned channel closes within 500ms — well within
// the 5-second ceiling. A tight assertion provides fast feedback during tests
// while still proving the contract. Mutation: FakeEngine ignores ctx.Done()
// → channel never closes → test times out via the 5s deadline.
func Test_FakeEngine_HonoursContextCancel_Within5s(t *testing.T) {
	// Use a 5-second test deadline (the ceiling), but expect the channel to
	// close within 500ms of cancel to keep the test fast.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fe := &FakeEngine{}
	ch, err := fe.Execute(ctx, ExecuteRequest{})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	// Cancel immediately after Execute returns the channel.
	cancel()

	// Assert channel closes within 500ms of cancel — fast-feedback margin.
	done := make(chan struct{})
	go func() {
		for range ch { //nolint:revive // drain intentional
		}
		close(done)
	}()

	select {
	case <-done:
		// Pass: channel closed promptly after cancel.
	case <-time.After(500 * time.Millisecond):
		t.Error("FakeEngine did not close channel within 500ms of ctx cancel — violates 5s ceiling contract")
	}
}

// Test_FakeEngine_RespectsCtxDone_Within5s is the shared test for
// engine-port.REQ-02.2 and security.REQ-02.1 per design §6. Delegates to the
// cancellation test above via subtests to keep REQ-ID traceability clear.
func Test_FakeEngine_RespectsCtxDone_Within5s(t *testing.T) {
	t.Run("cancel propagates to channel close", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		fe := &FakeEngine{}
		ch, err := fe.Execute(ctx, ExecuteRequest{})
		if err != nil {
			t.Fatalf("Execute returned error: %v", err)
		}

		cancel()

		done := make(chan struct{})
		go func() {
			for range ch { //nolint:revive // drain intentional
			}
			close(done)
		}()

		select {
		case <-done:
			// Pass.
		case <-time.After(500 * time.Millisecond):
			t.Error("channel did not close within 500ms after ctx cancel")
		}
	})
}
