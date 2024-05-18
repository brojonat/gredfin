package main

import (
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/brojonat/gredfin/client"
	"github.com/brojonat/gredfin/server"
	"github.com/brojonat/gredfin/worker"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:  "serve-http",
				Usage: "Run the HTTP server on the specified port.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "listen-port",
						Aliases: []string{"port", "p"},
						Value:   "8080",
						Usage:   "Port to listen on.",
					},
					&cli.StringFlag{
						Name:    "db-host",
						Aliases: []string{"db", "d"},
						Value:   os.Getenv("DATABASE_URL"),
						Usage:   "Database endpoint.",
					},
				},
				Action: func(ctx *cli.Context) error {
					return serve_http(ctx)
				},
			},
			{
				Name:  "run-search-worker",
				Usage: "Run a search worker.",
				Flags: []cli.Flag{
					&cli.DurationFlag{
						Name:    "interval",
						Aliases: []string{"i"},
						Value:   time.Hour,
						Usage:   "Minimum interval between running tasks.",
					},
					&cli.DurationFlag{
						Name:    "claim-tasks-older-than",
						Aliases: []string{"older", "o"},
						Value:   24 * time.Hour,
						Usage:   "Only claim tasks older than this value.",
					},
				},
				Action: func(ctx *cli.Context) error {
					return run_search_worker(ctx)
				},
			},
			{
				Name:  "run-property-scrape-worker",
				Usage: "Run a property scrape worker.",
				Flags: []cli.Flag{
					&cli.DurationFlag{
						Name:    "interval",
						Aliases: []string{"i"},
						Value:   time.Hour,
						Usage:   "Minimum interval between running tasks.",
					},
					&cli.DurationFlag{
						Name:    "claim-tasks-older-than",
						Aliases: []string{"older", "o"},
						Value:   7 * 24 * time.Hour,
						Usage:   "Only claim tasks older than this value.",
					},
				},
				Action: func(ctx *cli.Context) error {
					return run_property_scrape_worker(ctx)
				},
			},
		}}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func serve_http(ctx *cli.Context) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	redfinClient := client.NewClient("https://redfin.com/stingray/", "gredfin-client (brojonat@gmail.com)")
	return server.RunHTTPServer(
		ctx.Context,
		logger,
		ctx.String("db-host"),
		redfinClient,
		ctx.String("listen-port"),
	)
}

func run_search_worker(ctx *cli.Context) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	worker.RunSearchWorker(
		ctx.Context,
		logger,
		ctx.Duration("interval"),
		ctx.Duration("claim-tasks-older-than"),
	)
	return nil
}

func run_property_scrape_worker(ctx *cli.Context) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	worker.RunPropertyWorker(
		ctx.Context,
		logger,
		ctx.Duration("interval"),
		ctx.Duration("claim-tasks-older-than"),
	)
	return nil
}
