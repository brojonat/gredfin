// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0
// source: property_query.sql

package dbgen

import (
	"context"

	jsonb "github.com/brojonat/gredfin/server/dbgen/jsonb"
	"github.com/jackc/pgx/v5/pgtype"
)

const createProperty = `-- name: CreateProperty :exec
INSERT INTO property (
  property_id, listing_id, url
) VALUES (
  $1, $2, $3
)
`

type CreatePropertyParams struct {
	PropertyID int32       `json:"property_id"`
	ListingID  int32       `json:"listing_id"`
	URL        pgtype.Text `json:"url"`
}

func (q *Queries) CreateProperty(ctx context.Context, arg CreatePropertyParams) error {
	_, err := q.db.Exec(ctx, createProperty, arg.PropertyID, arg.ListingID, arg.URL)
	return err
}

const deletePropertyListing = `-- name: DeletePropertyListing :exec
DELETE FROM property
WHERE property_id = $1 AND listing_id = $2
`

type DeletePropertyListingParams struct {
	PropertyID int32 `json:"property_id"`
	ListingID  int32 `json:"listing_id"`
}

func (q *Queries) DeletePropertyListing(ctx context.Context, arg DeletePropertyListingParams) error {
	_, err := q.db.Exec(ctx, deletePropertyListing, arg.PropertyID, arg.ListingID)
	return err
}

const deletePropertyListingsByID = `-- name: DeletePropertyListingsByID :exec
DELETE FROM property
WHERE property_id = $1
`

func (q *Queries) DeletePropertyListingsByID(ctx context.Context, propertyID int32) error {
	_, err := q.db.Exec(ctx, deletePropertyListingsByID, propertyID)
	return err
}

const getNNextPropertyScrapeForUpdate = `-- name: GetNNextPropertyScrapeForUpdate :one
SELECT property_id, listing_id, url, last_scrape_ts, last_scrape_status, last_scrape_checksums FROM property
WHERE last_scrape_status = ANY($2::VARCHAR[])
ORDER BY NOW()::timestamp - last_scrape_ts
LIMIT $1
FOR UPDATE
`

type GetNNextPropertyScrapeForUpdateParams struct {
	Limit   int32    `json:"limit"`
	Column2 []string `json:"column_2"`
}

// Get the next N property entries that have a last_scrape_status in the
// supplied slice. Rows are locked for update; callers are expected to set
// status rows to PENDING after retrieving rows.
func (q *Queries) GetNNextPropertyScrapeForUpdate(ctx context.Context, arg GetNNextPropertyScrapeForUpdateParams) (Property, error) {
	row := q.db.QueryRow(ctx, getNNextPropertyScrapeForUpdate, arg.Limit, arg.Column2)
	var i Property
	err := row.Scan(
		&i.PropertyID,
		&i.ListingID,
		&i.URL,
		&i.LastScrapeTs,
		&i.LastScrapeStatus,
		&i.LastScrapeChecksums,
	)
	return i, err
}

const getPropertiesByID = `-- name: GetPropertiesByID :many
SELECT property_id, listing_id, url, last_scrape_ts, last_scrape_status, last_scrape_checksums FROM property
WHERE property_id = $1
`

func (q *Queries) GetPropertiesByID(ctx context.Context, propertyID int32) ([]Property, error) {
	rows, err := q.db.Query(ctx, getPropertiesByID, propertyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Property
	for rows.Next() {
		var i Property
		if err := rows.Scan(
			&i.PropertyID,
			&i.ListingID,
			&i.URL,
			&i.LastScrapeTs,
			&i.LastScrapeStatus,
			&i.LastScrapeChecksums,
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

const getProperty = `-- name: GetProperty :one
SELECT property_id, listing_id, url, last_scrape_ts, last_scrape_status, last_scrape_checksums FROM property
WHERE property_id = $1 AND listing_id = $2
LIMIT 1
`

type GetPropertyParams struct {
	PropertyID int32 `json:"property_id"`
	ListingID  int32 `json:"listing_id"`
}

func (q *Queries) GetProperty(ctx context.Context, arg GetPropertyParams) (Property, error) {
	row := q.db.QueryRow(ctx, getProperty, arg.PropertyID, arg.ListingID)
	var i Property
	err := row.Scan(
		&i.PropertyID,
		&i.ListingID,
		&i.URL,
		&i.LastScrapeTs,
		&i.LastScrapeStatus,
		&i.LastScrapeChecksums,
	)
	return i, err
}

const listProperties = `-- name: ListProperties :many
SELECT property_id, listing_id, url, last_scrape_ts, last_scrape_status, last_scrape_checksums FROM property
ORDER BY property_id
`

func (q *Queries) ListProperties(ctx context.Context) ([]Property, error) {
	rows, err := q.db.Query(ctx, listProperties)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Property
	for rows.Next() {
		var i Property
		if err := rows.Scan(
			&i.PropertyID,
			&i.ListingID,
			&i.URL,
			&i.LastScrapeTs,
			&i.LastScrapeStatus,
			&i.LastScrapeChecksums,
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

const postProperty = `-- name: PostProperty :exec
UPDATE property
  SET url = $3,
  last_scrape_ts = $4,
  last_scrape_status = $5,
  last_scrape_checksums = $6
WHERE property_id = $1 AND listing_id = $2
`

type PostPropertyParams struct {
	PropertyID          int32                        `json:"property_id"`
	ListingID           int32                        `json:"listing_id"`
	URL                 pgtype.Text                  `json:"url"`
	LastScrapeTs        pgtype.Timestamp             `json:"last_scrape_ts"`
	LastScrapeStatus    pgtype.Text                  `json:"last_scrape_status"`
	LastScrapeChecksums jsonb.PropertyScrapeMetadata `json:"last_scrape_checksums"`
}

func (q *Queries) PostProperty(ctx context.Context, arg PostPropertyParams) error {
	_, err := q.db.Exec(ctx, postProperty,
		arg.PropertyID,
		arg.ListingID,
		arg.URL,
		arg.LastScrapeTs,
		arg.LastScrapeStatus,
		arg.LastScrapeChecksums,
	)
	return err
}

const updatePropertyStatus = `-- name: UpdatePropertyStatus :exec
UPDATE property
  SET last_scrape_ts = NOW()::timestamp,
  last_scrape_status = $3
WHERE property_id = $1 AND listing_id = $2
`

type UpdatePropertyStatusParams struct {
	PropertyID       int32       `json:"property_id"`
	ListingID        int32       `json:"listing_id"`
	LastScrapeStatus pgtype.Text `json:"last_scrape_status"`
}

func (q *Queries) UpdatePropertyStatus(ctx context.Context, arg UpdatePropertyStatusParams) error {
	_, err := q.db.Exec(ctx, updatePropertyStatus, arg.PropertyID, arg.ListingID, arg.LastScrapeStatus)
	return err
}
