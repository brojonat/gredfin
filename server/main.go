package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/brojonat/gredfin/redfin"
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

func RunHTTPServer(
	ctx context.Context,
	l *slog.Logger,
	dbHost string,
	c redfin.Client,
	port string,
) error {
	db, err := getConnPool(ctx, dbHost)
	if err != nil {
		return fmt.Errorf("could not connect to db: %s", err)
	}
	q := dbgen.New(db)
	fmt.Printf("listening on %s...\n", port)
	return http.ListenAndServe(
		fmt.Sprintf(":%s", port),
		getRootHandler(l, db, q),
	)
}
