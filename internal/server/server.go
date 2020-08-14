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

// An IOC is an Iversion Of Control pattern used to init the server package.
type IOC struct {
	Version        string
	Database       database.Client
	NoRegistration bool
	// JWT params
	SigningKey []byte
	// Session params
	AccessTokenExpirationTime  time.Duration
	RefreshTokenExpirationTime time.Duration
}

// EchoEngine instantiates the wep server.
func EchoEngine(ctrl IOC) *echo.Echo {
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
		ctrl.AccessTokenExpirationTime,
		ctrl.RefreshTokenExpirationTime,
	)

	router := engine.Group("")
	restricted := router.Group("")
	restricted.Use(middlewares.Session(sessions))

	// generic handlers
	//
	router.GET("/version", func(c echo.Context) error {
		return c.JSON(http.StatusOK, echo.Map{
			"version": ctrl.Version,
		})
	})

	//
	// auth handlers
	//
	auth := &auth{
		db:         ctrl.Database,
		signingKey: ctrl.SigningKey,
	}
	if !ctrl.NoRegistration {
		router.POST("/auth", auth.Register)
	}
	router.GET("/auth/params", auth.Params) // Used for sign_in
	router.POST("/auth/sign_in", auth.Login)
	restricted.POST("/auth/update", auth.Update)
	restricted.POST("/auth/change_pw", auth.UpdatePassword)

	//
	// session handlers
	//
	session := &sess{
		db: ctrl.Database,
		m: session.NewManager(
			ctrl.Database,
			ctrl.SigningKey,
			ctrl.AccessTokenExpirationTime,
			ctrl.RefreshTokenExpirationTime,
		),
	}
	restricted.POST("/session/refresh", session.Refresh)
	restricted.GET("/sessions", session.List)
	restricted.DELETE("/session", session.Delete)
	restricted.DELETE("/session/all", session.DeleteAll)

	//
	// item handlers
	//
	item := &item{
		db: ctrl.Database,
	}
	restricted.POST("/items/sync", item.Sync)
	restricted.POST("/items/backup", item.Backup)
	restricted.DELETE("/items", item.Delete)

	return engine
}

// PrintRoutes prints the Echo engin exposed routes.
func PrintRoutes(e *echo.Echo) {
	ignored := map[string]bool{
		"":   true,
		".":  true,
		"/*": true,
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
