package errors

import (
	"errors"
	"fmt"
	"maps"
	"strconv"
	"strings"
)

const (
	UnknownCode = 500
)

// Error represents a structured error with HTTP status code, message, metadata and cause chain
type Error struct {
	Status
	cause error
}

// Error returns a human-readable error message with optional cause chain
func (e *Error) Error() string {
	var msg strings.Builder
	msg.WriteString("code=")
	msg.WriteString(strconv.Itoa(int(e.Code)))
	msg.WriteString(" │ message=")
	msg.WriteString(e.Message)

	// Append metadata if present
	if len(e.Metadata) > 0 {
		msg.WriteString(" │ metadata={")
		first := true
		for k, v := range e.Metadata {
			if !first {
				msg.WriteString(", ")
			}
			msg.WriteString(k)
			msg.WriteString("=")
			msg.WriteString(v)
			first = false
		}
		msg.WriteString("}")
	}

	// Append cause if present
	if e.cause != nil {
		msg.WriteString(" │ cause=")
		msg.WriteString(e.cause.Error())
	}

	return msg.String()
}

// Unwrap returns the cause of the error
func (e *Error) Unwrap() error {
	return e.cause
}

// WithMetadata adds metadata to the error. Returns a new error instance to maintain immutability.
func (e *Error) WithMetadata(m map[string]string) *Error {
	if len(m) == 0 {
		return e
	}

	// 只有在实际需要添加元数据时才克隆
	err := e.clone()
	if err.Metadata == nil {
		err.Metadata = make(map[string]string, len(m))
	}

	maps.Copy(err.Metadata, m)

	return err
}

// WithCause adds a cause to the error. Returns a new error instance to maintain immutability.
func (e *Error) WithCause(cause error) *Error {
	if cause == nil {
		return e
	}

	err := e.clone()
	err.cause = cause
	return err
}

// clone creates a shallow copy of the error with deep copy of metadata map
func (e *Error) clone() *Error {
	var metadata map[string]string
	if len(e.Metadata) > 0 {
		metadata = make(map[string]string, len(e.Metadata))
		maps.Copy(metadata, e.Metadata)
	}

	return &Error{
		Status: Status{
			Code:     e.Code,
			Message:  e.Message,
			Metadata: metadata,
		},
		cause: e.cause,
	}
}

// Is reports whether err is an *Error with the same code and message.
// This implements the standard errors.Is interface for better error comparison.
func (e *Error) Is(err error) bool {
	var ge *Error
	if errors.As(err, &ge) {
		return e.Code == ge.Code && e.Message == ge.Message
	}
	return false
}

// GetCode returns the error code
func (e *Error) GetCode() int32 {
	return e.Code
}

// GetMessage returns the error message
func (e *Error) GetMessage() string {
	return e.Message
}

// GetMetadata returns a copy of the metadata to prevent external modification
func (e *Error) GetMetadata() map[string]string {
	if len(e.Metadata) == 0 {
		return nil
	}

	result := make(map[string]string, len(e.Metadata))
	maps.Copy(result, e.Metadata)
	return result
}

// GetCause returns the underlying cause of the error
func (e *Error) GetCause() error {
	return e.cause
}

// New creates a new Error with the given code and formatted message.
// Uses efficient formatting to avoid unnecessary allocations.
func New(code int, format string, args ...any) *Error {
	var message string
	if len(args) == 0 {
		message = format
	} else {
		message = fmt.Sprintf(format, args...)
	}

	return &Error{
		Status: Status{
			Code:    int32(code),
			Message: message,
		},
	}
}

// NewWithMetadata creates a new Error with metadata
func NewWithMetadata(code int, metadata map[string]string, format string, args ...any) *Error {
	err := New(code, format, args...)
	if len(metadata) > 0 {
		err.Metadata = make(map[string]string, len(metadata))
		maps.Copy(err.Metadata, metadata)
	}
	return err
}

// Clone returns a deep copy of err.
// Deprecated: Use the clone() method instead for better performance.
func Clone(err *Error) *Error {
	if err == nil {
		return nil
	}
	return err.clone()
}

// FromError converts a generic error to an *Error.
// If the error is already an *Error, it returns it directly for better performance.
// Otherwise, it wraps it with UnknownCode.
func FromError(err error) *Error {
	if err == nil {
		return nil
	}

	var ge *Error
	if errors.As(err, &ge) {
		return ge
	}

	return New(UnknownCode, "%v", err)
}

// Wrap wraps an error with additional context while preserving the original error chain
func Wrap(err error, code int, format string, args ...any) *Error {
	if err == nil {
		return nil
	}

	newErr := New(code, format, args...)
	return newErr.WithCause(err)
}

// WrapWithMetadata wraps an error with metadata and additional context
func WrapWithMetadata(err error, code int, metadata map[string]string, format string, args ...any) *Error {
	if err == nil {
		return nil
	}

	newErr := NewWithMetadata(code, metadata, format, args...)
	return newErr.WithCause(err)
}
