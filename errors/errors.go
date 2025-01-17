package errors

import (
	"errors"
	"fmt"
)

const (
	UnknownCode = 500
)

type Error struct {
	Status
	cause error
}

// Error returns the string representation of the error message.
func (e *Error) Error() string {
	return fmt.Sprintf("code=%d message=%s metadata=%v cause=%v", e.Code, e.Message, e.Metadata, e.cause)
}

// Unwrap returns the cause of the error
func (e *Error) Unwrap() error {
	return e.cause
}

func (e *Error) WithMetadata(m map[string]string) *Error {
	err := Clone(e)
	for k, v := range m {
		err.Metadata[k] = v
	}

	return err
}

func (e *Error) WithCause(cause error) *Error {
	err := Clone(e)
	err.cause = cause

	return err
}

// Is reports whether err is an *Error with the same code and reason.
func (e *Error) Is(err error) bool {
	if ge := new(Error); errors.As(err, &ge) {
		return e.Code == ge.Code && e.Message == ge.Message
	}

	return false
}

// New returns an *Error representing the given code and message.
func New(code int, format string, args ...any) *Error {
	return &Error{
		Status: Status{
			Code:    int32(code),
			Message: fmt.Sprintf(format, args...),
		},
	}
}

// Clone returns a deep copy of err.
func Clone(err *Error) *Error {
	if err == nil {
		return nil
	}

	metadata := make(map[string]string, len(err.Metadata))
	for k, v := range err.Metadata {
		metadata[k] = v
	}

	return &Error{
		Status: Status{
			Code:     err.Code,
			Message:  err.Message,
			Metadata: metadata,
		},
	}
}

// FromError returns an *Error representing err if it was produced by this
func FromError(err error) *Error {
	if err == nil {
		return nil
	}

	if ge := new(Error); errors.As(err, &ge) {
		return ge
	}

	return New(UnknownCode, "%v", err)
}
