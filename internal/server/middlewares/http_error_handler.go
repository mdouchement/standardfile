package middlewares

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v5"
	"github.com/mdouchement/standardfile/internal/sferror"
)

// HTTPErrorHandler is a middleware that formats rendered errors.
func HTTPErrorHandler(c *echo.Context, err error) {
	resp, uerr := echo.UnwrapResponse(c.Response())
	if uerr != nil {
		log.Printf("Error [unwrap_response]: %s", err.Error())
		return
	}

	if resp.Committed {
		// Cannot be changed
		return
	}

	switch err := err.(type) {
	case *echo.HTTPError:
		log.Printf("Error [ECHO]: %s", err.Unwrap())
		_ = c.JSON(err.Code, map[string]any{
			"error": map[string]any{
				"message": err.Message,
			},
		})
	case *sferror.SFError:
		status := sferror.StatusCode(err)
		if status < 500 {
			_ = c.JSON(status, err)
			return
		}

		internal(err, c)
	default:
		internal(err, c)
	}
}

func internal(err error, c *echo.Context) {
	id := uuid.Must(uuid.NewV4()).String()
	log.Printf("Error [%s]: %s", id, err.Error())

	_ = c.JSON(http.StatusInternalServerError, map[string]any{
		"error": map[string]any{
			"message": fmt.Sprintf("Unexpected error (id: %s)", id),
		},
	})
}
