package errors

import (
	stdErrors "errors"
	"fmt"
	"io"
)

type ErrMessage struct {
	Code         int
	InternalCode int
	Public       bool
	Message      string
	Cause        error
	Stack        *stack
}

// Option configures an ErrMessage created by a public error constructor.
type Option func(*ErrMessage)

// WithCode assigns an application response code to a public error.
func WithCode(code int) Option {
	return func(err *ErrMessage) {
		err.Code = code
	}
}

// WithInternalCode assigns an internal-only code for diagnostics and handling
// during error propagation. It does not affect the public response code.
func WithInternalCode(code int) Option {
	return func(err *ErrMessage) {
		err.InternalCode = code
	}
}

func (t *ErrMessage) StackTrace() StackTrace {
	if t.Stack == nil {
		return nil
	}
	return t.Stack.StackTrace()
}

func (t *ErrMessage) Error() string {
	if t.Cause != nil {
		if t.Message == "" {
			return t.Cause.Error()
		}
		return t.Message + ": " + t.Cause.Error()
	}
	return t.Message
}

// Unwrap exposes the original cause to the standard errors package.
func (t *ErrMessage) Unwrap() error {
	return t.Cause
}

func (t *ErrMessage) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			if t.Message != "" {
				fmt.Fprintf(s, "%s", t.Message)
			}
			if t.Cause != nil {
				if t.Message != "" {
					io.WriteString(s, ": ")
				}
				fmt.Fprintf(s, "%+v", t.Cause)
			}
			if t.Stack != nil {
				t.Stack.Format(s, verb)
			}
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, t.Error())
		return
	case 'q':
		fmt.Fprintf(s, "%q", t.Error())
	}
}

func New(message string) error {
	return &ErrMessage{Message: message}
}

func Errorf(format string, args ...interface{}) error {
	return &ErrMessage{Message: fmt.Sprintf(format, args...)}
}

func WithStackf(format string, args ...interface{}) error {
	return &ErrMessage{
		Message: fmt.Sprintf(format, args...),
		Stack:   callers(3),
	}
}

func WithStack(message string) error {
	return &ErrMessage{
		Message: message,
		Stack:   callers(3),
	}
}

// Wrap adds internal context to err. It captures a stack trace only when the
// cause chain does not already contain one.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return wrap(err, message, false, nil)
}

func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return wrap(err, fmt.Sprintf(format, args...), false, nil)
}

// Public creates an error whose message is safe to return to an API caller.
func Public(message string, options ...Option) error {
	err := &ErrMessage{Public: true, Message: message}
	applyOptions(err, options)
	return err
}

// PublicWrap adds a caller-safe message while retaining the original cause.
// Like Wrap, it captures a stack trace only when the cause chain has none.
func PublicWrap(err error, message string, options ...Option) error {
	if err == nil {
		return nil
	}
	return wrap(err, message, true, options)
}

// EnsurePublic preserves errors that are already safe to expose. Other errors
// are wrapped with message so callers receive a safe, consistent response.
func EnsurePublic(err error, message string, options ...Option) error {
	if err == nil {
		return nil
	}

	var appErr *ErrMessage
	if stdErrors.As(err, &appErr) && appErr.Public {
		return err
	}
	return PublicWrap(err, message, options...)
}

func wrap(cause error, message string, public bool, options []Option) error {
	err := &ErrMessage{Public: public, Message: message, Cause: cause}
	applyOptions(err, options)
	if !hasStack(cause) {
		err.Stack = callers(4)
	}
	return err
}

func applyOptions(err *ErrMessage, options []Option) {
	for _, option := range options {
		if option != nil {
			option(err)
		}
	}
}

func hasStack(err error) bool {
	if err == nil {
		return false
	}
	if stackErr, ok := err.(interface{ StackTrace() StackTrace }); ok && len(stackErr.StackTrace()) > 0 {
		return true
	}
	if unwrapMany, ok := err.(interface{ Unwrap() []error }); ok {
		for _, cause := range unwrapMany.Unwrap() {
			if hasStack(cause) {
				return true
			}
		}
		return false
	}
	if unwrapOne, ok := err.(interface{ Unwrap() error }); ok {
		return hasStack(unwrapOne.Unwrap())
	}
	return false
}
