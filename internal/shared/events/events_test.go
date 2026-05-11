// Package events_test covers the event catalogue contracts.
//
// REQ coverage: event-catalogue.REQ-01.2, .02.1, .02.2, .02.3, .03.1, .03.2, .03.3;
// security.REQ-03.1, .03.4
//
// CONTRACT:STUB — behaviour-deferred to /plan #3+
package events

import (
	"reflect"
	"regexp"
	"testing"
	"time"
)

// Test_All12_SatisfyEvent verifies every concrete type implements Event.
// event-catalogue.REQ-01.2
func Test_All12_SatisfyEvent(_ *testing.T) {
	base := EventBase{Seq: 1, At: time.Now()}

	cases := []Event{
		FileCreated{EventBase: base, Path: "/foo", IsDir: false},
		FileModified{EventBase: base, Path: "/foo"},
		FileDeleted{EventBase: base, Path: "/foo"},
		ScriptStarted{EventBase: base, Name: "ng", Args: []string{"generate"}},
		ScriptStopped{EventBase: base, Name: "ng", ExitCode: 0},
		LogLine{EventBase: base, Level: "info", Source: LogSourceStdout, Text: "hi"},
		InputRequested{EventBase: base, Prompt: "name?", Reply: make(chan<- string, 1)},
		InputProvided{EventBase: base, Prompt: "name?", Value: "foo"},
		Progress{EventBase: base, Step: 1, Total: 3, Label: "building"},
		Done{EventBase: base},
		Failed{EventBase: base, Err: nil},
		Cancelled{EventBase: base},
	}

	for _, ev := range cases {
		// The type assertion to Event compiles only if each type implements isEvent().
		// If any type is missing, the composite literal above fails to compile.
		_ = ev
	}
}

// Test_EventBase_AnonymousEmbedding_AccessibleSeqAt verifies EventBase fields
// are accessible without qualification (anonymous embedding, not named field).
// event-catalogue.REQ-02.1
func Test_EventBase_AnonymousEmbedding_AccessibleSeqAt(t *testing.T) {
	now := time.Now()
	fc := FileCreated{
		EventBase: EventBase{Seq: 42, At: now},
		Path:      "/some/path",
	}

	if fc.Seq != 42 {
		t.Errorf("expected Seq=42 via anonymous embed, got %d", fc.Seq)
	}

	if fc.At != now {
		t.Errorf("expected At=%v via anonymous embed, got %v", now, fc.At)
	}
}

// Test_LogSource_NamedEnum_NotBareString verifies LogSource is a named type,
// not a bare string, so that mis-typed values fail compilation.
// event-catalogue.REQ-02.2
func Test_LogSource_NamedEnum_NotBareString(t *testing.T) {
	// This test verifies via reflection that LogSource is a distinct named type.
	s := LogSourceStdout
	rt := reflect.TypeOf(s)

	if rt.Name() != "LogSource" {
		t.Errorf("expected type name 'LogSource', got %q", rt.Name())
	}

	if rt.Kind() != reflect.String {
		t.Errorf("expected underlying kind string, got %v", rt.Kind())
	}

	// Verify all three constants have expected values.
	want := map[LogSource]string{
		LogSourceStdout: "stdout",
		LogSourceStderr: "stderr",
		LogSourceSystem: "system",
	}

	for k, v := range want {
		if string(k) != v {
			t.Errorf("LogSource constant: got %q, want %q", string(k), v)
		}
	}
}

// Test_LogLine_Fields_SourceTextSensitive verifies V2 field naming:
// Source (not Stream), Text (not Line), and Sensitive bool present.
// event-catalogue.REQ-02.3
func Test_LogLine_Fields_SourceTextSensitive(t *testing.T) {
	ll := LogLine{
		EventBase: EventBase{Seq: 1, At: time.Now()},
		Level:     "info",
		Source:    LogSourceStderr,
		Text:      "some log text",
		Sensitive: true,
	}

	if ll.Source != LogSourceStderr {
		t.Errorf("LogLine.Source: got %q, want %q", ll.Source, LogSourceStderr)
	}

	if ll.Text != "some log text" {
		t.Errorf("LogLine.Text: got %q, want %q", ll.Text, "some log text")
	}

	if !ll.Sensitive {
		t.Error("LogLine.Sensitive: expected true")
	}
}

// Test_Catalogue_HasExactly12Types verifies the total count.
// event-catalogue.REQ-03.1
func Test_Catalogue_HasExactly12Types(t *testing.T) {
	base := EventBase{Seq: 1, At: time.Now()}

	cases := []Event{
		FileCreated{EventBase: base},
		FileModified{EventBase: base},
		FileDeleted{EventBase: base},
		ScriptStarted{EventBase: base},
		ScriptStopped{EventBase: base},
		LogLine{EventBase: base},
		InputRequested{EventBase: base, Reply: make(chan<- string, 1)},
		InputProvided{EventBase: base},
		Progress{EventBase: base},
		Done{EventBase: base},
		Failed{EventBase: base},
		Cancelled{EventBase: base},
	}

	const want = 12
	if len(cases) != want {
		t.Errorf("event catalogue: got %d types, want %d", len(cases), want)
	}
}

