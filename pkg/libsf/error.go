package libsf

import (
	"encoding/json"
	"io"
)

// An SFError reprensents an HTTP error returned by StandardFile server.
type SFError struct {
	StatusCode int
	Err        struct {
		Message string `json:"message"`
	} `json:"error"`
}

func parseSFError(r io.Reader, code int) error {
	var sferr SFError
	dec := json.NewDecoder(r)
	if err := dec.Decode(&sferr); err != nil {
		return err
	}
	sferr.StatusCode = code
	return &sferr
}

func (e *SFError) Error() string {
	return e.Err.Message
}
