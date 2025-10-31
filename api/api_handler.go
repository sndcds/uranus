package api

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sndcds/uranus/app"
)

type ApiHandler struct {
	DbPool *pgxpool.Pool
	Config *app.Config
}
