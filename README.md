# gredfin

I wanted to build a package to scrape property data. I was able to find an unofficial Redfin client implemented in Python, so I decided I'd implement it in Go and build this project around it. The ultimate goal is to surface a list of realtors and their property sale history to provide everyone with a clear record of their realtor's past performance. The idea is to regularly scrape Redfin (or any realty API) for property data, store the data, perform analytics on it, and expose the data to a client (Flutter) app. An HTTP server manages job queues and provides REST API endpoints for accessing the data; distributed workers pull jobs from the queue and upload the results back to the server.

## Overall Strategy

1. A worker will regularly run "search queries" to collect a CSV of property listings on a zipcode-by-zipcode basis. The worker uses the URL field in the CSV to fetch the corresponding `(propertyID, listingID)` from Redfin and insert a corresponding row the property table.

2. A different worker will claim property listings, and do a full deep dive on each property. It will grab all the data and upload it to cloud storage (with some hashing to avoid uploading duplicate data). The worker may also upload data of interest back to the server (e.g., the corresponding realtor and listing price).

## How to Use

This repo has 4 top level packages: `redfin`, `server`, `worker`, and `cmd`. The `redfin` package provides a Redfin client. The `cmd` package provides the entry point for all the `server` and `worker` packages. You can build the CLI with `make build cli`. This will output a binary named `cli`. You can run the various packages like:

```bash
./cli --help
./cli run http-server [OPTIONS]
./cli run search-worker [OPTIONS]
./cli run property-worker [OPTIONS]
```

However, there are a number of options you'll need to specify, which can be error prone. As a result, you'll want to instead likely run something like:

```bash
make build-cli && make run-http-server
```

This will automatically run the `run http-server` subcommand with options populated from the contents of `server/.env`. You can look at the resulting command to see the necessary envs to specify. Similarly, the worker commands will use the contents of `worker/.env`.

## Package Redfin

This is a client wrapper around the unofficial Redfin API. Workers will typically instantiate a client for running scraping jobs.

## Package Server

This is an HTTP server that provides an interface to the DB and cloud storage. Clients use this API to pull "jobs" (i.e., scraping targets), run their job, and then upload data to the cloud and/or server. The server also provides things like S3 Presigned URLs to workers to they can upload their data to the cloud without needing any cloud credentials, bucket details, etc. All routes require authentication in the form of a `Authorization` header specifying a `Bearer` token in the form of a JWT. Tokens can be obtained from the server by supplying the necessary passphrase.

## Package Worker

This is a collection of workers that run tasks on regular intervals. They'll do things like pull a list of properties from the server and scrape each one for details. You can implement your own worker function easily; the interface is rather simple: `func(context.Context, *slog.Logger)`. Any function implementing this interface can be supplied as a worker that runs on the specified interval.

# Questions to Ask a Prospective Realtor (from Redfin)

Here's an interesting excerpt from a Redfin post. We should strive to answer these questions. Users will want to provide a realtor's name and get answers to these. And by answering these, maybe Redfin will acquire us.

https://www.redfin.com/guides/how-to-choose-a-real-estate-agent-top-15-questions

How to Choose a Real Estate Agent - Top 15 Questions to Ask
Get advice on hiring a real estate agent.

Without knowing how to properly interview and hire a real estate agent, many homebuyers end up hiring the first real estate agent they meet. It can be difficult to reject someone, but you’ll get the best home buying experience if you know how to choose a realtor that’s right for you. Make sure you ask these 15 questions before selecting a real estate agent.

1. Is this your full-time gig? How many clients have you served this year?
   You want an active real estate agent who isn’t going to be distracted by other obligations, and who is up to date on current market conditions and laws.

2. How many sales have you handled in my target neighborhoods?
   Local expertise is key. Try finding an agent with at least a few recent deals in the neighborhoods you’re interested in.

3. When clients are unhappy with your service, what has gone wrong?
   Asking why a client has been a bad fit for an agent can help you figure out if you’re a good fit.

4. Has a client ever filed a complaint against you?
   If you’re uncomfortable asking, just check with the state real estate agent licensing board.

5. What’s your fee?
   In addition to paying their listing agent, the seller also pays the buyer’s agent out of the money you pay for the house — typically 2.5%–3% of the sale price. Some buyer’s agents — like Redfin Agents — refund part of this fee. Learn more about real estate commission.

6. What services do you offer beyond negotiations and escrow?
   Make a list of what you need: negotiations, paperwork, and contingencies are the minimum.

7. When am I committed to working with you?
   Many homebuyers start touring homes without realizing this can obligate them to work with an agent, contract or no contract. Make sure you know the limits, or go on a free, no-obligation home tour with a Redfin Agent.

8. How many foreclosure or short-sale transactions have you handled?
   Distressed properties can be great deals, but the paperwork is complicated and your liability is greater. Whether or not you’re interested in this type of transaction, the best agents have experience closing deals with banks.

9. Who else will be working with me?
   An agent is often supported by a team, but the person you hire should do most of the work. Get an understanding of what your agent is taking responsibility for.

10. Will you show me all the properties for sale?
    Good agents show their clients all of the properties they may be interested in, even for-sale-by-owner properties that don’t pay a commission.

11. How quickly can you get me into a home?
    Hot homes move fast. Ask how the agent handles tours on short notice, and what their day-to-day availability for a consultation is.

12. Do you represent buyers and sellers on the same house?
    This is called dual agency, and it’s something you want to avoid. No agent can fairly represent both sides of a deal. Make sure your agent is only advocating for you.

13. What sets you apart from other agents?
    Look for expertise, not just eagerness. You aren’t hiring the neighborhood kid to rake your leaves. You need someone who is confident and dedicated to their craft.

14. What if I’m unhappy with your service?
    Agents get paid when you buy a house, but most customer complaints occur during the closing process. Ask for some sort of satisfaction guarantee.

15. Can I get references for your last five deals?
    Every agent has some clients that were served well, but the best agents serve all of their clients well. Getting an agent’s last five customers will give you a more balanced picture of their service than letting them choose their most favorable references. Look closely at these deals to see how they compare to your needs, and if the agent negotiated a good price on each one.
