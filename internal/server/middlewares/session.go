package middlewares

import (
	"net/http"
	"strings"

	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mdouchement/middlewarex"
	"github.com/mdouchement/standardfile/internal/server/session"
	"github.com/o1egl/paseto/v2"
)

const (
	// CurrentUserContextKey is the key to retrieve the current_user from echo.Context.
	CurrentUserContextKey = "current_user"
	// CurrentSessionContextKey is the key to retrieve the current_session from echo.Context.
	CurrentSessionContextKey = "current_session"
)

// Session returns a Session auth middleware.
// It also handle JWT tokens from previous API versions.
// It stores current_user into echo.Context
func Session(m session.Manager) echo.MiddlewareFunc {
	jwt := echojwt.JWT(m.JWTSigningKey())
	paseto := middlewarex.PASETOWithConfig(middlewarex.PASETOConfig{
		SigningKey: m.SessionSecret(),
		Validators: []paseto.Validator{
			paseto.IssuedBy("standardfile"),
			paseto.ForAudience(session.TypeAccessToken),
		},
	})

	fake := func(echo.Context) error {
		return nil
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			authorization := c.Request().Header.Get(echo.HeaderAuthorization)
			token := token(authorization)

			if token == "" {
				return c.JSON(http.StatusUnauthorized, echo.Map{
					"error": echo.Map{
						"tag":     "invalid-auth",
						"message": "Invalid login credentials.",
					},
				})
			}

			//
			// Session
			//

			if strings.HasPrefix(token, "v2.local.") {
				err = paseto(fake)(c) // Check PASETO validity according its claims.
				if err != nil && !strings.Contains(err.Error(), "token has expired: token validation error") {
					// Token is not valid.
					// We do not catch token expiration here and let the session manager performs its validation.
					return c.JSON(http.StatusUnauthorized, echo.Map{
						"error": echo.Map{
							"tag":     "invalid-auth",
							"message": "Invalid login credentials.",
						},
					})
				}

				tk := c.Get(middlewarex.DefaultPASETOConfig.ContextKey).(middlewarex.Token)

				// Find, validate and store current_session for handlers.
				session, err := m.Validate(tk.Subject, tk.Jti)
				if err != nil {
					return err
				}

				c.Set(CurrentSessionContextKey, session)

				// Find and store current_user for handlers.
				user, err := m.UserFromToken(tk)
				if err != nil {
					return err
				}
				c.Set(CurrentUserContextKey, user)

				// TODO: Find a way to extract `api` (apiversion) from the requests body.
				// Revoke old JWT.
				// if apiversion >= 20190520 && session.UserSupportsSessions(user) {
				// 	return c.JSON(http.StatusUnauthorized, echo.Map{
				// 		"error": echo.Map{
				// 			"tag":     "invalid-auth",
				// 			"message": "Invalid login credentials.",
				// 		},
				// 	})
				// }

				return next(c)
			}

			//
			// JWT
			//

			err = jwt(fake)(c) // Check JWT validity according its claims.
			if err != nil {
				return c.JSON(http.StatusUnauthorized, echo.Map{
					"error": echo.Map{
						"tag":     "invalid-auth",
						"message": "Invalid login credentials.",
					},
				})
			}

			user, err := m.UserFromToken(c.Get(middleware.DefaultJWTConfig.ContextKey))
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
