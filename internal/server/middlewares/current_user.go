package middlewares

import (
	"encoding/json"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mdouchement/standardfile/internal/database"
	"github.com/pkg/errors"
)

// CurrentUserContextKey is the key to retrieve the current_user from echo.Context.
const CurrentUserContextKey = "current_user"

// CurrentUser checks current_user based on JWT and store it into echo.Context.
func CurrentUser(db database.Client) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			token, ok := c.Get(middleware.DefaultJWTConfig.ContextKey).(*jwt.Token)
			if !ok {
				panic("token implementation has changed")
			}
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				panic("token implementation has wrong type of claims")
			}

			// Get current_user.
			user, err := db.FindUser(claims["user_uuid"].(string))
			if err != nil {
				if db.IsNotFound(err) {
					return c.JSON(http.StatusUnauthorized, echo.Map{
						"error": echo.Map{
							"tag":     "invalid-auth",
							"message": "No such user for given token.",
						},
					})
				}
				return errors.Wrap(err, "could not get access to database")
			}

			// Check if password has changed since token was generated.
			var iat int64
			switch v := claims["iat"].(type) {
			case float64:
				iat = int64(v)
			case json.Number:
				iat, _ = v.Int64()
			default:
				panic("unsuported iat underlying type")
			}

			if iat < user.PasswordUpdatedAt {
				return c.JSON(http.StatusUnauthorized, echo.Map{
					"error": echo.Map{
						"tag":     "invalid-auth",
						"message": "Revoked token.",
					},
				})
			}

			// Store current_user for handlers.
			c.Set(CurrentUserContextKey, user)

			if err = next(c); err != nil {
				c.Error(err)
			}

			return nil
		}
	}
}
