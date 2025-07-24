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
	Id   string
	Lang Language
}

// ////////
type KeyStroke int

const (
	KeyDown KeyStroke = iota
	KeyUp
)

type KeyDef struct {
	Action KeyStroke `json:"action"`
	Effect rune      `json:"effect"`
	Time   int64     `json:"time"`
}
