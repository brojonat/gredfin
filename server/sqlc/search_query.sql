-- name: GetSearch :one
SELECT * FROM search
WHERE search_id = $1 LIMIT 1;

-- name: ListSearches :many
SELECT * FROM search
ORDER BY search_id;

-- name: CreateSearch :one
INSERT INTO search (
  search_id, query, last_scrape_ts, last_scrape_status
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

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