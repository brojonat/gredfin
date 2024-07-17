// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: property_events_query.sql

package dbgen

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const blocklistProperty = `-- name: BlocklistProperty :exec
INSERT INTO property_blocklist (
  url, expl
) VALUES (
  $1, $2
)
`

type BlocklistPropertyParams struct {
	URL  string      `json:"url"`
	Expl pgtype.Text `json:"expl"`
}

func (q *Queries) BlocklistProperty(ctx context.Context, arg BlocklistPropertyParams) error {
	_, err := q.db.Exec(ctx, blocklistProperty, arg.URL, arg.Expl)
	return err
}

type CreatePropertyEventParams struct {
	PropertyID       int32            `json:"property_id"`
	ListingID        int32            `json:"listing_id"`
	Price            int32            `json:"price"`
	EventDescription pgtype.Text      `json:"event_description"`
	Source           pgtype.Text      `json:"source"`
	SourceID         pgtype.Text      `json:"source_id"`
	EventTS          pgtype.Timestamp `json:"event_ts"`
}

const deleteBlocklistedProperty = `-- name: DeleteBlocklistedProperty :exec
DELETE FROM property_blocklist
WHERE url = $1
`

func (q *Queries) DeleteBlocklistedProperty(ctx context.Context, url string) error {
	_, err := q.db.Exec(ctx, deleteBlocklistedProperty, url)
	return err
}

const deletePropertyEvents = `-- name: DeletePropertyEvents :exec
DELETE FROM property_events
WHERE event_id = ANY($1::INT[])
`

func (q *Queries) DeletePropertyEvents(ctx context.Context, ids []int32) error {
	_, err := q.db.Exec(ctx, deletePropertyEvents, ids)
	return err
}

const deletePropertyEventsByProperty = `-- name: DeletePropertyEventsByProperty :exec
DELETE FROM property_events
WHERE
  (property_id = $1) AND
  (listing_id = $2 OR $2 = 0)
`

type DeletePropertyEventsByPropertyParams struct {
	PropertyID int32 `json:"property_id"`
	ListingID  int32 `json:"listing_id"`
}

func (q *Queries) DeletePropertyEventsByProperty(ctx context.Context, arg DeletePropertyEventsByPropertyParams) error {
	_, err := q.db.Exec(ctx, deletePropertyEventsByProperty, arg.PropertyID, arg.ListingID)
	return err
}

const getPropertyEvents = `-- name: GetPropertyEvents :many
SELECT event_id, property_id, listing_id, price, event_description, source, source_id, event_ts
FROM property_events
WHERE
  (property_id = $1 OR $1 = 0) AND
  (listing_id = $2 OR $2 = 0) AND
  (event_description = $3 OR $3 IS NULL OR $3 = '') AND
  (source = $4 OR $4 IS NULL OR $4 = '') AND
  (source_id = $5 OR $5 IS NULL OR $5 = '')
ORDER BY event_ts
`

type GetPropertyEventsParams struct {
	PropertyID       int32       `json:"property_id"`
	ListingID        int32       `json:"listing_id"`
	EventDescription pgtype.Text `json:"event_description"`
	Source           pgtype.Text `json:"source"`
	SourceID         pgtype.Text `json:"source_id"`
}

// GetPropertyEvents uses an exemplar pattern for SQLC filtering. FIXME: It is
// unfortunate that the current implementation relies on the zero value of the
// Go type. This might be fixable by using the pgtype.
func (q *Queries) GetPropertyEvents(ctx context.Context, arg GetPropertyEventsParams) ([]PropertyEvent, error) {
	rows, err := q.db.Query(ctx, getPropertyEvents,
		arg.PropertyID,
		arg.ListingID,
		arg.EventDescription,
		arg.Source,
		arg.SourceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []PropertyEvent
	for rows.Next() {
		var i PropertyEvent
		if err := rows.Scan(
			&i.EventID,
			&i.PropertyID,
			&i.ListingID,
			&i.Price,
			&i.EventDescription,
			&i.Source,
			&i.SourceID,
			&i.EventTS,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listBlocklistedProperties = `-- name: ListBlocklistedProperties :many
SELECT url, expl
FROM property_blocklist
WHERE url = ANY($1::VARCHAR[])
`

func (q *Queries) ListBlocklistedProperties(ctx context.Context, urls []string) ([]PropertyBlocklist, error) {
	rows, err := q.db.Query(ctx, listBlocklistedProperties, urls)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []PropertyBlocklist
	for rows.Next() {
		var i PropertyBlocklist
		if err := rows.Scan(&i.URL, &i.Expl); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
