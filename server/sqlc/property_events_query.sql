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

-- GetPropertyEvents uses an exemplar pattern for SQLC filtering. FIXME: It is
-- unfortunate that the current implementation relies on the zero value of the
-- Go type. This might be fixable by using the pgtype but I don't have time
-- right now.
-- name: GetPropertyEvents :many
SELECT *
FROM property_events
WHERE
  (property_id = @property_id OR @property_id = 0) AND
  (listing_id = @listing_id OR @listing_id = 0) AND
  (event_description = @event_description OR @event_description IS NULL OR @event_description = '') AND
  (source = @source OR @source IS NULL OR @source = '') AND
  (source_id = @source_id OR @source_id IS NULL OR @source_id = '')
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