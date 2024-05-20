# gredfin

I wanted to build a package to scrape property data. I was able to find an unofficial Redfin client implemented in Python, so I decided I'd implement it in Go and build this project around it. The ultimate goal is to surface a list of realtors and their property sale history to provide everyone with a clear record of their realtor's past performance. The idea is to regularly scrape Redfin (or any realty API) for property data, store the data, and then perform analytics on it. The implementation uses an HTTP server manage job queues and surface data; distributed workers pull jobs from the queue and query the property API(s).

## How to Use

This repo has 4 top level packages: `server`, `worker`, `redfin`, and `cmd`. The `cmd` package provides the entry point for all the others. You can build the CLI with `make build cli`. This will output a binary to `cmd/cli`. You can run the various packages like:

```bash
cmd/cli run-http-server [OPTIONS]
cmd/cli run-search-worker [OPTIONS]
cmd/cli run-property-worker [OPTIONS]
```

However, there are a number of options you'll need to specify, which can be error prone. As a result, you'll want to instead likely run something like:

```bash
make build-cli && make run-http-server
```

This will automatically run the `run-http-server` subcommand with options populated from the contents of `server/.env`. You can look at the resulting command to see the necessary envs to specify. Similarly, the worker commands will use the contents of `worker/.env`.

## Package Server

This is an HTTP server that provides an interface to the DB and cloud storage. Clients use this API to pull "jobs" (i.e., scraping targets), run their job, and then POST some data back to the server. The server also provides things like Presigned URLs to workers to they can upload their data to the cloud without needing any cloud credentials, bucket details, etc.

## Package Redfin

This is a client wrapper around the unofficial Redfin API. Workers will typically instantiate a client for running scraping jobs.

## Package Worker

This is a collection of workers that run tasks on regular intervals. They'll do things like pull a list of properties from the server and scrape each one for details. You can implement your own worker function easily; the interface is rather simple: `func(context.Context, *slog.Logger)`. Any function implementing this interface can be supplied as a worker that runs on the specified interval.
