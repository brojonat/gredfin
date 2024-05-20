# gredfin

## How to Use

This repo has 4 top level packages: `server`, `worker`, `redfin`, and `cmd`. The `cmd` package provides the entry point for all the others. You can build the CLI with `make build cli`. This will output a binary to `cmd/cli`. You can run the various packages like:

```bash
cmd/cli run-http-server [OPTIONS]
cmd/cli run-search-worker [OPTIONS]
cmd/cli run-property-worker [OPTIONS]
```

## Package Server

This is an HTTP server that provides an interface to the DB and cloud storage. Clients use this API to pull "jobs" (i.e., scraping targets), run their job, and then POST some data back to the server. The server also provides things like Presigned URLs to workers to they can upload their data to the cloud without needing any cloud credentials or bucket details and whatnot.

## Package Redfin

This is a client wrapper around the unofficial Redfin API. Workers will typically instantiate a client for running scraping jobs.

## Package Worker

This is a collection of workers that run tasks on regular intervals. They'll do things like pull a list of properties from the server and scrape each one for details. You can implement you own worker function easily; the interface is rather simple.
