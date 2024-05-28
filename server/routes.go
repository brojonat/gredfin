package server

import (
	"log/slog"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/brojonat/gredfin/server/dbgen"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getRootHandler(
	l *slog.Logger,
	p *pgxpool.Pool,
	q *dbgen.Queries,
	s3 *s3.Client,
) http.Handler {
	mux := http.NewServeMux()
	allowedOrigins := []string{}
	maxBytes := int64(1048576)

	// helper routes
	mux.HandleFunc("GET /ping", adaptHandler(
		handlePing(l, p),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	))
	mux.HandleFunc("POST /token", adaptHandler(
		handleIssueToken(l),
		apiMode(l, maxBytes, allowedOrigins),
		// no token required here
	))

	// realtor CRUDL routes
	mux.HandleFunc("GET /realtor", adaptHandler(
		handleRealtorGet(l, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	))
	mux.HandleFunc("POST /realtor", adaptHandler(
		handleRealtorPost(l, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	))
	mux.HandleFunc("DELETE /realtor", adaptHandler(
		handleRealtorDelete(l, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	))

	// search CRUDL routes
	mux.HandleFunc("GET /search-query", adaptHandler(
		handleSearchQueryGet(l, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	))
	mux.HandleFunc("POST /search-query", adaptHandler(
		handleSearchQueryPost(l, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	))
	mux.HandleFunc("DELETE /search-query", adaptHandler(
		handleSearchQueryDelete(l, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	))

	// property CRUDL routes
	mux.HandleFunc("GET /property-query", adaptHandler(
		handlePropertyQueryGet(l, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	))
	mux.HandleFunc("POST /property-query", adaptHandler(
		handlePropertyQueryPost(l, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	))
	mux.HandleFunc("DELETE /property-query", adaptHandler(
		handlePropertyQueryDelete(l, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	))

	// search worker routes
	mux.HandleFunc("POST /search-query/claim-next", adaptHandler(
		handleSearchQueryClaimNext(l, p, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	))
	mux.HandleFunc("POST /search-query/set-status", adaptHandler(
		handleSearchQuerySetStatus(l, q),
		mustAuth(),
	))
	// property worker routes
	mux.HandleFunc("POST /property-query/claim-next", adaptHandler(
		handlePropertyQueryClaimNext(l, p, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	))
	mux.HandleFunc("POST /property-query/set-status", adaptHandler(
		handlePropertySetStatus(l, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	))
	mux.HandleFunc("POST /property-query/get-presigned-put-url", adaptHandler(
		handleGetPresignedPutURL(l, s3, q),
		apiMode(l, maxBytes, allowedOrigins),
		mustAuth(),
	))
	return mux
}
