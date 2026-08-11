package db

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	uniqueViolationCode     = "23505"
	foreignKeyViolationCode = "23503"
)

func violation(err error, code string) (*pgconn.PgError, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != code {
		return nil, false
	}
	return pgErr, true
}

// IsUniqueViolation checks if the error is a PostgreSQL unique_violation (23505)
// for the given constraint name.
func IsUniqueViolation(err error, constraint string) bool {
	pgErr, ok := violation(err, uniqueViolationCode)
	return ok && pgErr.ConstraintName == constraint
}

// IsForeignKeyViolation checks if the error is a PostgreSQL foreign_key_violation (23503)
// for the given constraint name.
func IsForeignKeyViolation(err error, constraint string) bool {
	pgErr, ok := violation(err, foreignKeyViolationCode)
	return ok && pgErr.ConstraintName == constraint
}

// IsAnyForeignKeyViolation checks if the error is a PostgreSQL foreign_key_violation
// (23503) for any constraint. Deleting a row can violate any foreign key pointing at
// it, so callers that only need to know "this record is still referenced" should use
// this instead of enumerating every constraint name.
func IsAnyForeignKeyViolation(err error) bool {
	_, ok := violation(err, foreignKeyViolationCode)
	return ok
}
