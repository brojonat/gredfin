package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"firebase.google.com/go/auth"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/brojonat/gredfin/redfin"
	"github.com/brojonat/gredfin/server/db/dbgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxgeom "github.com/twpayne/pgx-geom"
)

func getConnPool(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxgeom.Register(ctx, conn)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
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
	port string,
	l *slog.Logger,
	dbHost string,
	c redfin.Client,
	s3 *s3.Client,
	fbc *auth.Client,
) error {
	db, err := getConnPool(ctx, dbHost)
	if err != nil {
		return fmt.Errorf("could not connect to db: %s", err)
	}
	q := dbgen.New(db)

	l.Info(fmt.Sprintf("listening on %s...", port))
	return http.ListenAndServe(
		fmt.Sprintf(":%s", port),
		getRootHandler(l, db, q, s3, fbc),
	)
}
