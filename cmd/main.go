package main

import (
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/brojonat/gredfin/redfin"
	"github.com/brojonat/gredfin/server"
	"github.com/brojonat/gredfin/worker"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:  "run-http-server",
				Usage: "Run the HTTP server on the specified port.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "listen-port",
						Aliases: []string{"port", "p"},
						Value:   "8080",
						Usage:   "Port to listen on.",
					},
					&cli.StringFlag{
						Name:     "db-host",
						Aliases:  []string{"db", "d"},
						Value:    os.Getenv("DATABASE_URL"),
						Usage:    "Database endpoint.",
						Required: true,
					},
				},
				Action: func(ctx *cli.Context) error {
					return serve_http(ctx)
				},
			},
			{
				Name:  "run-search-worker",
				Usage: "Run a search scrape worker.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "server-endpoint",
						Aliases: []string{"server", "s"},
						Value:   os.Getenv("SERVER_ENDPOINT"),
						Usage:   "Server endpoint.",
					},
					&cli.StringFlag{
						Name:    "auth-token",
						Aliases: []string{"token", "t"},
						Value:   os.Getenv("AUTH_TOKEN"),
						Usage:   "Auth token for server requests.",
					},
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
				Name:  "run-property-worker",
				Usage: "Run a property scrape worker.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "server-endpoint",
						Aliases: []string{"server", "s"},
						Value:   os.Getenv("SERVER_ENDPOINT"),
						Usage:   "Server endpoint.",
					},
					&cli.StringFlag{
						Name:    "auth-token",
						Aliases: []string{"token", "t"},
						Value:   os.Getenv("AUTH_TOKEN"),
						Usage:   "Auth token for server requests.",
					},
					&cli.DurationFlag{
						Name:    "interval",
						Aliases: []string{"i"},
						Value:   time.Second,
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
	redfinClient := redfin.NewClient("https://redfin.com/stingray/", "gredfin-client (brojonat@gmail.com)")
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
	redfinClient := redfin.NewClient("https://redfin.com/stingray/", "gredfin-client (brojonat@gmail.com)")
	cfg, err := config.LoadDefaultConfig(ctx.Context)
	if err != nil {
		log.Fatal(err)
	}
	s3Client := s3.NewFromConfig(cfg)
	worker.RunWorkerFunc(
		ctx.Context,
		logger,
		ctx.Duration("interval"),
		worker.MakeSearchWorkerFunc(
			ctx.String("server-endpoint"),
			ctx.String("auth-token"),
			redfinClient, s3Client,
		),
	)
	return nil
}

func run_property_scrape_worker(ctx *cli.Context) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	redfinClient := redfin.NewClient("https://redfin.com/stingray/", "gredfin-client (brojonat@gmail.com)")
	cfg, err := config.LoadDefaultConfig(ctx.Context)
	if err != nil {
		log.Fatal(err)
	}
	s3Client := s3.NewFromConfig(cfg)
	worker.RunWorkerFunc(
		ctx.Context,
		logger,
		ctx.Duration("interval"),
		worker.MakePropertyWorkerFunc(
			ctx.String("server-endpoint"),
			ctx.String("auth-token"),
			redfinClient,
			s3Client,
		),
	)
	return nil
}
