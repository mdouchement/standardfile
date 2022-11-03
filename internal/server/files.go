package server

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
)

type files struct{}

type Resource struct {
	RemoteIdentifier string `json:"remoteIdentifier"`
}

func respond(context echo.Context, code int, message string) error {
	success := code < 300
	return context.JSON(code, echo.Map{
		"success": success,
		"message": message,
	})
}

func readValetToken(context echo.Context) (*ValetToken, error) {
	valetTokenBase64 := context.Request().Header.Get("x-valet-token")
	valetTokenBytes, err := base64.StdEncoding.DecodeString(valetTokenBase64)
	if err != nil {
		return nil, errors.New("Unable to parse base64 valet token")
	}
	valetTokenJson := string(valetTokenBytes)
	var token ValetToken
	if err := json.Unmarshal([]byte(valetTokenJson), &token); err != nil {
		return nil, errors.New("Unable to parse json valet token")
	}
	return &token, nil
}

type ValetRequestParams struct {
	Operation string `json:"operation"`
	Resources []Resource
}

type ValetToken struct {
	Authorization string `json:"authorization"`
	FileID        string `json:"fileId"`
}

func (token *ValetToken) getFilePath() (*string, error) {
	id, err := uuid.FromString(token.FileID)
	if err != nil {
		return nil, errors.New("Unable to parse json valet token")
	} else if !fs.ValidPath(id.String()) {
		return nil, errors.New("Invalid path")
	}
	// TODO: Allow custom path in config
	// TODO: Subfolders for each user (Compatible format with official server)
	path := filepath.Join("etc", "standardfile", "database", id.String())
	return &path, nil
}

// Provides a valet token that is required to execute an operation
func (h *files) ValetTokens(c echo.Context) error {
	var params ValetRequestParams
	if err := c.Bind(&params); err != nil {
		return respond(c, http.StatusBadRequest, "Unable to parse request")
	} else if len(params.Resources) != 1 {
		return respond(c, http.StatusBadRequest, "Multi file requests unsupported")
	}

	// Generate valet token. Used for actual file operations
	var token ValetToken
	token.Authorization = c.Request().Header.Get(echo.HeaderAuthorization)
	token.FileID = params.Resources[0].RemoteIdentifier
	valetTokenJson, err := json.Marshal(token)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	return c.JSON(http.StatusOK, echo.Map{
		"success":    true,
		"valetToken": base64.StdEncoding.EncodeToString(valetTokenJson),
	})
}

// Called before uploading chunks of a file
func (h *files) CreateUploadSession(c echo.Context) error {
	token, err := readValetToken(c)
	if err != nil {
		return respond(c, http.StatusBadRequest, err.Error())
	}

	// Validate file path
	path, err := token.getFilePath()
	if err != nil {
		return respond(c, http.StatusBadRequest, err.Error())
	}

	// Create empty file
	if _, err := os.Create(*path); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusOK, echo.Map{
		"success":  true,
		"uploadId": token.FileID,
	})
}

// Called when uploaded all chunks of a file
func (h *files) CloseUploadSession(c echo.Context) error {
	token, err := readValetToken(c)
	if err != nil {
		return respond(c, http.StatusBadRequest, err.Error())
	}
	path, err := token.getFilePath()
	if err != nil {
		return respond(c, http.StatusBadRequest, err.Error())
	} else if _, err := os.Stat(*path); errors.Is(err, os.ErrNotExist) {
		return respond(c, http.StatusBadRequest, "File not created")
	}
	return respond(c, http.StatusOK, "File uploaded successfully")
}

// Upload parts of a file in an existing session
func (h *files) UploadChunk(c echo.Context) error {
	token, err := readValetToken(c)
	if err != nil {
		return respond(c, http.StatusBadRequest, err.Error())
	}

	// Validate file path
	path, err := token.getFilePath()
	if err != nil {
		return respond(c, http.StatusBadRequest, err.Error())
	} else if _, err := os.Stat(*path); errors.Is(err, os.ErrNotExist) {
		return c.JSON(http.StatusBadRequest, "File not created")
	}

	// Open file
	f, err := os.OpenFile(*path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return respond(c, http.StatusBadRequest, "Unable to open file")
	}
	defer f.Close()

	// Append chunk to file via buffer
	writer := bufio.NewWriter(f)
	reader := c.Request().Body
	if _, err := io.Copy(writer, reader); err != nil {
		return respond(c, http.StatusBadRequest, "Unable to store file")
	}
	return respond(c, http.StatusOK, "Chunk uploaded successfully")
}

// Delete file from server
func (h *files) Delete(c echo.Context) error {
	token, err := readValetToken(c)
	if err != nil {
		return respond(c, http.StatusBadRequest, err.Error())
	}
	path, err := token.getFilePath()
	if err != nil {
		return respond(c, http.StatusBadRequest, err.Error())
	} else if err := os.Remove(*path); err != nil {
		return respond(c, http.StatusBadRequest, "Unable to remove file")
	}
	return respond(c, http.StatusOK, "File removed successfully")
}

// Send encrypted file to client
func (h *files) Download(c echo.Context) error {
	token, err := readValetToken(c)
	if err != nil {
		return respond(c, http.StatusBadRequest, err.Error())
	}

	// Validate file path
	path, err := token.getFilePath()
	if err != nil {
		return respond(c, http.StatusBadRequest, err.Error())
	}
	return c.File(*path)
}
