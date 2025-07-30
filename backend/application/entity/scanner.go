package entity

type ScanCurr int

const (
	ALPHANUM ScanCurr = iota
	OPERATOR
	FREE
)
