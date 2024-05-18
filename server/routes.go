package server

import (
	"log/slog"
	"net/http"

	grc "github.com/brojonat/gredfin/client"
	"github.com/brojonat/gredfin/server/dbgen"
	"github.com/brojonat/server-tools/stools"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupRoutes(l *slog.Logger, c grc.Client, p *pgxpool.Pool, q *dbgen.Queries) {
	http.HandleFunc("/test", stools.AdaptHandler(handleTest(l, c, q)))
	http.HandleFunc("/ping", stools.AdaptHandler(handlePing(l, p)))
}
