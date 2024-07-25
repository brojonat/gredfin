-- name: GetRealtor :one
SELECT *
FROM realtor
WHERE name = @name AND company = @company;

-- name: SearchRealtorProperties :many
-- List realtors with some useful aggregate data. This is like the "realtor
-- stats" handler. This lets us do more aggregation on the backend and reduce
-- bandwidth.
SELECT *
FROM (
	SELECT
		rp.name, rp.company,
		COUNT(*)::INT AS "property_count",
		AVG(rp.price)::INT AS "avg_price",
		PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY rp.price)::INT AS "median_price",
		STRING_AGG(DISTINCT rp.zipcode, ',')::TEXT AS "zipcodes"
	FROM (
		SELECT *
		FROM property_price pp
		LEFT JOIN realtor_property_through rpt ON pp.property_id = rpt.property_id AND pp.listing_id = rpt.listing_id
		LEFT JOIN realtor r ON rpt.realtor_id = r.realtor_id
		WHERE r.name IS NOT NULL AND r.company IS NOT NULL AND pp.zipcode IS NOT NULL
	) rp
	GROUP BY rp.name, rp.company
) AS rs
WHERE
  (POSITION(LOWER(@search) IN LOWER(rs.name)) > 0) OR
  (POSITION(LOWER(@search) IN LOWER(rs.company)) > 0) OR
  (POSITION(@search IN rs.zipcodes) > 0)
ORDER BY rs.property_count DESC
LIMIT 100;

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
