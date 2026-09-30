package entity

import "strconv"

// Language is a codegen target. The value is persisted and also derived from a
// round's seed, so it is deliberately a small dense integer.
type Language int

// The languages the snippet service can generate for.
const (
	C          Language = iota
	GO         Language = 1
	CPP        Language = 2
	JAVA       Language = 3
	RUST       Language = 4
	TYPESCRIPT Language = 5
)

// KnownLanguages is every supported language, in value order.
//
// The count matters: a round derives its language as seed % len(KnownLanguages),
// so this slice and String must stay in step.
var KnownLanguages = []Language{C, GO, CPP, JAVA, RUST, TYPESCRIPT}

// languageNames is indexed by language value.
//
// String bounds checks before indexing. The value reaches the process from the
// database and from a seed supplied over the WebSocket, so indexing directly
// panicked with an index out of range for anything outside the known set, and
// any client could trigger it.
var languageNames = [...]string{"c", "go", "cpp", "java", "rs", "ts"}

// String is the codegen name for the language, or an empty string if the value
// is not a language.
func (l Language) String() string {
	if l < 0 || int(l) >= len(languageNames) {
		return ""
	}
	return languageNames[l]
}

// LanguageFromString resolves a language name, reporting whether it was known.
func LanguageFromString(s string) (Language, bool) {
	for i, name := range languageNames {
		if name == s {
			return Language(i), true
		}
	}
	return C, false
}

// LanguageFromInt converts a raw integer, as stored in a database column or
// sent by a client, clamping to a known language rather than failing.
//
// The frontend hits this constantly: contest.lang is a plain integer in the
// database, and indexing a Record<Language, ...> with it used to produce
// undefined for anything the enum had no case for.
func LanguageFromInt(n int) (Language, bool) {
	if n < 0 || n >= len(languageNames) {
		return C, false
	}
	return Language(n), true
}

// ParseLanguage resolves a language from either its name or its number, which
// lets an endpoint accept "go" or "1" for the same language.
func ParseLanguage(s string) (Language, bool) {
	if lang, ok := LanguageFromString(s); ok {
		return lang, true
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return C, false
	}
	return LanguageFromInt(n)
}

// TypeInfo is the snippet a run used, so a past run can be replayed exactly.
type TypeInfo struct {
	SnippetSeed uint32
	Lang        Language
}

// DisplayName is the language's name as shown in the interface, which differs
// from its codegen name: the generator calls it "rs" and "ts", a person
// expects "Rust" and "TypeScript".
func (l Language) DisplayName() string {
	switch l {
	case C:
		return "C"
	case GO:
		return "Go"
	case CPP:
		return "C++"
	case JAVA:
		return "Java"
	case RUST:
		return "Rust"
	case TYPESCRIPT:
		return "TypeScript"
	}
	return "Unknown"
}

// LineComment returns the language's line comment marker, which the interface
// needs to syntax highlight generated code.
func (l Language) LineComment() string {
	switch l {
	case GO, CPP, JAVA, RUST:
		return "//"
	case TYPESCRIPT:
		return "//"
	case C:
		return "//"
	}
	return "//"
}
