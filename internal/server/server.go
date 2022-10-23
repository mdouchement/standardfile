package server

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"time"

	"encoding/base64"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mdouchement/standardfile/internal/database"
	"github.com/mdouchement/standardfile/internal/model"
	"github.com/mdouchement/standardfile/internal/server/middlewares"
	"github.com/mdouchement/standardfile/internal/server/session"
)

// A Controller is an Iversion Of Control pattern used to init the server package.
type Controller struct {
	Version            string
	Database           database.Client
	NoRegistration     bool
	ShowRealVersion    bool
	EnableSubscription bool
	// JWT params
	SigningKey []byte
	// Session params
	SessionSecret              []byte
	AccessTokenExpirationTime  time.Duration
	RefreshTokenExpirationTime time.Duration
}

type Resource struct {
	RemoteIdentifier string `json:"remoteIdentifier"`
}

type ValetRequestParams struct {
	Operation string `json:"operation"`
	Resources []Resource
}

type ValetToken struct {
	Authorization string `json:"authorization"`
	FileId        string `json:"fileId"`
}

func (token *ValetToken) GetFilePath() string {
	// TODO check format of fileID
	return "/etc/standardfile/database/" + token.FileId
}

// EchoEngine instantiates the wep server.
func EchoEngine(ctrl Controller) *echo.Echo {
	engine := echo.New()
	engine.Use(middleware.Recover())
	// engine.Use(middleware.CSRF()) // not supported by StandardNotes
	engine.Use(middleware.Secure())

	// Expose headers for file download
	cors := middleware.DefaultCORSConfig
	cors.ExposeHeaders = append(cors.ExposeHeaders, "Content-Range", "Accept-Ranges")
	engine.Use(middleware.CORSWithConfig(cors))

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

	//
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

	v1.GET("/login-params", auth.Params)
	v1.POST("/login", auth.Login)
	v1restricted.POST("/logout", auth.Logout)
	v1restricted.PUT("/users/:id/attributes/credentials", auth.UpdatePassword)

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

	//
	// files
	//
	v1restricted.POST("/files/valet-tokens", func(c echo.Context) error {
		var params ValetRequestParams
		if err := c.Bind(&params); err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}

		if len(params.Resources) != 1 {
			return c.JSON(http.StatusBadRequest, "Multi file requests not supported")
		}

		// {"operation":"write","resources":[{"remoteIdentifier":"2ef2a4af-2a3c-41ac-b409-78471e6f4a81","unencryptedFileSize":3427}]}
		// {"operation":"delete","resources":[{"remoteIdentifier":"b0383bfa-8d9f-4023-8aa1-5c9e3011a0ef","unencryptedFileSize":0}]}
		// {"operation":"read","resources":[{"remoteIdentifier":"2ef2a4af-2a3c-41ac-b409-78471e6f4a81","unencryptedFileSize":0}]}

		var token ValetToken
		token.Authorization = c.Request().Header.Get(echo.HeaderAuthorization)
		token.FileId = params.Resources[0].RemoteIdentifier
		valetTokenJson, err := json.Marshal(token)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}

		// token := auth + " " + fileId
		return c.JSON(http.StatusOK, echo.Map{
			"success":    true,
			"valetToken": base64.StdEncoding.EncodeToString(valetTokenJson),
		})
	})
	v1.POST("/files/upload/create-session", func(c echo.Context) error {
		valetTokenBase64 := c.Request().Header.Get("x-valet-token")
		valetTokenBytes, err := base64.StdEncoding.DecodeString(valetTokenBase64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}
		valetTokenJson := string(valetTokenBytes)

		var token ValetToken
		if err := json.Unmarshal([]byte(valetTokenJson), &token); err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}

		fmt.Println("create-session. valet_token: " + valetTokenJson)

		if _, err := os.Create(token.GetFilePath()); err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}

		return c.JSON(http.StatusOK, echo.Map{
			"success":  true,
			"uploadId": token.FileId,
		})
	})
	v1.POST("/files/upload/close-session", func(c echo.Context) error {
		valetTokenBase64 := c.Request().Header.Get("x-valet-token")
		valetTokenBytes, err := base64.StdEncoding.DecodeString(valetTokenBase64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}
		valetTokenJson := string(valetTokenBytes)
		var token ValetToken
		if err := json.Unmarshal([]byte(valetTokenJson), &token); err != nil {
			return c.JSON(http.StatusBadRequest, err)
		} else if _, err := os.Stat(token.GetFilePath()); errors.Is(err, os.ErrNotExist) {
			return c.JSON(http.StatusBadRequest, "File not created")
		}

		fmt.Println("close-session. valet_token: " + valetTokenJson)
		return c.JSON(http.StatusOK, echo.Map{
			"success": true,
			"message": "File uploaded successfully",
		})
	})
	v1.POST("/files/upload/chunk", func(c echo.Context) error {
		valetTokenBase64 := c.Request().Header.Get("x-valet-token")
		valetTokenBytes, err := base64.StdEncoding.DecodeString(valetTokenBase64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}
		valetTokenJson := string(valetTokenBytes)
		var token ValetToken
		if err := json.Unmarshal([]byte(valetTokenJson), &token); err != nil {
			return c.JSON(http.StatusBadRequest, err)
		} else if _, err := os.Stat(token.GetFilePath()); errors.Is(err, os.ErrNotExist) {
			return c.JSON(http.StatusBadRequest, "File not created")
		}

		chunk_id := c.Request().Header.Get("x-chunk-id")
		fmt.Println("chunk. valet_token: " + valetTokenJson + " chunk_id: " + chunk_id)

		f, err := os.OpenFile(token.GetFilePath(), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}
		// remember to close the file
		defer f.Close()

		// create new buffer
		writer := bufio.NewWriter(f)
		reader := c.Request().Body
		io.Copy(writer, reader)

		return c.JSON(http.StatusOK, echo.Map{
			"success": true,
			"message": "Chunk uploaded successfully",
		})
	})
	v1.DELETE("/files", func(c echo.Context) error {
		valetTokenBase64 := c.Request().Header.Get("x-valet-token")
		valetTokenBytes, err := base64.StdEncoding.DecodeString(valetTokenBase64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}
		valetTokenJson := string(valetTokenBytes)
		var token ValetToken
		if err := json.Unmarshal([]byte(valetTokenJson), &token); err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}
		err = os.Remove(token.GetFilePath())
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}
		return c.JSON(http.StatusOK, echo.Map{
			"success": true,
			"message": "File removed successfully",
		})
	})
	v1.GET("/files", func(c echo.Context) error {
		valetTokenBase64 := c.Request().Header.Get("x-valet-token")
		valetTokenBytes, err := base64.StdEncoding.DecodeString(valetTokenBase64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}
		valetTokenJson := string(valetTokenBytes)
		var token ValetToken
		if err := json.Unmarshal([]byte(valetTokenJson), &token); err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}
		return c.File(token.GetFilePath())
	})

	//
	// subscription handlers
	//
	if ctrl.EnableSubscription {
		subscription := &subscription{}
		router.GET("/v2/subscriptions", func(c echo.Context) error {
			return c.HTML(http.StatusInternalServerError, "getaddrinfo EAI_AGAIN payments")
		})
		v1restricted.GET("/users/:id/subscription", subscription.SubscriptionV1)
		v1restricted.GET("/users/:id/features", subscription.Features)
	}

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
