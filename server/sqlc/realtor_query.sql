-- name: GetRealtor :one
SELECT * FROM realtor
WHERE realtor_id = $1 LIMIT 1;

-- name: GetRealtorsByName :many
SELECT * FROM realtor
WHERE realtor_name = $1;

-- name: ListRealtors :many
SELECT * FROM realtor
ORDER BY realtor_name;

-- name: CreateRealtor :one
INSERT INTO realtor (
  realtor_id, realtor_name, realtor_region, property_id, listing_id, list_price
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: PostRealtor :exec
UPDATE realtor
  SET realtor_name = $2,
  realtor_region = $3,
  property_id = $4,
  listing_id = $5,
  list_price = $6
WHERE realtor_id = $1;

-- name: DeleteRealtor :exec
DELETE FROM realtor
WHERE realtor_id = $1;
