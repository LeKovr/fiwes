package upload

// HTTPError holds error with HTTP status
type HTTPError struct {
	error
	status int
}

// NewHTTPError returns new HTTPError
func NewHTTPError(status int, err error) *HTTPError {
	return &HTTPError{status: status, error: err}
}

// Status returns HTTPError status
func (e HTTPError) Status() int {
	return e.status
}
