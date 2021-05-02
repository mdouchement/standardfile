package server_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/appleboy/gofight/v2"
	"github.com/gofrs/uuid"
	"github.com/mdouchement/standardfile/internal/model"
	"github.com/mdouchement/standardfile/internal/server"
	"github.com/mdouchement/standardfile/internal/server/service"
	"github.com/mdouchement/standardfile/pkg/libsf"
	"github.com/stretchr/testify/assert"
)

type sync20161215 struct {
	Retrieved     []*model.Item          `json:"retrieved_items"`
	Saved         []*model.Item          `json:"saved_items"`
	Unsaved       []*service.UnsavedItem `json:"unsaved"`
	SyncToken     string                 `json:"sync_token"`
	CursorToken   string                 `json:"cursor_token"`
	IntegrityHash string                 `json:"integrity_hash,omitempty"`
}

func TestRequestItemsSync20161215(t *testing.T) {
	engine, ctrl, r, cleanup := setup()
	defer cleanup()

	r.POST("/items/sync").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})

	user := createUser(ctrl)
	header := gofight.H{
		"Authorization": "Bearer " + server.CreateJWT(ctrl, user),
	}

	r.POST("/items/sync").SetHeader(header).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.JSONEq(t, `{"error":{"message":"Could not get syncing params."}}`, r.Body.String())
	})

	//
	//

	item := &model.Item{
		Base: model.Base{
			ID: "d989ccc9-15c6-475e-839b-1690bd07d073",
		},
		UserID:           "b329a187-ddf8-4e9b-960d-49c272a58794",
		Content:          "003:d83ea9b696c313c8a352795264873fefebbd60a92c5d9a89e3a380a0d3a68b62:d989ccc9-15c6-475e-839b-1690bd07d073:400591db0ad08c0847f45a1e76ceb87d:HI0PGPWB667YzIlWPR4A8VpGuDb9YOcTRknFb4CZXM2yJ0KK68W2giX6AV6KNV19exUvnunTmkfxlOWUfiG2m7YU2rIO76MMMfs5wqBKqO4eTuootiYVi5JbCW2BFHJcDnj3seb8juBV95Bm5lm4tQ==:eyJpZGVudGlmaWVyIjoiZ2VvcmdlLmFiaXRib2xAbm93aGVyZS5sYW4iLCJ2ZXJzaW9uIjoiMDAzIiwicHdfY29zdCI6NDIwMDAwLCJwd19ub25jZSI6Im5vbmNlIn0=",
		ContentType:      libsf.ContentTypeNote,
		EncryptedItemKey: "003:3c69d9526d2846671c7e8cf89968f3b6ffd92e82ca15b04d29a3f77100ce857c:d989ccc9-15c6-475e-839b-1690bd07d073:93b257d16f53732d81230e41b62eab7c:Ai0xyC1CFcah3/rubAXV+j433oXoABPU8kmYdAzE1WlscKQIXbds8USDG0HmoC1XkCHerozTcJc5IgTAN2JZZBYttmllRswgpn7vDKZIUbXa/FDao3l6a43hedxIfd+4b1moSnB1IgG/T8c+WoA0zDd5vKtB5EMyljLVbyItBZnNrg7toV1bSWQ1t+8xUcKm:eyJpZGVudGlmaWVyIjoiZ2VvcmdlLmFiaXRib2xAbm93aGVyZS5sYW4iLCJ2ZXJzaW9uIjoiMDAzIiwicHdfY29zdCI6NDIwMDAwLCJwd19ub25jZSI6Im5vbmNlIn0=",
		Deleted:          false,
	}
	err := ctrl.Database.Save(item)
	if err != nil {
		panic(err)
	}

	//
	//

	params := gofight.D{
		"compute_integrity": false,
		"limit":             100000,
		"sync_token":        "",
		"cursor_token":      "",
		"items":             []*model.Item{},
	}

	r.POST("/items/sync").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		var v sync20161215
		err := json.Unmarshal(r.Body.Bytes(), &v)
		assert.NoError(t, err)

		at := libsf.TimeFromToken(v.SyncToken)
		assert.NotZero(t, at)
		assert.WithinDuration(t, time.Now(), at, 2*time.Second)

		assert.Empty(t, v.Retrieved) // Nothing for this user
		assert.Empty(t, v.Saved)
		assert.Empty(t, v.Unsaved)
	})

	//
	//

	item.UserID = user.ID
	err = ctrl.Database.Save(item)
	if err != nil {
		panic(err)
	}

	//
	//

	r.POST("/items/sync").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		var v sync20161215
		err := json.Unmarshal(r.Body.Bytes(), &v)
		assert.NoError(t, err)

		at := libsf.TimeFromToken(v.SyncToken)
		assert.NotZero(t, at)
		assert.WithinDuration(t, time.Now(), at, 2*time.Second)

		assert.Len(t, v.Retrieved, 1)
		assert.Empty(t, v.Saved)
		assert.Empty(t, v.Unsaved)
	})

	item.SetID(uuid.Must(uuid.NewV4()).String())
	params["items"] = []*model.Item{item}
	r.POST("/items/sync").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		var v sync20161215
		err := json.Unmarshal(r.Body.Bytes(), &v)
		assert.NoError(t, err)

		at := libsf.TimeFromToken(v.SyncToken)
		assert.NotZero(t, at)
		assert.WithinDuration(t, time.Now(), at, 2*time.Second)

		assert.Len(t, v.Retrieved, 1)
		assert.Len(t, v.Saved, 1)
		assert.Empty(t, v.Unsaved)
	})

	params["items"] = []*model.Item{}
	r.POST("/items/sync").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		var v sync20161215
		err := json.Unmarshal(r.Body.Bytes(), &v)
		assert.NoError(t, err)

		at := libsf.TimeFromToken(v.SyncToken)
		assert.NotZero(t, at)
		assert.WithinDuration(t, time.Now(), at, 2*time.Second)

		assert.Len(t, v.Retrieved, 2)
		assert.Empty(t, v.Saved)
		assert.Empty(t, v.Unsaved)
	})
}