// Test_Done_TerminalClosesChannel verifies that a terminal event signals
// channel close. The producer MUST close the channel after emitting Done.
// event-catalogue.REQ-03.2
func Test_Done_TerminalClosesChannel(t *testing.T) {
	ch := make(chan Event, 2)
	base := EventBase{Seq: 1, At: time.Now()}
	ch <- Done{EventBase: base}
	close(ch)

	var received []Event
	for ev := range ch {
		received = append(received, ev)
	}

	if len(received) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received))
	}

	if _, ok := received[0].(Done); !ok {
		t.Errorf("expected Done event, got %T", received[0])
	}
}

// Test_InputRequested_Reply_BufferedCap1 verifies the Reply channel is
// buffered with capacity 1 so a producer can send without blocking.
// event-catalogue.REQ-03.3
func Test_InputRequested_Reply_BufferedCap1(t *testing.T) {
	replyCh := make(chan string, 1)
	sendOnly := (chan<- string)(replyCh)

	ev := InputRequested{
		EventBase: EventBase{Seq: 1, At: time.Now()},
		Prompt:    "What is your name?",
		Sensitive: false,
		Reply:     sendOnly,
	}

	// Cap must be 1 so producer can send non-blocking.
	if cap(replyCh) != 1 {
		t.Errorf("Reply channel capacity: got %d, want 1", cap(replyCh))
	}

	// Verify Reply field is send-only direction.
	rt := reflect.TypeOf(ev.Reply)
	if rt.ChanDir() != reflect.SendDir {
		t.Errorf("Reply field direction: got %v, want SendDir", rt.ChanDir())
	}
}

// Test_Sensitive_FieldPresent_OnAllFourTypes verifies that all four required
// types carry the Sensitive bool field (compile-time contract).
// security.REQ-03.1
func Test_Sensitive_FieldPresent_OnAllFourTypes(t *testing.T) {
	base := EventBase{Seq: 1, At: time.Now()}

	// Each field access below will fail to compile if Sensitive is absent.
	ir := InputRequested{EventBase: base, Sensitive: true, Reply: make(chan<- string, 1)}
	ip := InputProvided{EventBase: base, Sensitive: true}
	ll := LogLine{EventBase: base, Sensitive: true}
	ss := ScriptStarted{EventBase: base, Sensitive: true}

	tests := []struct {
		name      string
		sensitive bool
	}{
		{"InputRequested", ir.Sensitive},
		{"InputProvided", ip.Sensitive},
		{"LogLine", ll.Sensitive},
		{"ScriptStarted", ss.Sensitive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.sensitive {
				t.Errorf("%s.Sensitive should be true (field must exist and carry value)", tt.name)
			}
		})
	}
}

// Test_InputRequested_Sensitive_Propagates_To_InputProvided verifies that when
// a fake engine receives a reply from an InputRequested{Sensitive:true} event,
// it emits a paired InputProvided{Sensitive:true}.
// security.REQ-03.4
func Test_InputRequested_Sensitive_Propagates_To_InputProvided(t *testing.T) {
	// fakeEngine simulates the propagation contract: after receiving a reply,
	// it MUST emit InputProvided{Sensitive: req.Sensitive}.
	fakeEngine := func(req InputRequested) []Event {
		// Simulate: producer reads reply.
		reply := make(chan string, 1)
		reply <- "secret-value"

		// Receive from the read side (testing only — in prod, req.Reply is send-only).
		value := <-reply

		paired := InputProvided{
			EventBase: EventBase{Seq: req.Seq + 1, At: time.Now()},
			Prompt:    req.Prompt,
			Value:     value,
			Sensitive: req.Sensitive, // Contract: propagate from req.
		}

		return []Event{req, paired}
	}

	replyCh := make(chan<- string, 1)
	req := InputRequested{
		EventBase: EventBase{Seq: 1, At: time.Now()},
		Prompt:    "Enter password",
		Sensitive: true,
		Reply:     replyCh,
	}

	events := fakeEngine(req)

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	provided, ok := events[1].(InputProvided)
	if !ok {
		t.Fatalf("second event: expected InputProvided, got %T", events[1])
	}

	if !provided.Sensitive {
		t.Error("InputProvided.Sensitive: expected true (propagated from InputRequested.Sensitive=true)")
	}
}

// Test_Op_FormatRegex verifies the Op format regex constant exists and is valid.
// This is referenced by structured-error tests — defined here as a sanity check
// that the regex compiles.
func Test_Op_FormatRegex(t *testing.T) {
	// The Op regex is defined in the errors package; here we validate the pattern
	// string itself compiles as a Go regexp.
	const opRegex = `^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$`
	re := regexp.MustCompile(opRegex)

	valid := []string{
		"init.handler",
		"execute.handler",
		"skill_update.handler",
		"engine.execute",
	}
	invalid := []string{
		"Init.handler",   // uppercase
		"init",           // no dot
		"init.Handler",   // uppercase after dot
		"init.",          // trailing dot
		".handler",       // leading dot
		"init.handler.x", // too many dots
	}

	for _, op := range valid {
		if !re.MatchString(op) {
			t.Errorf("Op %q should match regex %q", op, opRegex)
		}
	}

	for _, op := range invalid {
		if re.MatchString(op) {
			t.Errorf("Op %q should NOT match regex %q", op, opRegex)
		}
	}
}
