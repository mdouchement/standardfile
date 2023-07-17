package middlewares

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type binder struct {
	echo.DefaultBinder
	methodsWithBody map[string]bool
}

// NewBinder returns a wrapp of the default binder implementation with extra checks.
func NewBinder() echo.Binder {
	return &binder{
		methodsWithBody: map[string]bool{
			http.MethodPost:  true,
			http.MethodPatch: true,
			http.MethodPut:   true,
		},
	}
}

// Bind implements the echo.Bind interface.
func (b *binder) Bind(i any, c echo.Context) (err error) {
	if c.Request().ContentLength == 0 && b.methodsWithBody[c.Request().Method] {
		return echo.NewHTTPError(http.StatusBadRequest, "Request body can't be empty")
	}

	if c.Request().Header.Get("Content-Type") == "" {
		c.Request().Header.Set("Content-Type", "application/json")
	}

	return b.DefaultBinder.Bind(i, c)
}
