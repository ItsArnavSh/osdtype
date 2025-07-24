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
}
