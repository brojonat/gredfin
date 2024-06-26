-- name: GetProperty :one
SELECT * FROM property
WHERE property_id = $1 AND listing_id = $2
LIMIT 1;

-- name: GetPropertiesByID :many
SELECT * FROM property
WHERE property_id = $1;

-- name: GetNNextPropertyScrapeForUpdate :one
-- Get the next N property entries that have a last_scrape_status in the
-- supplied slice. Rows are locked for update; callers are expected to set
-- status rows to PENDING after retrieving rows.
SELECT * FROM property
WHERE last_scrape_status = ANY($2::VARCHAR[])
ORDER BY NOW()::timestamp - last_scrape_ts DESC
LIMIT $1
FOR UPDATE;

-- name: ListProperties :many
SELECT * FROM property
ORDER BY property_id;

-- name: CreateProperty :exec
INSERT INTO property (
  property_id, listing_id, url, zipcode, city, state, location
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
);

-- name: PutProperty :exec
UPDATE property
  SET url = $3,
  zipcode = $4,
  city = $5,
  state = $6,
  location = $7,
  list_price = $8,
  last_scrape_ts = $9,
  last_scrape_status = $10,
  last_scrape_checksums = $11
WHERE property_id = $1 AND listing_id = $2;

-- name: UpdatePropertyStatus :exec
UPDATE property
  SET last_scrape_ts = NOW()::timestamp,
  last_scrape_status = $3
WHERE property_id = $1 AND listing_id = $2;

-- name: DeletePropertyListing :exec
DELETE FROM property
WHERE property_id = $1 AND listing_id = $2;

-- name: DeletePropertyListingsByID :exec
DELETE FROM property
WHERE property_id = $1;
