package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	grc "github.com/brojonat/gredfin/client"
	"github.com/brojonat/gredfin/server/dbgen"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

func main() {
	l := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	c := grc.NewClient("https://redfin.com/stingray/", "gredfin-client (brojonat@gmail.com)")
	db, err := getConnPool(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not connect to db: %s\n", err)
		os.Exit(1)
	}
	q := dbgen.New(db)
	setupRoutes(l, c, db, q)

	fmt.Println("listening on :8080...")
	http.ListenAndServe(":8080", nil)
}
