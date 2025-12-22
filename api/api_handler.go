package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

type ApiHandler struct {
	Config   *app.Config
	DbPool   *pgxpool.Pool
	DbSchema string
}

type ApiTxError struct {
	Code int
	Err  error
}

func (e *ApiTxError) Error() string {
	return e.Err.Error()
}

func WithTransaction(ctx context.Context, db *pgxpool.Pool, fn func(tx pgx.Tx) *ApiTxError) *ApiTxError {
	tx, err := db.Begin(ctx)
	if err != nil {
		return &ApiTxError{Code: http.StatusInternalServerError, Err: fmt.Errorf("failed to start transaction: %w", err)}
	}

	defer func() {
		// rollback if not already committed/rolled back
		_ = tx.Rollback(ctx)
	}()

	if herr := fn(tx); herr != nil {
		return herr // transaction will rollback
	}

	if err := tx.Commit(ctx); err != nil {
		return &ApiTxError{Code: http.StatusInternalServerError, Err: fmt.Errorf("failed to commit transaction: %w", err)}
	}

	return nil
}
