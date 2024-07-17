-- name: GetRealtor :one
SELECT *
FROM realtor
WHERE name = @name AND company = @company;

-- name: GetRealtorProperties :many
SELECT *
FROM realtor r
INNER JOIN realtor_property_through rp
  ON r.realtor_id = rp.realtor_id
INNER JOIN property_price p
  ON rp.property_id = p.property_id AND rp.listing_id = p.listing_id
WHERE
  (rp.realtor_id = @realtor_id OR @realtor_id = 0) AND
  (r.name = @name OR @name = '' OR @name IS NULL)
  -- FIXME: add a bunch more filters, this is the main query
ORDER BY r.name;

-- name: SearchRealtorProperties :many
SELECT *
FROM realtor r
INNER JOIN realtor_property_through rp
  ON r.realtor_id = rp.realtor_id
INNER JOIN property_price p
  ON rp.property_id = p.property_id AND rp.listing_id = p.listing_id
WHERE
  (POSITION(LOWER(@search) IN LOWER(r.name)) > 0) OR
  (POSITION(LOWER(@search) IN LOWER(r.company)) > 0)
ORDER BY r.name
LIMIT 100;

-- name: CreateRealtor :exec
INSERT INTO realtor (
  name, company
) VALUES (
  @name, @company
) ON CONFLICT ON CONSTRAINT unique_person DO NOTHING;

-- name: DeleteRealtor :exec
DELETE FROM realtor
WHERE realtor_id = $1;

-- name: CreateRealtorPropertyListing :exec
INSERT INTO realtor_property_through (
  realtor_id, property_id, listing_id
) VALUES (
  @realtor_id, @property_id, @listing_id
);

-- name: DeleteRealtorPropertyListing :exec
DELETE FROM realtor_property_through
WHERE realtor_id = @realtor_id AND property_id = @property_id AND listing_id = @listing_id;
