package server

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mdouchement/standardfile/internal/database"
	"github.com/mdouchement/standardfile/internal/model"
	"github.com/mdouchement/standardfile/internal/server/middlewares"
	"github.com/mdouchement/standardfile/internal/server/session"
)

// A Controller is an Iversion Of Control pattern used to init the server package.
type Controller struct {
	Version         string
	Database        database.Client
	NoRegistration  bool
	ShowRealVersion bool
	// JWT params
	SigningKey []byte
	// Session params
	SessionSecret              []byte
	AccessTokenExpirationTime  time.Duration
	RefreshTokenExpirationTime time.Duration
}

// EchoEngine instantiates the wep server.
func EchoEngine(ctrl Controller) *echo.Echo {
	engine := echo.New()
	engine.Use(middleware.Recover())
	// engine.Use(middleware.CSRF()) // not supported by StandardNotes
	engine.Use(middleware.Secure())
	engine.Use(middleware.CORSWithConfig(middleware.DefaultCORSConfig))
	engine.Use(middleware.Gzip())

	engine.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "[${status}] ${method} ${uri} (${bytes_in}) ${latency_human}\n",
	}))
	engine.Binder = middlewares.NewBinder()
	// Error handler
	engine.HTTPErrorHandler = middlewares.HTTPErrorHandler

	engine.Pre(middleware.Rewrite(map[string]string{
		"/": "/version",
	}))

	////////////
	// Router //
	////////////

	sessions := session.NewManager(
		ctrl.Database,
		ctrl.SigningKey,
		ctrl.SessionSecret,
		ctrl.AccessTokenExpirationTime,
		ctrl.RefreshTokenExpirationTime,
	)

	router := engine.Group("")
	restricted := router.Group("")
	restricted.Use(middlewares.Session(sessions))

	v1 := router.Group("/v1")
	v1restricted := restricted.Group("/v1")

	// generic handlers
	//
	router.GET("/version", func(c echo.Context) error {
		version := "n/a"
		if ctrl.ShowRealVersion {
			version = ctrl.Version
		}

		return c.JSON(http.StatusOK, echo.Map{
			"version": version,
		})
	})

	//
	// auth handlers
	//
	auth := &auth{
		db:       ctrl.Database,
		sessions: sessions,
	}
	if !ctrl.NoRegistration {
		router.POST("/auth", auth.Register)

		v1.POST("/users", auth.Register)
	}
	router.GET("/auth/params", auth.Params) // Used for sign_in
	router.POST("/auth/sign_in", auth.Login)
	restricted.POST("/auth/sign_out", auth.Logout)
	restricted.POST("/auth/update", auth.Update)
	restricted.POST("/auth/change_pw", auth.UpdatePassword)
	v1restricted.PUT("/users/:id/attributes/credentials", auth.UpdatePassword)

	v1.GET("/login-params", auth.Params)
	v1.POST("/login", auth.Login)
	v1restricted.POST("/logout", auth.Logout)

	// TODO: GET    /auth/methods
	// TODO: GET    /v1/users/:id/params => currentuser auth.Params
	// TODO: PATCH  /v1/users/:id
	// TODO: PUT    /v1/users/:id/settings
	// TODO: DELETE /v1/users/:id/settings/:param

	//
	// session handlers
	//
	session := &sess{
		db:       ctrl.Database,
		sessions: sessions,
	}
	router.POST("/session/refresh", session.Refresh)
	restricted.GET("/sessions", session.List)
	restricted.DELETE("/session", session.Delete)
	restricted.DELETE("/session/all", session.DeleteAll)

	v1.POST("/sessions/refresh", session.Refresh)
	v1restricted.GET("/sessions", session.List)
	v1restricted.DELETE("/sessions/:id", session.Delete)
	v1restricted.DELETE("/sessions", session.DeleteAll)

	//
	// item handlers
	//
	item := &item{
		db: ctrl.Database,
	}
	restricted.POST("/items/sync", item.Sync)
	restricted.POST("/items/backup", item.Backup)
	restricted.DELETE("/items", item.Delete)

	v1restricted.POST("/items", item.Sync)

	v2 := router.Group("/v2")
	v2.POST("/login", auth.LoginPKCE)
	v2.POST("/login-params", auth.ParamsPKCE)
	//v2restricted := restricted.Group("/v2")

	return engine
}

// PrintRoutes prints the Echo engin exposed routes.
func PrintRoutes(e *echo.Echo) {
	ignored := map[string]bool{
		"":      true,
		".":     true,
		"/*":    true,
		"/v1":   true,
		"/v1/*": true,
		"/v2":   true,
		"/v2/*": true,
	}

	routes := e.Routes()
	sort.Slice(routes, func(i int, j int) bool {
		return routes[i].Path < routes[j].Path
	})

	fmt.Println("Routes:")
	for _, route := range routes {
		if ignored[route.Path] {
			continue
		}
		fmt.Printf("%6s %s\n", route.Method, route.Path)
	}
}

func currentUser(c echo.Context) *model.User {
	user, ok := c.Get(middlewares.CurrentUserContextKey).(*model.User)
	if ok {
		return user
	}
	return nil
}

func currentSession(c echo.Context) *model.Session {
	session, ok := c.Get(middlewares.CurrentSessionContextKey).(*model.Session)
	if ok {
		return session
	}
	return nil
}
