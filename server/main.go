package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/brojonat/gredfin/redfin"
	"github.com/brojonat/gredfin/server/dbgen"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twpayne/go-geos"
	pgxgeos "github.com/twpayne/pgx-geos"
)

func getConnPool(ctx context.Context, url string) (*pgxpool.Pool, error) {
	dummy, err := pgxpool.New(ctx, url)
	cfg := dummy.Config()
	cfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		if err := pgxgeos.Register(ctx, c, geos.NewContext()); err != nil {
			return err
		}
		return nil
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
) error {
	db, err := getConnPool(ctx, dbHost)
	if err != nil {
		return fmt.Errorf("could not connect to db: %s", err)
	}
	q := dbgen.New(db)
	l.Info(fmt.Sprintf("listening on %s...", port))
	return http.ListenAndServe(
		fmt.Sprintf(":%s", port),
		getRootHandler(l, db, q, s3),
	)
}
