package server

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

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

	// max body size
	maxBytes := int64(1048576)

	// parse and transform the comma separated envs that configure CORS
	hs := os.Getenv("CORS_HEADERS")
	ms := os.Getenv("CORS_METHODS")
	ogs := os.Getenv("CORS_ORIGINS")
	normalizeCORSParams := func(e string) []string {
		params := strings.Split(e, ",")
		for i, p := range params {
			params[i] = strings.ReplaceAll(p, " ", "")
		}
		return params
	}
	headers := normalizeCORSParams(hs)
	methods := normalizeCORSParams(ms)
	origins := normalizeCORSParams(ogs)

	// helper routes
	mux.HandleFunc("OPTIONS /", adaptHandler(
		func(w http.ResponseWriter, r *http.Request) {},
		apiMode(l, maxBytes, headers, methods, origins),
	))
	mux.HandleFunc("GET /ping", adaptHandler(
		handlePing(l, p),
		apiMode(l, maxBytes, headers, methods, origins),
		mustAuth(),
	))
	mux.HandleFunc("POST /token", adaptHandler(
		handleIssueToken(l),
		apiMode(l, maxBytes, headers, methods, origins),
		// no token required here
	))

	// realtor CRUDL routes
	mux.HandleFunc("GET /realtor", adaptHandler(
		handleRealtorGet(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		mustAuth(),
	))
	mux.HandleFunc("POST /realtor", adaptHandler(
		handleRealtorPost(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		mustAuth(),
	))
	mux.HandleFunc("DELETE /realtor", adaptHandler(
		handleRealtorDelete(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		mustAuth(),
	))

	// search CRUDL routes
	mux.HandleFunc("GET /search", adaptHandler(
		handleSearchGet(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		mustAuth(),
	))
	mux.HandleFunc("POST /search", adaptHandler(
		handleSearchPost(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		mustAuth(),
	))
	mux.HandleFunc("DELETE /search", adaptHandler(
		handleSearchDelete(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		mustAuth(),
	))

	// property CRUDL routes
	mux.HandleFunc("GET /property", adaptHandler(
		handlePropertyGet(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		mustAuth(),
	))
	mux.HandleFunc("POST /property", adaptHandler(
		handlePropertyPost(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		mustAuth(),
	))
	mux.HandleFunc("PUT /property", adaptHandler(
		handlePropertyUpdate(l, p, q),
		apiMode(l, maxBytes, headers, methods, origins),
		mustAuth(),
	))
	mux.HandleFunc("DELETE /property", adaptHandler(
		handlePropertyDelete(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		mustAuth(),
	))

	// property-event CRUDL routes
	mux.HandleFunc("GET /property-events", adaptHandler(
		handlePropertyEventsGet(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		mustAuth(),
	))
	mux.HandleFunc("POST /property-events", adaptHandler(
		handlePropertyEventsPost(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		mustAuth(),
	))
	mux.HandleFunc("PUT /property-events", adaptHandler(
		handlePropertyEventsPut(l, p, q),
		apiMode(l, maxBytes, headers, methods, origins),
		mustAuth(),
	))
	mux.HandleFunc("DELETE /property-events", adaptHandler(
		handlePropertyEventsDelete(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		mustAuth(),
	))

	// search worker routes
	mux.HandleFunc("POST /search-query/claim-next", adaptHandler(
		handleSearchClaimNext(l, p, q),
		apiMode(l, maxBytes, headers, methods, origins),
		mustAuth(),
	))
	mux.HandleFunc("POST /search-query/set-status", adaptHandler(
		handleSearchSetStatus(l, q),
		mustAuth(),
	))
	// property worker routes
	mux.HandleFunc("POST /property-query/claim-next", adaptHandler(
		handlePropertyClaimNext(l, p, q),
		apiMode(l, maxBytes, headers, methods, origins),
		mustAuth(),
	))
	mux.HandleFunc("POST /property-query/get-presigned-put-url", adaptHandler(
		handleGetPresignedPutURL(l, s3, q),
		apiMode(l, maxBytes, headers, methods, origins),
		mustAuth(),
	))

	// plot data routes
	mux.HandleFunc("GET /realtor-plot", adaptHandler(
		handlePlotDataRealtorPrices(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		mustAuth(),
	))
	mux.HandleFunc("GET /property-event-plot", adaptHandler(
		handlePlotDataPropertyPrices(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		mustAuth(),
	))
	return mux
}
