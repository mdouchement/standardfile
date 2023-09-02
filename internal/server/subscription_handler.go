package server

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/valyala/fastjson"
)

type subscription struct {
	SubscriptionPayload []byte
	FeaturesPayload     []byte
}

func (h *subscription) SubscriptionV1(c echo.Context) error {
	user := currentUser(c)

	// The official Standard Notes client has a race condition,
	// the features endpoint will only be called when delaying response...
	time.Sleep(1 * time.Second)

	// Overrides some fields of the raw payload to match the current user.
	v, err := fastjson.ParseBytes(h.SubscriptionPayload)
	if err != nil {
		return err
	}
	v.Get("meta", "auth").Set("userUuid", new(fastjson.Arena).NewString(user.ID))
	v.Get("data", "user").Set("uuid", new(fastjson.Arena).NewString(user.ID))
	v.Get("data", "user").Set("email", new(fastjson.Arena).NewString(user.Email))

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	return c.String(http.StatusOK, v.String())
}

func (h *subscription) Features(c echo.Context) error {
	user := currentUser(c)

	// Overrides some fields of the raw payload to match the current user.
	v, err := fastjson.ParseBytes(h.FeaturesPayload)
	if err != nil {
		return err
	}
	v.Get("meta", "auth").Set("userUuid", new(fastjson.Arena).NewString(user.ID))
	v.Get("data").Set("userUuid", new(fastjson.Arena).NewString(user.ID))

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	return c.String(http.StatusOK, v.String())
}
