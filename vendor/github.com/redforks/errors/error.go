package errors

import "bytes"

// Error contains error, causedBy, and stack.
type Error struct {
	Err error

	msg string // overloaded error message

	stack  []uintptr
	frames []StackFrame

	code Code
}

var _ CausedByError = &Error{}

func (err *Error) Error() string {
	if err.msg != "" {
		return err.msg
	}
	return err.Err.Error()
}

// Code returns error code.
func (err *Error) Code() Code {
	return err.code
}

// Inner returns inner error (.Err field), implements CausedByError interface
func (err *Error) Inner() error {
	return err.Err
}

// StackFrames returns an array of frames containing information about the
// stack.
func (err *Error) StackFrames() []StackFrame {
	if err.frames == nil {
		err.frames = make([]StackFrame, len(err.stack))

		for i, pc := range err.stack {
			err.frames[i] = NewStackFrame(pc)
		}
	}

	return err.frames
}

// Stack returns the callstack formatted the same way that go does
// in runtime/debug.Stack()
func (err *Error) Stack() string {
	buf := bytes.Buffer{}

	for _, frame := range err.StackFrames() {
		buf.WriteString(frame.String())
	}

	return string(buf.Bytes())
}

// ErrorStack returns a string that contains both the
// error message and the callstack, and inner Error's ErrorStack().
func (err *Error) ErrorStack() string {
	return err.Error() + "\n" + err.Stack()
	// if err.Err == nil {
	// 	return r
	// }

	// r += "\nInner error:\n"
	// switch inner := err.Err.(type) {
	// case CausedByError:
	// 	return r + inner.ErrorStack()
	// default:
	// 	return r + inner.Error()
	// }
}
