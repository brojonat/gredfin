-- name: GetRealtorProperties :many
SELECT * FROM realtor
WHERE realtor_id = $1;

-- name: GetRealtorPropertiesByName :many
SELECT * FROM realtor
WHERE name = $1;

-- name: ListRealtors :many
SELECT * FROM realtor
ORDER BY name;

-- name: CreateRealtor :exec
INSERT INTO realtor (
  name, company, property_id, listing_id, list_price, created_ts
) VALUES (
  $1, $2, $3, $4, $5, NOW()::timestamp
) ON CONFLICT ON CONSTRAINT realtor_pkey DO NOTHING;

-- name: PostRealtor :exec
UPDATE realtor
  SET name = $2,
  company = $3,
  property_id = $4,
  listing_id = $5,
  list_price = $6
WHERE realtor_id = $1;

-- name: DeleteRealtorListing :exec
DELETE FROM realtor
WHERE realtor_id = $1 AND property_id = $2 AND listing_id = $3;

-- name: DeleteRealtor :exec
DELETE FROM realtor
WHERE realtor_id = $1;
