-- name: GetPropertyWithPrice :one
SELECT *
FROM property_price
WHERE property_id = $1 AND listing_id = $2
LIMIT 1;

-- name: GetPropertyBasic :one
SELECT *
FROM property
WHERE property_id = $1 AND listing_id = $2
LIMIT 1;

-- FIXME: It is unfortunate that the current implementation relies on the zero
-- value of the Go type. This might be fixable by using the pgtype.
-- name: GetPropertiesBasic :many
SELECT *
FROM property
WHERE
  (property_id = @property_id OR @property_id = 0) AND
  (listing_id = @listing_id OR @listing_id = 0) AND
  (last_scrape_status = @last_scrape_status OR @last_scrape_status IS NULL OR @last_scrape_status = '')
ORDER BY property_id;

-- FIXME: It is unfortunate that the current implementation relies on the zero
-- value of the Go type. This might be fixable by using the pgtype.
-- name: GetPropertiesWithPrice :many
SELECT *
FROM property_price
WHERE
  (property_id = @property_id OR @property_id = 0) AND
  (listing_id = @listing_id OR @listing_id = 0) AND
  (last_scrape_status = @last_scrape_status OR @last_scrape_status IS NULL OR @last_scrape_status = '')
ORDER BY property_id;

-- name: GetNNextPropertyScrapeForUpdate :one
-- Get the next N property entries that have a last_scrape_status in the
-- supplied slice. Rows are locked for update; callers are expected to set
-- status rows to PENDING after retrieving rows. Note that this query uses
-- the "basic" property table, and NOT the property_price view because
-- callers may expect this to return properties with no price events.
SELECT *
FROM property
WHERE last_scrape_status = ANY(sqlc.arg(statuses)::VARCHAR[])
ORDER BY NOW()::timestamp - last_scrape_ts DESC
LIMIT sqlc.arg(count)
FOR UPDATE;

-- name: ListProperties :many
SELECT *
FROM property_price
ORDER BY property_id;

-- name: ListPropertiesPrices :many
SELECT *
FROM property_price p
ORDER BY p.property_id;

-- name: CreateProperty :exec
INSERT INTO property (
  property_id, listing_id, url, location
) VALUES (
  $1, $2, $3, $4
);

-- name: PutProperty :exec
UPDATE property
  SET url = $3,
  zipcode = $4,
  city = $5,
  state = $6,
  location = $7,
  last_scrape_ts = $8,
  last_scrape_status = $9,
  last_scrape_checksums = $10
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
