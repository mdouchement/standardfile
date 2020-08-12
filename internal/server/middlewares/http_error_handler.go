package middlewares

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/mdouchement/standardfile/internal/sferror"
)

// HTTPErrorHandler is a middleware that formats rendered errors.
func HTTPErrorHandler(err error, c echo.Context) {
	if !c.Response().Committed {
		switch err := err.(type) {
		case *echo.HTTPError:
			log.Printf("Error [ECHO]: %s", err.Internal)
			_ = c.JSON(err.Code, echo.Map{
				"error": echo.Map{
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
}

func internal(err error, c echo.Context) {
	id := uuid.Must(uuid.NewV4()).String()
	log.Printf("Error [%s]: %s", id, err.Error())

	_ = c.JSON(http.StatusInternalServerError, echo.Map{
		"error": echo.Map{
			"message": fmt.Sprintf("Unexpected error (id: %s)", id),
		},
	})
}
