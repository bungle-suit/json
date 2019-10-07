package errors

// CausedByError is interface provided CausedBy info.
// CausedByError make custom Error implementation possible.
type CausedByError interface {
	error

	// Inner error maybe nil
	Inner() error

	Code() Code

	// ErrorStack returns a string that contains both the
	// error message and the callstack.
	ErrorStack() string
}
