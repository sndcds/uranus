package api

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

type ApiHandler struct {
	Config   *app.Config
	DbPool   *pgxpool.Pool
	DbSchema string
}
