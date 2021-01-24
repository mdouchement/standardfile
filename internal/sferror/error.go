package sferror

import "net/http"

// StatusExpiredAccessToken is an HTTP status code used when an access token is expired.
const StatusExpiredAccessToken = 498

type (
	// An SFError represents the error format that can be rendered by stnadardfile server.
	SFError struct {
		HTTPCode   int `json:"-"`
		FieldError err `json:"error"`
	}

	err struct {
		Tag     string `json:"tag,omitempty"`
		Message string `json:"message"`
	}
)

// StatusCode returns the HTTP status code.
func StatusCode(err error) int {
	if sferr, ok := err.(*SFError); ok {
		return sferr.HTTPCode
	}
	return http.StatusInternalServerError
}

// New returns a new SFError with the given message.
func New(message string) *SFError {
	return &SFError{FieldError: err{Message: message}}
}

// NewWithTagCode returns a new SFError with the given code, tag and message.
func NewWithTagCode(code int, tag, message string) *SFError {
	return &SFError{HTTPCode: code, FieldError: err{Tag: tag, Message: message}}
}

// Error implements error interface.
func (e *SFError) Error() string {
	return e.FieldError.Message
}
