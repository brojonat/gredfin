package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
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
				Name:  "admin",
				Usage: "Commands for performing administrative tasks.",
				Subcommands: []*cli.Command{
					{
						Name:  "add-search-query",
						Usage: "Add a search query that search workers will run.",
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
							&cli.StringFlag{
								Name:     "query",
								Aliases:  []string{"q"},
								Usage:    "Redfin search query (this should just be a zipcode).",
								Required: true,
							},
							&cli.IntFlag{
								Name:    "log-level",
								Aliases: []string{"ll", "l"},
								Usage:   "Logging level for the slog.Logger. Default is 0 (INFO), use -4 for DEBUG.",
								Value:   0,
							},
						},
						Action: func(ctx *cli.Context) error {
							return add_search_query(ctx)
						},
					},
					{
						Name:  "test-search-query",
						Usage: "Run a particular search query.",
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
							&cli.StringFlag{
								Name:     "query",
								Aliases:  []string{"q"},
								Usage:    "Redfin search query (this should just be a zipcode).",
								Required: true,
							},
							&cli.StringFlag{
								Name:    "user-agent",
								Aliases: []string{"ua", "u"},
								Value:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
								Usage:   "Redfin client User-Agent",
							},
							&cli.IntFlag{
								Name:    "log-level",
								Aliases: []string{"ll", "l"},
								Usage:   "Logging level for the slog.Logger. Default is 0 (INFO), use -4 for DEBUG.",
								Value:   0,
							},
						},
						Action: func(ctx *cli.Context) error {
							return test_search_query(ctx)
						},
					},
					{
						Name:  "add-property-query",
						Usage: "Add a property query that property workers will run.",
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
							&cli.StringFlag{
								Name:     "property_id",
								Aliases:  []string{"pid"},
								Usage:    "Redfin property ID.",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "listing_id",
								Aliases:  []string{"lid"},
								Usage:    "Redfin listing ID.",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "url",
								Aliases:  []string{"u"},
								Usage:    "Redfin URL.",
								Required: true,
							},
							&cli.IntFlag{
								Name:    "log-level",
								Aliases: []string{"ll", "l"},
								Usage:   "Logging level for the slog.Logger. Default is 0 (INFO), use -4 for DEBUG.",
								Value:   0,
							},
						},
						Action: func(ctx *cli.Context) error {
							return add_property_query(ctx)
						},
					},
				},
			},
			{
				Name:  "run",
				Usage: "Commands for running various components (server, workers, etc.)",
				Subcommands: []*cli.Command{
					{
						Name:  "http-server",
						Usage: "Run the HTTP server on the specified port.",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "listen-port",
								Aliases: []string{"port", "p"},
								Value:   os.Getenv("SERVER_PORT"),
								Usage:   "Port to listen on.",
							},
							&cli.StringFlag{
								Name:    "db-host",
								Aliases: []string{"db", "d"},
								Value:   os.Getenv("DATABASE_URL"),
								Usage:   "Database endpoint.",
							},
							&cli.StringFlag{
								Name:    "user-agent",
								Aliases: []string{"ua", "u"},
								Value:   os.Getenv("REDFIN_USER_AGENT"),
								Usage:   "Redfin client User-Agent",
							},
							&cli.IntFlag{
								Name:    "log-level",
								Aliases: []string{"ll", "l"},
								Usage:   "Logging level for the slog.Logger. Default is 0 (INFO), use -4 for DEBUG.",
								Value:   0,
							},
						},
						Action: func(ctx *cli.Context) error {
							return serve_http(ctx)
						},
					},
					{
						Name:  "search-worker",
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
							&cli.DurationFlag{
								Name:    "property-query-delay",
								Aliases: []string{"pqd", "d"},
								Value:   500 * time.Millisecond,
								Usage:   "Delay between search result property queries.",
							},
							&cli.StringFlag{
								Name:    "user-agent",
								Aliases: []string{"ua", "u"},
								Value:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
								Usage:   "Redfin client User-Agent",
							},
							&cli.IntFlag{
								Name:    "log-level",
								Aliases: []string{"ll", "l"},
								Usage:   "Logging level for the slog.Logger. Default is 0 (INFO), use -4 for DEBUG.",
								Value:   0,
							},
						},
						Action: func(ctx *cli.Context) error {
							return run_search_worker(ctx)
						},
					},
					{
						Name:  "property-worker",
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
							&cli.StringFlag{
								Name:    "user-agent",
								Aliases: []string{"ua", "u"},
								Value:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
								Usage:   "Redfin client User-Agent",
							},
							&cli.IntFlag{
								Name:    "log-level",
								Aliases: []string{"ll", "l"},
								Usage:   "Logging level for the slog.Logger. Default is 0 (INFO), use -4 for DEBUG.",
								Value:   0,
							},
						},
						Action: func(ctx *cli.Context) error {
							return run_property_scrape_worker(ctx)
						},
					},
				},
			},
		}}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error running command: %s\n", err.Error())
		os.Exit(1)
	}
}

