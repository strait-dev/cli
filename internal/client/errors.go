package client

import (
	"errors"
	"fmt"
	"net/http"
)

// APIError is a structured error returned by client request methods when the
// server responds with a non-2xx status. Callers can extract it with errors.As
// to branch on the underlying StatusCode (e.g. fall back to a list lookup only
// when GetX returns 404 rather than on a 500 or a transport error).
type APIError struct {
	StatusCode int
	Message    string
	// Code is the machine-readable error code from the server's error envelope
	// (e.g. "bad_request", "not_found", "validation_error"). Empty when the
	// response body carried no structured code.
	Code string
	// Op identifies the operation that failed, e.g. "request", "upload",
	// "run stream". Used as the prefix of Error so errors stringify the same
	// way they did before this type was introduced.
	Op string
}

// Error returns the human-readable form: "<op> failed (<status>): <msg>".
// This matches the prior fmt.Errorf format so existing tests and log output
// continue to read identically.
func (e *APIError) Error() string {
	op := e.Op
	if op == "" {
		op = "request"
	}
	return fmt.Sprintf("%s failed (%d): %s", op, e.StatusCode, e.Message)
}

// IsAPIErrorWithStatus reports whether err is (or wraps) an APIError with the
// given status code.
func IsAPIErrorWithStatus(err error, status int) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == status
	}
	return false
}

// IsNotFound reports whether err is (or wraps) an APIError with status 404.
func IsNotFound(err error) bool {
	return IsAPIErrorWithStatus(err, http.StatusNotFound)
}
