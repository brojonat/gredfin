package server

import (
	"log/slog"
	"net/http"

	"github.com/brojonat/gredfin/server/dbgen"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getRootHandler(l *slog.Logger, p *pgxpool.Pool, q *dbgen.Queries) http.Handler {
	r := mux.NewRouter()
	allowedOrigins := []string{}
	maxBytes := int64(1048576)

	// helper routes
	r.Handle("/ping", adaptHandler(
		handlePing(l, p),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	)).Methods(http.MethodGet)
	r.Handle("/token", adaptHandler(
		handleIssueToken(l),
		apiMode(l, maxBytes, allowedOrigins),
		// no token required here
	)).Methods(http.MethodPost)

	// realtor CRUDL routes
	r.Handle("/realtor", adaptHandler(
		handleRealtorGet(l, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	)).Methods(http.MethodGet)
	r.Handle("/realtor", adaptHandler(
		handleRealtorPost(l, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	)).Methods(http.MethodPost)
	r.Handle("/realtor", adaptHandler(
		handleRealtorDelete(l, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	)).Methods(http.MethodDelete)

	// search CRUDL routes
	r.Handle("/search-query", adaptHandler(
		handleSearchQueryGet(l, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	)).Methods(http.MethodGet)
	r.Handle("/search-query", adaptHandler(
		handleSearchQueryPost(l, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	)).Methods(http.MethodPost)
	r.Handle("/search-query", adaptHandler(
		handleSearchQueryDelete(l, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	)).Methods(http.MethodDelete)

	// property CRUDL routes
	r.Handle("/property-query", adaptHandler(
		handlePropertyQueryGet(l, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	)).Methods(http.MethodGet)
	r.Handle("/property-query", adaptHandler(
		handlePropertyQueryPost(l, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	)).Methods(http.MethodPost)
	r.Handle("/property-query", adaptHandler(
		handlePropertyQueryDelete(l, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	)).Methods(http.MethodDelete)

	// worker routes
	r.Handle("/search-query/claim-next", adaptHandler(
		handleSearchQueryClaimNext(l, p, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	)).Methods(http.MethodPost)
	r.Handle("/search-query/set-status", adaptHandler(
		handleSearchQuerySetStatus(l, q),
		mustAuth(),
	)).Methods(http.MethodPost)

	r.Handle("/property-query/claim-next", adaptHandler(
		handlePropertyQueryClaimNext(l, p, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	)).Methods(http.MethodPost)
	r.Handle("/property-query/set-status", adaptHandler(
		handlePropertySetStatus(l, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	)).Methods(http.MethodPost)
	return r
}
