package services

import "fmt"

type Error struct {
	Kind error
	Msg  string
}

func (e *Error) Error() string { return e.Msg }
func (e *Error) Unwrap() error { return e.Kind }

func errf(kind error, format string, args ...any) error {
	return &Error{Kind: kind, Msg: fmt.Sprintf(format, args...)}
}

var (
	ErrNotFound         = fmt.Errorf("not found")
	ErrForbidden        = fmt.Errorf("forbidden")
	ErrConflict         = fmt.Errorf("conflict")
	ErrBadRequest       = fmt.Errorf("bad request")
	ErrPasswordRequired = fmt.Errorf("password required")
	ErrWrongPassword    = fmt.Errorf("wrong password")
	ErrFull             = fmt.Errorf("room full")
	ErrBanned           = fmt.Errorf("banned")
	ErrUnauthorized     = fmt.Errorf("unauthorized")
	ErrRateLimited      = fmt.Errorf("rate limited")
)
