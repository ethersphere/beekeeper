package api

import (
	"errors"
	"fmt"
	"net/http"
)

// HTTPStatusError represents the error derived from the HTTP response status
// code.
type HTTPStatusError struct {
	Code int
}

// NewHTTPStatusError creates a new instance of HTTPStatusError based on the
// provided code.
func NewHTTPStatusError(code int) *HTTPStatusError {
	return &HTTPStatusError{
		Code: code,
	}
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("%d %s", e.Code, http.StatusText(e.Code))
}

// IsHTTPStatusErrorCode return whether the error is HTTPStatusError with a
// specific HTTP status code.
func IsHTTPStatusErrorCode(err error, code int) bool {
	var e *HTTPStatusError
	if errors.As(err, &e) {
		return e.Code == code
	}
	return false
}
