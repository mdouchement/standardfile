package middlewares

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mdouchement/standardfile/internal/server/session"
)

// CurrentUserContextKey is the key to retrieve the current_user from echo.Context.
const CurrentUserContextKey = "current_user"

// Session returns a Session auth middleware.
// It also handle JWT tokens from previous API versions.
// It stores current_user into echo.Context
func Session(m session.Manager) echo.MiddlewareFunc {
	jwt := middleware.JWTWithConfig(middleware.JWTConfig{
		SigningKey: m.JWTSigningKey(),
	})
	fake := func(echo.Context) error {
		return nil
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			authorization := c.Request().Header.Get(echo.HeaderAuthorization)

			//
			// JWT
			//

			if strings.Count(authorization, ".") == 2 {
				err = jwt(fake)(c)
				if err != nil {
					return err
				}

				user, err := m.UserFromToken(c.Get(middleware.DefaultJWTConfig.ContextKey))
				if err != nil {
					return err
				}

				// TODO: Find a way to extract `api` (apiversion) from the requests body.
				// if apiversion >= 20190520 && session.UserSupportsSessions(user) {
				// 	return c.JSON(http.StatusUnauthorized, echo.Map{
				// 		"error": echo.Map{
				// 			"tag":     "invalid-auth",
				// 			"message": "Invalid login credentials.",
				// 		},
				// 	})
				// }

				// Store current_user for handlers.
				c.Set(CurrentUserContextKey, user)
				return next(c)
			}

			//
			// Session
			//

			token := token(authorization)
			if token == "" {
				return c.JSON(http.StatusUnauthorized, echo.Map{
					"error": echo.Map{
						"tag":     "invalid-auth",
						"message": "Invalid login credentials.",
					},
				})
			}

			user, err := m.UserFromToken(token)
			if err != nil {
				return err
			}

			// Store current_user for handlers.
			c.Set(CurrentUserContextKey, user)
			return next(c)
		}
	}
}

func token(authorization string) string {
	parts := strings.Split(authorization, " ")
	if strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}
