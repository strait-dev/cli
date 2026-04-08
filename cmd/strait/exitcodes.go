package main

import (
	"regexp"
	"strconv"
	"strings"
)

// Exit codes for machine-readable status signalling.
// Agents and scripts should test these codes rather than parsing stderr.
const (
	// ExitOK is the success exit code.
	ExitOK = 0
	// ExitGeneralError is returned for unexpected or unclassified errors.
	ExitGeneralError = 1
	// ExitPanic is returned when the CLI panics.
	ExitPanic = 2
	// ExitConfigError is returned for configuration or usage errors
	// (missing flags, invalid arguments, bad config file).
	ExitConfigError = 3
	// ExitAuthError is returned when the API key is missing, invalid, or expired (HTTP 401/403).
	ExitAuthError = 4
	// ExitNotFound is returned when the requested resource does not exist (HTTP 404).
	ExitNotFound = 5
	// ExitConflict is returned when an operation conflicts with existing state (HTTP 409).
	ExitConflict = 6
	// ExitValidation is returned when the request is rejected by the server due to invalid
	// input (HTTP 400/422).
	ExitValidation = 7
	// ExitServerError is returned when the server reports an internal error (HTTP 5xx).
	ExitServerError = 8
)

// exitCodeName maps a numeric exit code to a stable slug used in JSON error output.
func exitCodeName(code int) string {
	switch code {
	case ExitOK:
		return "ok"
	case ExitGeneralError:
		return "error"
	case ExitPanic:
		return "panic"
	case ExitConfigError:
		return "config_error"
	case ExitAuthError:
		return "auth_error"
	case ExitNotFound:
		return "not_found"
	case ExitConflict:
		return "conflict"
	case ExitValidation:
		return "validation_error"
	case ExitServerError:
		return "server_error"
	default:
		return "error"
	}
}

var reRequestFailed = regexp.MustCompile(`request failed \((\d{3})\)`)

// exitCodeFromError maps a CLI error to a specific exit code.
// If the error carries an embedded HTTP status code in the standard
// "request failed (NNN): ..." format, the code is mapped to a
// semantic exit constant. All other errors return ExitGeneralError.
func exitCodeFromError(err error) int {
	if err == nil {
		return ExitOK
	}

	msg := err.Error()

	// Detect HTTP status code in "request failed (NNN): ..." format.
	if m := reRequestFailed.FindStringSubmatch(msg); len(m) == 2 {
		status, _ := strconv.Atoi(m[1])
		switch {
		case status == 400 || status == 422:
			return ExitValidation
		case status == 401 || status == 403:
			return ExitAuthError
		case status == 404:
			return ExitNotFound
		case status == 409:
			return ExitConflict
		case status >= 500:
			return ExitServerError
		}
	}

	// Config / usage errors that do not originate from HTTP.
	configPhrases := []string{
		"project ID is required",
		"is required",
		"invalid",
		"must be",
		"non-interactive mode",
		"interactive prompt blocked",
		"no config",
		"API key",
		"server URL",
	}
	lower := strings.ToLower(msg)
	for _, phrase := range configPhrases {
		if strings.Contains(lower, strings.ToLower(phrase)) {
			return ExitConfigError
		}
	}

	return ExitGeneralError
}
