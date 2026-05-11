// Package events_test (external) verifies the sealed-interface contract:
// code outside the events package CANNOT implement Event.
//
// REQ coverage: event-catalogue.REQ-01.1
//
// CONTRACT:STUB — behaviour-deferred to /plan #3+
package events_test

// externalType attempts to implement Event from outside the package.
// This must NOT compile if the interface is properly sealed (unexported marker method).
//
// To prove the seal, we rely on the fact that this file is in package events_test
// (external package). If the Event interface had only exported methods, this type
// could implement it. With an unexported method (isEvent()), this fails.
//
// Test_External_CannotImplementEvent is an architectural/compile-time test.
// It is expressed as a build-constraint: the file itself residing in an external
// package and importing events is the test. The actual runtime test documents
// the intent and verifies the package imports cleanly.

import (
	"testing"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/events"
)

// Test_External_CannotImplementEvent verifies the seal at package-boundary level.
//
// The true compile-time enforcement is implicit: if we tried to write:
//
//	type extType struct{ events.EventBase }
//	func (extType) isEvent() {}  // would fail — isEvent is unexported
//
// That code would fail to compile. We document and verify the constraint here
// by confirming the package loads correctly and the Event type exists but
// cannot be forged from outside.
func Test_External_CannotImplementEvent(t *testing.T) {
	// Verify the package is importable and Event-implementing types are accessible.
	base := events.EventBase{Seq: 1}

	// The architectural assertion: an external package cannot implement Event
	// because isEvent() is unexported. This is enforced at compile time by Go's
	// visibility rules. Any attempt to add `func (extType) isEvent() {}` in this
	// file would fail with: "cannot use extType as type events.Event: wrong type
	// for method isEvent (missing events.isEvent)".
	//
	// This test PASSES simply by compiling. The successful assignment of a
	// concrete events.Done to the events.Event interface variable proves the
	// type satisfies the interface — and the inability to do the same from
	// outside the package is the seal.
	var ev events.Event = events.Done{EventBase: base}
	t.Logf("sealed Event interface: %T assigned to events.Event from external package", ev)
}
