-- name: GetSearch :one
SELECT * FROM search
WHERE search_id = $1 LIMIT 1;

-- name: GetSearchByQuery :one
SELECT * FROM search
WHERE query = $1;

-- name: GetNNextSearchScrapeForUpdate :one
-- Get the next N property entries that have a last_scrape_status in the
-- supplied slice. Rows are locked for update; callers are expected to set
-- status rows to PENDING after retrieving rows.
SELECT * FROM search
WHERE last_scrape_status = ANY(sqlc.arg(statuses)::VARCHAR[])
ORDER BY NOW()::timestamp - last_scrape_ts DESC
LIMIT sqlc.arg(count)
FOR UPDATE;

-- name: ListSearches :many
SELECT * FROM search
ORDER BY search_id;

-- name: CreateSearch :exec
INSERT INTO search (
  query
) VALUES (
  $1
);

-- name: PostSearch :exec
UPDATE search
  SET query = $2,
  last_scrape_ts = $3,
  last_scrape_status = $4,
  last_scrape_metadata = $5
WHERE search_id = $1;

-- name: UpdateSearchStatus :exec
UPDATE search
  SET last_scrape_ts = NOW()::timestamp,
  last_scrape_status = @last_scrape_status,
  last_scrape_metadata = COALESCE(sqlc.narg('last_scrape_metadata'), last_scrape_metadata)
WHERE search_id = @search_id;

-- name: DeleteSearch :exec
DELETE FROM search
WHERE search_id = $1;

-- name: DeleteSearchByQuery :exec
DELETE FROM search
WHERE query = $1;

-- name: GetRecentSearchScrapeStats :one
SELECT
       COUNT(*) FILTER (WHERE last_scrape_status = 'good') AS good,
       COUNT(*) FILTER (WHERE last_scrape_status = 'pending') AS pending,
       COUNT(*) FILTER (WHERE last_scrape_status = 'bad') AS bad,
       COUNT(*) FILTER (WHERE last_scrape_status IS NULL) AS "null"
FROM search
WHERE last_scrape_ts > $1;
