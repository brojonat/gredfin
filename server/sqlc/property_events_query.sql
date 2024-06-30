-- name: BlocklistProperty :exec
INSERT INTO property_blocklist (
  url, expl
) VALUES (
  $1, $2
);

-- name: ListBlocklistedProperties :many
SELECT *
FROM property_blocklist
WHERE url = ANY(sqlc.arg(urls)::VARCHAR[]);

-- name: DeleteBlocklistedProperty :exec
DELETE FROM property_blocklist
WHERE url = $1;

-- GetPropertyEvents uses an exemplar pattern for SQLC filtering.
-- name: GetPropertyEvents :many
SELECT *
FROM property_events
WHERE
  (property_id = @property_id OR @property_id IS NULL) OR
  (listing_id = @listing_id OR @listing_id IS NULL) OR
  (event_description = @event_description OR @event_description IS NULL) OR
  (source = @source OR @source IS NULL) OR
  (source_id = @source_id OR @source_id IS NULL)
ORDER BY event_ts;

-- name: CreatePropertyEvent :copyfrom
INSERT INTO property_events (
  property_id, listing_id, price, event_description, source, source_id, event_ts
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
);

-- name: DeletePropertyEvent :exec
DELETE FROM property_events
WHERE event_id = ANY(sqlc.arg(ids)::INT[]);