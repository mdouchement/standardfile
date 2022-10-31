package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mdouchement/standardfile/internal/database"
	"github.com/mdouchement/standardfile/internal/server/service"
	"github.com/mdouchement/standardfile/internal/sferror"
)

// item contains all item handlers.
type item struct {
	db database.Client
}

///// Sync
////
//

// Sync used for saves local changes as well as retrieves remote changes.
// https://standardfile.org/#post-items-sync
func (h *item) Sync(c echo.Context) error {
	// Filter params
	var params service.SyncParams
	if err := c.Bind(&params); err != nil {
		return c.JSON(http.StatusBadRequest, sferror.New("Could not get syncing params."))
	}
	params.UserAgent = c.Request().UserAgent()
	params.Session = currentSession(c)

	sync := service.NewSync(h.db, currentUser(c), params)
	if err := sync.Execute(); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, sync)
}

///// Backup
////
//

// Backup used for writes all user data to backup extension.
// This is called when a new extension is registered.
func (h *item) Backup(c echo.Context) error {
	// In reference implementation, there is post_to_extension but not implemented here.
	// See README.md
	return c.NoContent(http.StatusOK)
}

///// Delete
////
//

// Delete used for remove all defined items.
func (h *item) Delete(c echo.Context) error {
	// user := currentUser(c)
	// https://github.com/standardfile/ruby-server/blob/master/app/controllers/api/items_controller.rb#L72-L76

	// TODO undocumented feature and seems not used by official client.

	return c.NoContent(http.StatusNoContent)
}
