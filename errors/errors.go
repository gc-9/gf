package errors

import (
	"fmt"
	"io"
)

type ErrMessage struct {
	Code     int
	Cause    error
	HumanMsg string // humanMsg
	Stack    *stack
}

func (t *ErrMessage) StackTrace() StackTrace {
	if t.Stack == nil {
		return nil
	}
	return t.Stack.StackTrace()
}

func (t *ErrMessage) Error() string {
	if t.Cause != nil {
		if t.HumanMsg == "" {
			return t.Cause.Error()
		}
		return t.HumanMsg + ": " + t.Cause.Error()
	}
	return t.HumanMsg
}

func (t *ErrMessage) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			if t.HumanMsg != "" {
				fmt.Fprintf(s, "%s: ", t.HumanMsg)
			}
			if t.Cause != nil {
				fmt.Fprintf(s, "%+v", t.Cause)
			}
			t.Stack.Format(s, verb)
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, t.Error())
	case 'q':
		fmt.Fprintf(s, "%q", t.Error())
	}
}

func New(humanMsg string) error {
	return &ErrMessage{HumanMsg: humanMsg}
}

func Errorf(format string, args ...interface{}) error {
	return &ErrMessage{HumanMsg: fmt.Sprintf(format, args...)}
}

func WithStackf(format string, args ...interface{}) error {
	return &ErrMessage{
		Cause: Errorf(format, args...),
		Stack: callers(),
	}
}

func WithStack(humanMsg string) error {
	return &ErrMessage{
		HumanMsg: humanMsg,
		Stack:    callers(),
	}
}

func Wrap(err error, humanMsg string) error {
	if err == nil {
		return nil
	}
	return &ErrMessage{HumanMsg: humanMsg, Cause: err, Stack: callers()}
}

func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return &ErrMessage{HumanMsg: fmt.Sprintf(format, args...), Cause: err, Stack: callers()}
}
