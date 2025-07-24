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

type KeyDef struct {
	Delete bool  `json:"Delete"`
	Effect rune  `json:"effect"`
	Time   int64 `json:"time"`
}
