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
WHERE last_scrape_status = ANY($2::VARCHAR[])
ORDER BY NOW()::timestamp - last_scrape_ts
LIMIT $1
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
  last_scrape_status = $4
WHERE search_id = $1;

-- name: UpdateSearchStatus :exec
UPDATE search
  SET last_scrape_ts = NOW()::timestamp,
  last_scrape_status = $2
WHERE search_id = $1;

-- name: DeleteSearch :exec
DELETE FROM search
WHERE search_id = $1;

-- name: DeleteSearchByQuery :exec
DELETE FROM search
WHERE query = $1;