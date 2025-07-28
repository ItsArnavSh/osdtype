package entity

type Language int

const (
	GO Language = iota
	Java
	Python
	Typescript
	Javascript
	CPP
)

func (l Language) String() string {
	return [...]string{"go", "java", "python", "typescript", "javascript", "cpp"}[l]
}

type TypeInfo struct {
	SnippetID string
	Lang      Language
}

// ////////
// If Deleted is false, that means only +1 appended
// If Deleted is true, then the val at effect was removed
// Client side will apply diffs on the text and only delta will be sent to us
type KeyDef struct {
	Delete bool   `json:"delete"`
	Delta  string `json:"delta"`
	Time   int64  `json:"time"`
}

// ///////
type Recording struct {
	Recording  []byte   //Compressed Recording
	Diff       []KeyDef //All the keystrokes recording
	Final      string   //What did the person write
	OriginalID string   //
	RunID      string   //Special ID of the run
	Timestamps []int64  //Timestamps
}
