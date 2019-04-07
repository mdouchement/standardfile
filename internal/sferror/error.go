package sferror

type (
	// An SFError represents the error format that can be rendered by stnadardfile server.
	SFError struct {
		FieldError err `json:"error"`
	}

	err struct {
		Message string `json:"message"`
	}
)

// New returns a new SFError with the given message.
func New(message string) *SFError {
	return &SFError{FieldError: err{Message: message}}
}

// Error implements error interface.
func (e *SFError) Error() string {
	return e.FieldError.Message
}
