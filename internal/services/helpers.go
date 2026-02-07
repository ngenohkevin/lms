package services

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
)

// isNoRows checks if an error is a "no rows" error from either pgx or database/sql.
// pgx v5 can return either pgx.ErrNoRows or sql.ErrNoRows depending on context.
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows)
}
