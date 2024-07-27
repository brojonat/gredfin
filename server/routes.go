package server

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"firebase.google.com/go/auth"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/brojonat/gredfin/server/db/dbgen"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getRootHandler(
	l *slog.Logger,
	p *pgxpool.Pool,
	q *dbgen.Queries,
	s3 *s3.Client,
	fbc *auth.Client,
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
		atLeastOneAuth(bearerAuthorizer(), firebaseAuthorizer(FirebaseJWTHeader, fbc)),
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
		atLeastOneAuth(bearerAuthorizer(), firebaseAuthorizer(FirebaseJWTHeader, fbc)),
	))
	mux.HandleFunc("POST /realtor", adaptHandler(
		handleRealtorPost(l, p, q),
		apiMode(l, maxBytes, headers, methods, origins),
		atLeastOneAuth(bearerAuthorizer()),
	))
	mux.HandleFunc("DELETE /realtor", adaptHandler(
		handleRealtorDelete(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		atLeastOneAuth(bearerAuthorizer()),
	))

	// search CRUDL routes
	mux.HandleFunc("GET /search", adaptHandler(
		handleSearchGet(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		atLeastOneAuth(bearerAuthorizer()),
	))
	mux.HandleFunc("POST /search", adaptHandler(
		handleSearchPost(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		atLeastOneAuth(bearerAuthorizer()),
	))
	mux.HandleFunc("DELETE /search", adaptHandler(
		handleSearchDelete(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		atLeastOneAuth(bearerAuthorizer()),
	))

	// property CRUDL routes
	mux.HandleFunc("GET /property", adaptHandler(
		handlePropertyGet(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		atLeastOneAuth(bearerAuthorizer(), firebaseAuthorizer(FirebaseJWTHeader, fbc)),
	))
	mux.HandleFunc("POST /property", adaptHandler(
		handlePropertyPost(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		atLeastOneAuth(bearerAuthorizer()),
	))
	mux.HandleFunc("PUT /property", adaptHandler(
		handlePropertyUpdate(l, p, q),
		apiMode(l, maxBytes, headers, methods, origins),
		atLeastOneAuth(bearerAuthorizer()),
	))
	mux.HandleFunc("DELETE /property", adaptHandler(
		handlePropertyDelete(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		atLeastOneAuth(bearerAuthorizer()),
	))

	// property-event CRUDL routes
	mux.HandleFunc("GET /property-events", adaptHandler(
		handlePropertyEventsGet(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		atLeastOneAuth(bearerAuthorizer(), firebaseAuthorizer(FirebaseJWTHeader, fbc)),
	))
	mux.HandleFunc("POST /property-events", adaptHandler(
		handlePropertyEventsPost(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		atLeastOneAuth(bearerAuthorizer()),
	))
	mux.HandleFunc("PUT /property-events", adaptHandler(
		handlePropertyEventsPut(l, p, q),
		apiMode(l, maxBytes, headers, methods, origins),
		atLeastOneAuth(bearerAuthorizer()),
	))
	mux.HandleFunc("DELETE /property-events", adaptHandler(
		handlePropertyEventsDelete(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		atLeastOneAuth(bearerAuthorizer()),
	))

	// search worker routes
	mux.HandleFunc("POST /search-query/claim-next", adaptHandler(
		handleSearchClaimNext(l, p, q),
		apiMode(l, maxBytes, headers, methods, origins),
		atLeastOneAuth(bearerAuthorizer()),
	))
	mux.HandleFunc("POST /search-query/set-status", adaptHandler(
		handleSearchSetStatus(l, q),
		atLeastOneAuth(bearerAuthorizer()),
	))
	// property worker routes
	mux.HandleFunc("POST /property-query/claim-next", adaptHandler(
		handlePropertyClaimNext(l, p, q),
		apiMode(l, maxBytes, headers, methods, origins),
		atLeastOneAuth(bearerAuthorizer()),
	))
	mux.HandleFunc("POST /property-query/get-presigned-put-url", adaptHandler(
		handleGetPresignedPutURL(l, s3, q),
		apiMode(l, maxBytes, headers, methods, origins),
		atLeastOneAuth(bearerAuthorizer()),
	))

	// plot data routes
	mux.HandleFunc("GET /realtor-prices-plot", adaptHandler(
		handlePlotDataRealtorPrices(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		atLeastOneAuth(bearerAuthorizer(), firebaseAuthorizer(FirebaseJWTHeader, fbc)),
	))
	mux.HandleFunc("GET /property-prices-plot", adaptHandler(
		handlePlotDataPropertyPrices(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		atLeastOneAuth(bearerAuthorizer(), firebaseAuthorizer(FirebaseJWTHeader, fbc)),
	))

	// scrape stats routes
	mux.HandleFunc("GET /admin/search-scrape-stats", adaptHandler(
		handleGetRecentSearchScrapeStats(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		atLeastOneAuth(bearerAuthorizer(), firebaseAuthorizer(FirebaseJWTHeader, fbc)),
	))
	mux.HandleFunc("GET /admin/property-scrape-stats", adaptHandler(
		handleGetRecentPropertyScrapeStats(l, q),
		apiMode(l, maxBytes, headers, methods, origins),
		atLeastOneAuth(bearerAuthorizer(), firebaseAuthorizer(FirebaseJWTHeader, fbc)),
	))
	return mux
}
