// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: copyfrom.go

package dbgen

import (
	"context"
)

// iteratorForCreatePropertyEvent implements pgx.CopyFromSource.
type iteratorForCreatePropertyEvent struct {
	rows                 []CreatePropertyEventParams
	skippedFirstNextCall bool
}

func (r *iteratorForCreatePropertyEvent) Next() bool {
	if len(r.rows) == 0 {
		return false
	}
	if !r.skippedFirstNextCall {
		r.skippedFirstNextCall = true
		return true
	}
	r.rows = r.rows[1:]
	return len(r.rows) > 0
}

func (r iteratorForCreatePropertyEvent) Values() ([]interface{}, error) {
	return []interface{}{
		r.rows[0].PropertyID,
		r.rows[0].ListingID,
		r.rows[0].Price,
		r.rows[0].EventDescription,
		r.rows[0].Source,
		r.rows[0].SourceID,
		r.rows[0].EventTS,
	}, nil
}

func (r iteratorForCreatePropertyEvent) Err() error {
	return nil
}

func (q *Queries) CreatePropertyEvent(ctx context.Context, arg []CreatePropertyEventParams) (int64, error) {
	return q.db.CopyFrom(ctx, []string{"property_events"}, []string{"property_id", "listing_id", "price", "event_description", "source", "source_id", "event_ts"}, &iteratorForCreatePropertyEvent{rows: arg})
}
