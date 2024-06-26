-- name: GetRealtorProperties :many
SELECT
  r.realtor_id,
  r.name,
  r.company,
  r.created_ts,
  p.property_id,
  p.listing_id,
  p.url,
  p.zipcode,
  p.city,
  p.state,
  p.location,
  p.price
FROM realtor r
INNER JOIN property_price p
  ON r.property_id = p.property_id AND r.listing_id = p.listing_id
WHERE realtor_id = $1;

-- name: GetRealtorPropertiesByName :many
SELECT
  r.realtor_id,
  r.name,
  r.company,
  r.created_ts,
  p.property_id,
  p.listing_id,
  p.url,
  p.zipcode,
  p.city,
  p.state,
  p.location,
  p.price
FROM realtor r
INNER JOIN property_price p
  ON r.property_id = p.property_id AND r.listing_id = p.listing_id
WHERE name = $1;

-- name: GetRealtorPropertiesFullByName :many
SELECT
  r.realtor_id,
  r.name,
  r.company,
  r.created_ts,
  p.property_id,
  p.listing_id,
  p.url,
  p.zipcode,
  p.city,
  p.state,
  p.location,
  p.price
FROM realtor r
INNER JOIN property_price p
  ON r.property_id = p.property_id AND r.listing_id = p.listing_id
WHERE name = $1;

-- name: ListRealtors :many
SELECT
  r.realtor_id,
  r.name,
  r.company,
  r.created_ts,
  p.property_id,
  p.listing_id,
  p.url,
  p.zipcode,
  p.city,
  p.state,
  p.location,
  p.price
FROM realtor r
INNER JOIN property_price p
  ON r.property_id = p.property_id AND r.listing_id = p.listing_id
ORDER BY name;

-- name: CreateRealtor :exec
INSERT INTO realtor (
  name, company, property_id, listing_id, created_ts
) VALUES (
  $1, $2, $3, $4, NOW()::timestamp
) ON CONFLICT ON CONSTRAINT realtor_pkey DO NOTHING;

-- name: PostRealtor :exec
UPDATE realtor
  SET name = $2,
  company = $3,
  property_id = $4,
  listing_id = $5
WHERE realtor_id = $1;

-- name: DeleteRealtorListing :exec
DELETE FROM realtor
WHERE realtor_id = $1 AND property_id = $2 AND listing_id = $3;

-- name: DeleteRealtor :exec
DELETE FROM realtor
WHERE realtor_id = $1;
