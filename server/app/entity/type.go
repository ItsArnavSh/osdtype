package entity

type Language int

const (
	C          Language = iota
	GO         Language = 1
	CPP        Language = 2
	JAVA       Language = 3
	RUST       Language = 4
	TYPESCRIPT Language = 5
)

// String returns the codegen name for the language.
//
// The value can come from the database or from a client-supplied seed, so the
// lookup is bounds checked: indexing the table directly panicked with an index
// out of range for anything outside the known set.
func (l Language) String() string {
	names := [...]string{"c", "go", "cpp", "java", "rs", "ts"}
	if l < 0 || int(l) >= len(names) {
		return ""
	}
	return names[l]
}

type TypeInfo struct {
	SnippetSeed uint32
	Lang        Language
}

// ////////
// If Deleted is false, that means only +1 appended
// If Deleted is true, then the val at effect was removed
// Client side will apply diffs on the text and only delta will be sent to us
type KeyDef struct {
	Delete bool   `json:"delete"`
	Delta  string `json:"delta"`
	Time   int32  `json:"time"`
}

// ///////
type Recording struct {
	ID         uint32
	Recording  []byte   // Compressed Recording
	Diff       []KeyDef // All the keystrokes recording
	Final      string   // What did the person write
	OriginalID string   //
	RunID      string   // Special ID of the run
	Timestamps []int32  // Timestamps
}
