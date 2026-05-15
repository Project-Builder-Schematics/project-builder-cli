// Package tsident — reserved.go enumerates the canonical TypeScript reserved-word
// list used by EscapeIdent.
//
// REQ-TI-03: 69 entries — ECMAScript 2024 strict-mode keywords +
// TypeScript 5.x additional context-sensitive keywords.
// Sorted alphabetically; no duplicates.
//
// FF-14: every entry in this slice MUST have a corresponding test case in
// tsident_test.go asserting EscapeIdent(word) == word+"_".
// The fitness script scripts/fitness/tsident-reserved-coverage.sh enforces this.
package tsident

// ReservedWords is the canonical TypeScript reserved-word list (69 entries).
// EXPORTED so FF-14 (tsident-reserved-coverage.sh) can enumerate entries against
// the test matrix.
//
// Sorted alphabetically; no duplicates. Case-sensitive matching only.
//
// Sources:
//
//	ECMAScript 2024 strict-mode keywords (38, including yield):
//	  break case catch class const continue debugger default delete do
//	  else export extends false finally for function if import in
//	  instanceof let new null return static super switch this throw
//	  true try typeof var void while with yield
//	TypeScript 5.x additional keywords (31, adds infer):
//	  abstract any as async await constructor declare enum from get
//	  implements infer interface is keyof module namespace never of
//	  package private protected public readonly require set symbol type
//	  undefined unique unknown
var ReservedWords = []string{
	"abstract",
	"any",
	"as",
	"async",
	"await",
	"break",
	"case",
	"catch",
	"class",
	"const",
	"constructor",
	"continue",
	"debugger",
	"declare",
	"default",
	"delete",
	"do",
	"else",
	"enum",
	"export",
	"extends",
	"false",
	"finally",
	"for",
	"from",
	"function",
	"get",
	"if",
	"implements",
	"import",
	"in",
	"infer",
	"instanceof",
	"interface",
	"is",
	"keyof",
	"let",
	"module",
	"namespace",
	"never",
	"new",
	"null",
	"of",
	"package",
	"private",
	"protected",
	"public",
	"readonly",
	"require",
	"return",
	"set",
	"static",
	"super",
	"switch",
	"symbol",
	"this",
	"throw",
	"true",
	"try",
	"type",
	"typeof",
	"undefined",
	"unique",
	"unknown",
	"var",
	"void",
	"while",
	"with",
	"yield",
}
