package server

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

type files struct{}

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
	// TODO: Check format of fileId (Security)
	// TODO: Allow custom path in config
	// TODO: Subfolders for each user (Compatible format with official server)
	return "/etc/standardfile/database/" + token.FileId
}

// Provides a valet token that is required to execute an operation
func (h *files) ValetTokens(c echo.Context) error {
	var params ValetRequestParams
	if err := c.Bind(&params); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if len(params.Resources) != 1 {
		return c.JSON(http.StatusBadRequest, "Multi file requests not supported")
	}

	var token ValetToken
	token.Authorization = c.Request().Header.Get(echo.HeaderAuthorization)
	token.FileId = params.Resources[0].RemoteIdentifier
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
}

// Called when all chunks a file uploaded
func (h *files) CloseUploadSession(c echo.Context) error {
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
}

// Upload parts of a file in an existing session
func (h *files) UploadChunk(c echo.Context) error {
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
	defer f.Close()

	// Create new buffer
	writer := bufio.NewWriter(f)
	reader := c.Request().Body
	io.Copy(writer, reader)

	return c.JSON(http.StatusOK, echo.Map{
		"success": true,
		"message": "Chunk uploaded successfully",
	})
}

// Delete file from server
func (h *files) Delete(c echo.Context) error {
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
}

// Send encrypted file to client
func (h *files) Download(c echo.Context) error {
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
}
