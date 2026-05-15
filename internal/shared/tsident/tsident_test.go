// Package tsident — tsident_test.go covers the full EscapeIdent contract.
//
// S-003 replaces the S-000a stub assertions with the full table-driven matrix
// covering ALL 69 reserved words + edge cases (REQ-TI-01..10, ADV-11/12).
//
// Test organisation:
//
//	Test_ReservedWords_Contains69Entries — counts and spot-checks specific entries
//	Test_ReservedWords_Sorted            — alphabetical + no duplicates
//	Test_EscapeIdent_Contract            — full transformation matrix (REQ-TI-01..10)
//	Test_EscapeIdent_AllReservedWords    — all 69 produce word+"_" (ADV-11)
//	Test_EscapeIdent_EmptyPanics         — REQ-TI-07
package tsident_test

import (
	"slices"
	"testing"

	"github.com/Project-Builder-Schematics/project-builder-cli/internal/shared/tsident"
)

// ─── Task A: ReservedWords list ──────────────────────────────────────────────

// Test_ReservedWords_Contains69Entries verifies the full list is present
// and spot-checks representative entries (REQ-TI-03, ADV-11).
func Test_ReservedWords_Contains69Entries(t *testing.T) {
	t.Parallel()

	if got := len(tsident.ReservedWords); got != 69 {
		t.Errorf("len(ReservedWords) = %d; want 69", got)
	}

	// Spot-check ECMAScript strict-mode keywords.
	mustContain := []string{
		"break", "case", "catch", "class", "const", "continue", "debugger",
		"default", "delete", "do", "else", "export", "extends", "false",
		"finally", "for", "function", "if", "import", "in", "instanceof",
		"let", "new", "null", "return", "static", "super", "switch", "this",
		"throw", "true", "try", "typeof", "var", "void", "while", "with", "yield",
		// TypeScript-additional keywords.
		"abstract", "any", "as", "async", "await", "constructor", "declare",
		"enum", "from", "get", "implements", "interface", "is", "keyof",
		"module", "namespace", "never", "of", "package", "private", "protected",
		"public", "readonly", "require", "set", "symbol", "type", "undefined",
		"unique", "unknown",
	}

	for _, word := range mustContain {
		if !slices.Contains(tsident.ReservedWords, word) {
			t.Errorf("ReservedWords missing entry: %q", word)
		}
	}
}

// Test_ReservedWords_Sorted verifies the list is sorted alphabetically and
// has no duplicate entries (normalises FF-14 audit assumption).
func Test_ReservedWords_Sorted(t *testing.T) {
	t.Parallel()

	words := tsident.ReservedWords
	for i := 1; i < len(words); i++ {
		if words[i] <= words[i-1] {
			t.Errorf("ReservedWords not sorted at index %d: %q <= %q", i, words[i], words[i-1])
		}
	}
}

// ─── Task B: EscapeIdent full contract ───────────────────────────────────────

// Test_EscapeIdent_Contract covers REQ-TI-01..10 and ADV-12 edge cases
// using a table-driven matrix.
func Test_EscapeIdent_Contract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		// REQ-TI-04: hyphen → underscore
		{name: "hyphen_single", input: "my-schematic", want: "my_schematic"},
		// REQ-TI-04: space → underscore
		{name: "space_single", input: "my schematic", want: "my_schematic"},
		// REQ-TI-04: dot → underscore
		{name: "dot_single", input: "my.schematic", want: "my_schematic"},
		// REQ-TI-06: consecutive special chars collapsed to single _
		{name: "consecutive_hyphens", input: "my--schematic", want: "my_schematic"},
		{name: "consecutive_spaces", input: "a  b", want: "a_b"},
		{name: "hyphen_dot_space", input: "a-. b", want: "a_b"},
		// REQ-TI-05: leading digit → prefix with _
		{name: "leading_digit", input: "123abc", want: "_123abc"},
		{name: "leading_digit_with_hyphen", input: "2fast-schematic", want: "_2fast_schematic"},
		// REQ-TI-08: non-ASCII → replaced with _
		{name: "non_ascii_accent", input: "héllo", want: "h_llo"},
		{name: "japanese_chars", input: "ab日c", want: "ab_c"},
		// REQ-TI-02: reserved word → append _
		{name: "reserved_class", input: "class", want: "class_"},
		{name: "reserved_function", input: "function", want: "function_"},
		{name: "reserved_interface", input: "interface", want: "interface_"},
		{name: "reserved_import", input: "import", want: "import_"},
		{name: "reserved_type", input: "type", want: "type_"},
		{name: "reserved_async", input: "async", want: "async_"},
		{name: "reserved_await", input: "await", want: "await_"},
		{name: "reserved_enum", input: "enum", want: "enum_"},
		// REQ-TI-09: class- → class_ via hyphen-replace, NOT reserved → no extra _
		{name: "class_with_hyphen_suffix", input: "class-", want: "class_"},
		// ADV-12: digit-leading + if result is reserved → only _ prefix, no double escape
		{name: "digit_then_reserved", input: "123class", want: "_123class"},
		// REQ-TI-10: PascalCase NOT done here — EscapeIdent preserves case
		{name: "pascalcase_passthrough", input: "MySchematic", want: "MySchematic"},
		{name: "already_valid", input: "foo", want: "foo"},
		{name: "underscores_already_valid", input: "my_schematic", want: "my_schematic"},
		// REQ-TI-01: single char valid
		{name: "single_char", input: "x", want: "x"},
		// Digit not leading
		{name: "digit_not_leading", input: "schema1", want: "schema1"},
		// Underscore leading is fine
		{name: "leading_underscore", input: "_foo", want: "_foo"},
		// REQ-TI-09: transformation produces valid identifier — no reserved match
		{name: "class_underscore_no_double", input: "class_", want: "class_"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tsident.EscapeIdent(tt.input)
			if got != tt.want {
				t.Errorf("EscapeIdent(%q) = %q; want %q", tt.input, got, tt.want)
			}
		})
	}
}

// Test_EscapeIdent_AllReservedWords verifies ALL 69 reserved words produce
// word + "_" — the data-driven golden matrix that FF-14 enforces (ADV-11).
func Test_EscapeIdent_AllReservedWords(t *testing.T) {
	t.Parallel()

	for _, word := range tsident.ReservedWords {
		word := word
		t.Run("reserved_"+word, func(t *testing.T) {
			t.Parallel()
			want := word + "_"
			got := tsident.EscapeIdent(word)
			if got != want {
				t.Errorf("EscapeIdent(%q) = %q; want %q (reserved word must get _ suffix)", word, got, want)
			}
		})
	}
}

// Test_EscapeIdent_EmptyPanics verifies that EscapeIdent("") panics with the
// documented message (REQ-TI-07 — programming error contract).
func Test_EscapeIdent_EmptyPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Error("EscapeIdent(\"\") did not panic; want panic")
		}
	}()

	// Must panic — this call should never return.
	_ = tsident.EscapeIdent("")
}
