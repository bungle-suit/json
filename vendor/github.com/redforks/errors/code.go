package errors

// Code each code is an error, first 8 bit is CausedBy.
type Code uint32

// Caused corresponding CausedBy value
func (code Code) Caused() CausedBy {
	return CausedBy(code & 0xff000000)
}

// NewCode create a new code
func NewCode(cause CausedBy, low uint32) Code {
	return Code(uint32(cause) + low)
}

const (
	// NotError returned by GetCode() if value is nil.
	NotError = Code(0)

	// GeneralByBug generic ByBug error code
	GeneralByBug = Code(ByBug)

	// GeneralByRuntime general ByRuntime error code
	GeneralByRuntime = Code(ByRuntime)

	// GeneralByExternal general ByExternal error code
	GeneralByExternal = Code(ByExternal)

	// GeneralByInput general ByInput error code
	GeneralByInput = Code(ByInput)

	// GeneralByClientBug general ByClientBug error code
	GeneralByClientBug = Code(ByClientBug)
)
