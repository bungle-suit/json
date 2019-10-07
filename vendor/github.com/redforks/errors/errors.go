//go:generate stringer -type=CausedBy

// Package errors Enhanced errors package, add CausedBy information. Can
// replace standard errors package.
//
// Error classified by who caused the error:
//
//  1. Bug: we programmers are the one responsible for the error.
//  2. Runtime: OS, hardware, golang, code libraries, normally can be fixed by
//  DevOp.
//  3. External: caused by external service, or network connection.
//  4. Input: caused by bad input.
//
// The first two kinds can be fixed by ourself, and the last two we can not
// fixed must report back.
package errors

import (
	syserr "errors"
	"fmt"
	"runtime"
	"runtime/debug"
)

const maxStackDepth = 50

// New function replace of standard errors.New(), create a ByBug error.
func New(text string) *Error {
	return wrap(syserr.New(text), ByBug)
}

// Wrap an exist error
func Wrap(causedBy CausedBy, err interface{}, text string) *Error {
	var (
		er error
		ok bool
	)

	if er, ok = err.(error); !ok {
		er = fmt.Errorf("%v", err)
	}

	e := wrap(er, causedBy)
	e.msg = text
	return e
}

// Wrapf is format version of Wrap()
func Wrapf(causedBy CausedBy, err interface{}, text string, a ...interface{}) *Error {
	var (
		er error
		ok bool
	)

	if er, ok = err.(error); !ok {
		er = fmt.Errorf("%v", err)
	}

	e := wrap(er, causedBy)
	e.msg = fmt.Sprintf(text, a...)
	return e
}

func wrap(e error, causedBy CausedBy) *Error {
	if e == nil {
		return nil
	}

	stack := make([]uintptr, maxStackDepth)
	length := runtime.Callers(3, stack[:])
	stack = stack[:length]
	return &Error{
		Err:   e,
		stack: stack,

		code: Code(causedBy),
	}
}

// NewBug wrap an exist error to ByBug. If e is nil, return nil. If e is
// already an Error, wrap it to ByBug.
func NewBug(e error) *Error {
	return wrap(e, ByBug)
}

// NewRuntime wrap an exist error to ByRuntime. If e is nil, return nil. If e is
// already an Error, wrap it to ByRuntime.
func NewRuntime(e error) *Error {
	return wrap(e, ByRuntime)
}

// NewExternal wrap an exist error to ByRuntime. If e is nil, return nil. If e is
// already an Error, wrap it to ByExternal.
func NewExternal(e error) *Error {
	return wrap(e, ByExternal)
}

// NewInput wrap an exist error to ByRuntime. If e is nil, return nil. If e is
// already an Error, wrap it to ByInput.
func NewInput(e error) *Error {
	return wrap(e, ByInput)
}

// NewClientBug wrap an exist error to ByRuntime. If e is nil, return nil. If e is
// already an Error, wrap it to ByClientBug.
func NewClientBug(e error) *Error {
	return wrap(e, ByClientBug)
}

// Bug creates an Error from string.
func Bug(text string) *Error {
	return wrap(syserr.New(text), ByBug)
}

// Bugf sprintf version of Bug().
func Bugf(text string, a ...interface{}) *Error {
	return wrap(fmt.Errorf(text, a...), ByBug)
}

// Runtime creates an Error from string.
func Runtime(text string) *Error {
	return wrap(syserr.New(text), ByRuntime)
}

// Runtimef sprintf version of Runtime().
func Runtimef(text string, a ...interface{}) *Error {
	return wrap(fmt.Errorf(text, a...), ByRuntime)
}

// External creates an Error from string.
func External(text string) *Error {
	return wrap(syserr.New(text), ByExternal)
}

// Externalf sprintf version of Runtime().
func Externalf(text string, a ...interface{}) *Error {
	return wrap(fmt.Errorf(text, a...), ByExternal)
}

// Input creates an Error from string.
func Input(text string) *Error {
	return wrap(syserr.New(text), ByInput)
}

// Inputf sprintf version of Input.
func Inputf(text string, a ...interface{}) *Error {
	return wrap(fmt.Errorf(text, a...), ByInput)
}

// ClientBug creates an Error from string.
func ClientBug(text string) *Error {
	return wrap(syserr.New(text), ByClientBug)
}

// ClientBugf sprintf version of ClientBug.
func ClientBugf(text string, a ...interface{}) *Error {
	return wrap(fmt.Errorf(text, a...), ByClientBug)
}

// Caused create error causedBy set by argument
func Caused(causedBy CausedBy, text string) *Error {
	return wrap(syserr.New(text), causedBy)
}

// Causedf sprintf version of Caused
func Causedf(causedBy CausedBy, text string, a ...interface{}) *Error {
	return wrap(fmt.Errorf(text, a...), causedBy)
}

// NewCaused wraps an exist error to specified causedBy Error,
// If e is nil, return nil.
// If already an Error, returned directly if causedBy matches, re-wrap
// with specific causedBy if not matched.
func NewCaused(causedBy CausedBy, err error) *Error {
	return wrap(err, causedBy)
}

// GetCausedBy from any error. If the error is Error interface, call its
// CausedBy() method. Then all considered as ByBug.
//
// If the error is not a bug, wrap it use NewXXX() function before return:
//
//  if err := os.Open("file"); err != nil {
//		return errors.NewRuntime(err)
//  }
//
func GetCausedBy(e error) CausedBy {
	switch err := e.(type) {
	case nil:
		return NoError
	case CausedByError:
		return err.Code().Caused()
	default:
		return ByBug
	}
}

// GetPanicCausedBy resolve caused for recover() return value, if the value is
// nil, return NoError. For error value use GetCausedBy() to resolve, other
// value return ByBug.
func GetPanicCausedBy(v interface{}) CausedBy {
	switch err := v.(type) {
	case nil:
		return NoError
	case CausedByError:
		return err.Code().Caused()
	default:
		return ByBug
	}
}

// GetCode returns error code from any value, if v is not CausedByError,
// returns GenericByBug. Returns NotError if v is nil.
func GetCode(v interface{}) Code {
	switch er := v.(type) {
	case nil:
		return NotError
	case CausedByError:
		return er.Code()
	default:
		return GeneralByBug
	}
}

// ForLog convert value to string for better logging:
//
//  1. if v is *Error, use .ErrorStack
//  2. if v is error, use .Error()
//  3. otherwise, use fmt.Sprint(v)
func ForLog(v interface{}) string {
	switch e := v.(type) {
	case *Error:
		s := e.ErrorStack()
		if inner := e.Inner(); inner != nil {
			s += "\nInner error:\n" + ForLog(inner)
		}
		return s
	case error:
		return e.Error()
	default:
		// If a value is not error, then it must recovered from panic
		return fmt.Sprintf("%v\n%s", v, string(debug.Stack()))
	}
}
