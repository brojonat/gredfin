package server

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func isPGError(err error, code string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	if pgErr.Code != code {
		return false
	}
	return true
}

// see https://www.postgresql.org/docs/11/errcodes-appendix.html#ERRCODES-TABLE
const (
	pgErrorIntegrityConstrainViolation = "23000"
	pgErrorRestrictViolation           = "23001"
	pgErrorNotNullViolation            = "23502"
	pgErrorForeignKeyViolation         = "23503"
	pgErrorUniqueViolation             = "23505"
	pgErrorCheckViolation              = "23514"
	pgErrorExclusionViolation          = "23P01"
)

func pgErrorText(code string) string {
	switch code {
	case pgErrorIntegrityConstrainViolation:
		return "integrity_constraint_violation"
	case pgErrorRestrictViolation:
		return "restrict_violation"
	case pgErrorNotNullViolation:
		return "not_null_violation"
	case pgErrorForeignKeyViolation:
		return "foreign_key_violation"
	case pgErrorUniqueViolation:
		return "unique_violation"
	case pgErrorCheckViolation:
		return "check_violation"
	case pgErrorExclusionViolation:
		return "exclusion_violation"
	default:
		return ""
	}
}
