package transport

import "fmt"

type ErrorKind string

const (
	ErrorKindAuth       ErrorKind = "auth"
	ErrorKindTimeout    ErrorKind = "timeout"
	ErrorKindProtocol   ErrorKind = "protocol"
	ErrorKindValidation ErrorKind = "validation"
)

type Error struct {
	Kind ErrorKind
	Err  error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return string(e.Kind)
	}
	return fmt.Sprintf("%s: %v", e.Kind, e.Err)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func AuthError(err error) error {
	return &Error{Kind: ErrorKindAuth, Err: err}
}

func TimeoutError(err error) error {
	return &Error{Kind: ErrorKindTimeout, Err: err}
}

func ProtocolError(err error) error {
	return &Error{Kind: ErrorKindProtocol, Err: err}
}

func ValidationError(err error) error {
	return &Error{Kind: ErrorKindValidation, Err: err}
}

