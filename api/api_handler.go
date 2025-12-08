package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sndcds/uranus/app"
)

type ApiHandler struct {
	Config   *app.Config
	Context  context.Context
	DbPool   *pgxpool.Pool
	DbSchema string
	UserId   int
}

func (h *ApiHandler) InitFromGin(gc *gin.Context) {
	h.Context = gc.Request.Context()
	h.UserId = gc.GetInt("user-id")
}
