// Package filename handles disallowed characters in strings to make them
// usable as filenames.
package filename

// NOTE: As ErrInvalidFileNameLength is a format string, clients wouldn't be
// able to use it directly to compare against errors. Also, nyfiken doesn't make
// use of it. Therefore ErrInvalidFileNameLength could be made an unexported
// constant instead of a global variable.

// Common filename errors.
var (
	ErrInvalidFileNameLength = "Invalid filename length: %d - max length is: %d"
)