func serve_http(ctx *cli.Context) error {
	logger := getDefaultLogger(slog.Level(ctx.Int("log-level")))
	redfinClient := redfin.NewClient("https://www.redfin.com/stingray/", ctx.String("user-agent"))
	cfg, err := config.LoadDefaultConfig(ctx.Context)
	if err != nil {
		log.Fatal(err)
	}
	s3Client := s3.NewFromConfig(cfg)
	return server.RunHTTPServer(
		ctx.Context,
		ctx.String("listen-port"),
		logger,
		ctx.String("db-host"),
		redfinClient,
		s3Client,
	)
}

func run_search_worker(ctx *cli.Context) error {
	logger := getDefaultLogger(slog.Level(ctx.Int("log-level")))
	redfinClient := redfin.NewClient("https://www.redfin.com/stingray/", ctx.String("user-agent"))
	pqd, err := time.ParseDuration(ctx.String("property-query-delay"))
	if err != nil {
		log.Fatal(err)
	}
	worker.RunWorkerFunc(
		ctx.Context,
		logger,
		ctx.Duration("interval"),
		worker.MakeSearchWorkerFunc(
			ctx.String("server-endpoint"),
			ctx.String("auth-token"),
			redfinClient,
			pqd,
		),
	)
	return nil
}

func run_property_scrape_worker(ctx *cli.Context) error {
	logger := getDefaultLogger(slog.Level(ctx.Int("log-level")))
	redfinClient := redfin.NewClient("https://www.redfin.com/stingray/", ctx.String("user-agent"))
	worker.RunWorkerFunc(
		ctx.Context,
		logger,
		ctx.Duration("interval"),
		worker.MakePropertyWorkerFunc(
			ctx.String("server-endpoint"),
			ctx.String("auth-token"),
			redfinClient,
		),
	)
	return nil
}

func add_search_query(ctx *cli.Context) error {
	logger := getDefaultLogger(slog.Level(ctx.Int("log-level")))
	return AddSeachQuery(
		ctx.Context,
		logger,
		ctx.String("server-endpoint"),
		ctx.String("auth-token"),
		ctx.String("query"),
	)
}

func test_search_query(ctx *cli.Context) error {
	logger := getDefaultLogger(slog.Level(ctx.Int("log-level")))
	redfinClient := redfin.NewClient("https://www.redfin.com/stingray/", ctx.String("user-agent"))
	sp := worker.GetDefaultSearchParams()
	gissp := worker.GetDefaultGISCSVParams()
	urls, err := worker.GetURLSFromQuery(
		logger,
		redfinClient,
		ctx.String("query"),
		sp,
		gissp,
	)
	if err != nil {
		return err
	}
	for _, u := range urls {
		fmt.Printf("%s\n", u)
	}
	return nil
}

func add_property_query(ctx *cli.Context) error {
	logger := getDefaultLogger(slog.Level(ctx.Int("log-level")))
	return AddProperty(
		ctx.Context,
		logger,
		ctx.String("server-endpoint"),
		ctx.String("auth-token"),
		ctx.String("property_id"),
		ctx.String("listing_id"),
		ctx.String("url"),
	)
}

func getDefaultLogger(lvl slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     lvl,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				source, _ := a.Value.Any().(*slog.Source)
				if source != nil {
					source.Function = ""
					source.File = filepath.Base(source.File)
				}
			}
			return a
		},
	}))
}
