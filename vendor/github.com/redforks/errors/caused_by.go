package errors

// CausedBy describe who caused this error.
type CausedBy uint32

const (
	// ByBug error caused by a bug. This kind of error normally logged and/or
	// report to error report service, notify to developer. Did not show detail
	// error to end-user, it is not their fault, do not blame them, and apologize
	// for this internal error.
	ByBug CausedBy = (iota + 1) << 24

	// ByRuntime error caused by golang runtime, such as run out of memory, file
	// read/write error. Any error caused by OS, or hardware. Network interface
	// error is caused by Runtime, error caused by network cable is not. Report
	// this kind of error to health monitor service, notify maintains team as
	// fast as possible. It not need to report to error service.
	ByRuntime

	// ByExternal error caused by depended external service, such as a Database
	// or other app services. Or network environment, such as lost network
	// connection. This kind of error is can not fixed by patching code, patching
	// OS, upgrade or replace hardware, they are not our error, nothing we can
	// do.  Report to health monitor service, notify maintains team to contact
	// people who can fix this.
	// Give end-user a brief message that who caused this error that they can
	// understand, such as payment service, and user know this error is not
	// caused by us.
	ByExternal

	// ByInput error caused by bad input. A program always dealing with input,
	// such as user input, or a request for a service daemon. When the input is
	// not expected, return error with detail and precise reason. Of course, do
	// not need report to error report service or health monitor service.
	ByInput

	// ByClientBug error caused by wrong implemented client software, such as
	// wrong argument. ByInput is error caused by user. Normally ByClientBug not
	// report to health/crash report service.
	ByClientBug

	// NoError is a special value returned by GetPanicCausedBy() to indicate no
	// error happened
	NoError
)
