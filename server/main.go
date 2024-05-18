package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	grc "github.com/brojonat/gredfin/client"
	"github.com/brojonat/gredfin/server/dbgen"
	"github.com/jackc/pgx/v5/pgxpool"
)

func handleTest(l *slog.Logger, c grc.Client, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := c.Search("18 Brandywine St, Burlington, VT 05408", map[string]string{})
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		fmt.Printf("%s", b)

		props, err := q.ListProperties(r.Context())
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(props)
	}
}

func writeInternalError(l *slog.Logger, w http.ResponseWriter, e error) {
	l.Error(e.Error())
	json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{Error: "internal error"})
}

func handlePing(l *slog.Logger, p *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := p.Ping(r.Context())
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
	}
}

func getConnPool(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}
	if err = pool.Ping(ctx); err != nil {
		return nil, err
	}
	return pool, nil
}

func RunHTTPServer(ctx context.Context, l *slog.Logger, dbHost string, c grc.Client, port string) error {
	db, err := getConnPool(ctx, dbHost)
	if err != nil {
		return fmt.Errorf("could not connect to db: %s", err)
	}
	q := dbgen.New(db)
	setupRoutes(l, c, db, q)
	fmt.Printf("listening on %s...", port)
	return http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}
