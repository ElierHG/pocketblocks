package apis

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pedrozadotdev/pocketblocks/server/daos"
	"github.com/pedrozadotdev/pocketblocks/server/forms"
	"github.com/pedrozadotdev/pocketblocks/server/models"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/tools/search"
)

// BindConnectionApi registers the CRUD endpoints for connections.
func BindConnectionApi(dao *daos.Dao, g *echo.Group, logMiddleware echo.MiddlewareFunc) {
	api := connectionApi{dao: dao}

	sub := g.Group("/connections")
	sub.GET("", api.list, apis.RequireAdminOrRecordAuth("users"))
	sub.POST("", api.create, apis.RequireAdminAuth(), logMiddleware)
	sub.PATCH("/:id", api.update, apis.RequireAdminAuth(), logMiddleware)
	sub.DELETE("/:id", api.delete, apis.RequireAdminAuth(), logMiddleware)
}

type connectionApi struct {
	dao *daos.Dao
}

func (api *connectionApi) list(c echo.Context) error {
	fieldResolver := search.NewSimpleFieldResolver(
		"id", "name", "type", "config", "created", "updated",
	)

	connections := []*models.Connection{}

	result, err := search.NewProvider(fieldResolver).
		Query(api.dao.PblConnectionQuery()).
		ParseAndExec(c.QueryParams().Encode(), &connections)
	if err != nil {
		return apis.NewBadRequestError("", err)
	}

	return c.JSON(http.StatusOK, result)
}

func (api *connectionApi) create(c echo.Context) error {
	form := forms.NewConnectionUpsert(api.dao, &models.Connection{})
	if err := c.Bind(form); err != nil {
		return apis.NewBadRequestError("Failed to load the submitted data due to invalid formatting.", err)
	}
	conn, err := form.Submit()
	if err != nil {
		return apis.NewBadRequestError("Failed to load the submitted data. Try again later.", err)
	}
	return c.JSON(http.StatusOK, conn)
}

func (api *connectionApi) update(c echo.Context) error {
	id := c.PathParam("id")
	if id == "" {
		return apis.NewNotFoundError("", nil)
	}
	conn, err := api.dao.FindPblConnectionById(id)
	if err != nil || conn == nil {
		return apis.NewNotFoundError("", err)
	}
	form := forms.NewConnectionUpsert(api.dao, conn)
	if err := c.Bind(form); err != nil {
		return apis.NewBadRequestError("Failed to load the submitted data due to invalid formatting.", err)
	}
	updated, err := form.Submit()
	if err != nil {
		return apis.NewBadRequestError("Failed to load the submitted data. Try again later.", err)
	}
	return c.JSON(http.StatusOK, updated)
}

func (api *connectionApi) delete(c echo.Context) error {
	id := c.PathParam("id")
	if id == "" {
		return apis.NewNotFoundError("", nil)
	}
	conn, err := api.dao.FindPblConnectionById(id)
	if err != nil || conn == nil {
		return apis.NewNotFoundError("", err)
	}
	if err := api.dao.DeletePblConnection(conn); err != nil {
		return apis.NewBadRequestError("Failed to delete connection.", err)
	}
	return c.NoContent(http.StatusNoContent)
}
