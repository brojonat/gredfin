// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0
// source: property_query.sql

package dbgen

import (
	"context"

	jsonb "github.com/brojonat/gredfin/server/dbgen/jsonb"
	"github.com/jackc/pgx/v5/pgtype"
	geometry "github.com/twpayne/go-geos/geometry"
)

const createProperty = `-- name: CreateProperty :exec
INSERT INTO property (
  property_id, listing_id, url, location
) VALUES (
  $1, $2, $3, $4
)
`

type CreatePropertyParams struct {
	PropertyID int32              `json:"property_id"`
	ListingID  int32              `json:"listing_id"`
	URL        pgtype.Text        `json:"url"`
	Location   *geometry.Geometry `json:"location"`
}

func (q *Queries) CreateProperty(ctx context.Context, arg CreatePropertyParams) error {
	_, err := q.db.Exec(ctx, createProperty,
		arg.PropertyID,
		arg.ListingID,
		arg.URL,
		arg.Location,
	)
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
SELECT property_id, listing_id, url, zipcode, city, state, location, last_scrape_ts, last_scrape_status, last_scrape_metadata
FROM property
WHERE last_scrape_status = ANY($1::VARCHAR[])
ORDER BY NOW()::timestamp - last_scrape_ts DESC
LIMIT $2
FOR UPDATE
`

type GetNNextPropertyScrapeForUpdateParams struct {
	Statuses []string `json:"statuses"`
	Count    int32    `json:"count"`
}

// Get the next N property entries that have a last_scrape_status in the
// supplied slice. Rows are locked for update; callers are expected to set
// status rows to PENDING after retrieving rows. Note that this query uses
// the "basic" property table, and NOT the property_price view because
// callers may expect this to return properties with no price events.
func (q *Queries) GetNNextPropertyScrapeForUpdate(ctx context.Context, arg GetNNextPropertyScrapeForUpdateParams) (Property, error) {
	row := q.db.QueryRow(ctx, getNNextPropertyScrapeForUpdate, arg.Statuses, arg.Count)
	var i Property
	err := row.Scan(
		&i.PropertyID,
		&i.ListingID,
		&i.URL,
		&i.Zipcode,
		&i.City,
		&i.State,
		&i.Location,
		&i.LastScrapeTS,
		&i.LastScrapeStatus,
		&i.LastScrapeMetadata,
	)
	return i, err
}

const getPropertiesBasic = `-- name: GetPropertiesBasic :many
SELECT property_id, listing_id, url, zipcode, city, state, location, last_scrape_ts, last_scrape_status, last_scrape_metadata
FROM property
WHERE
  (property_id = $1 OR $1 = 0) AND
  (listing_id = $2 OR $2 = 0) AND
  (last_scrape_status = $3 OR $3 IS NULL OR $3 = '')
ORDER BY property_id
`

type GetPropertiesBasicParams struct {
	PropertyID       int32       `json:"property_id"`
	ListingID        int32       `json:"listing_id"`
	LastScrapeStatus pgtype.Text `json:"last_scrape_status"`
}

// FIXME: It is unfortunate that the current implementation relies on the zero
// value of the Go type. This might be fixable by using the pgtype.
func (q *Queries) GetPropertiesBasic(ctx context.Context, arg GetPropertiesBasicParams) ([]Property, error) {
	rows, err := q.db.Query(ctx, getPropertiesBasic, arg.PropertyID, arg.ListingID, arg.LastScrapeStatus)
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
			&i.Zipcode,
			&i.City,
			&i.State,
			&i.Location,
			&i.LastScrapeTS,
			&i.LastScrapeStatus,
			&i.LastScrapeMetadata,
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

const getPropertiesWithPrice = `-- name: GetPropertiesWithPrice :many
SELECT property_id, listing_id, price, url, zipcode, city, state, location, last_scrape_ts, last_scrape_status, last_scrape_metadata
FROM property_price
WHERE
  (property_id = $1 OR $1 = 0) AND
  (listing_id = $2 OR $2 = 0) AND
  (last_scrape_status = $3 OR $3 IS NULL OR $3 = '')
ORDER BY property_id
`

type GetPropertiesWithPriceParams struct {
	PropertyID       int32       `json:"property_id"`
	ListingID        int32       `json:"listing_id"`
	LastScrapeStatus pgtype.Text `json:"last_scrape_status"`
}

// FIXME: It is unfortunate that the current implementation relies on the zero
// value of the Go type. This might be fixable by using the pgtype.
func (q *Queries) GetPropertiesWithPrice(ctx context.Context, arg GetPropertiesWithPriceParams) ([]PropertyPrice, error) {
	rows, err := q.db.Query(ctx, getPropertiesWithPrice, arg.PropertyID, arg.ListingID, arg.LastScrapeStatus)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []PropertyPrice
	for rows.Next() {
		var i PropertyPrice
		if err := rows.Scan(
			&i.PropertyID,
			&i.ListingID,
			&i.Price,
			&i.URL,
			&i.Zipcode,
			&i.City,
			&i.State,
			&i.Location,
			&i.LastScrapeTS,
			&i.LastScrapeStatus,
			&i.LastScrapeMetadata,
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

const getPropertyBasic = `-- name: GetPropertyBasic :one
SELECT property_id, listing_id, url, zipcode, city, state, location, last_scrape_ts, last_scrape_status, last_scrape_metadata
FROM property
WHERE property_id = $1 AND listing_id = $2
LIMIT 1
`

type GetPropertyBasicParams struct {
	PropertyID int32 `json:"property_id"`
	ListingID  int32 `json:"listing_id"`
}

func (q *Queries) GetPropertyBasic(ctx context.Context, arg GetPropertyBasicParams) (Property, error) {
	row := q.db.QueryRow(ctx, getPropertyBasic, arg.PropertyID, arg.ListingID)
	var i Property
	err := row.Scan(
		&i.PropertyID,
		&i.ListingID,
		&i.URL,
		&i.Zipcode,
		&i.City,
		&i.State,
		&i.Location,
		&i.LastScrapeTS,
		&i.LastScrapeStatus,
		&i.LastScrapeMetadata,
	)
	return i, err
}

const getPropertyWithPrice = `-- name: GetPropertyWithPrice :one
SELECT property_id, listing_id, price, url, zipcode, city, state, location, last_scrape_ts, last_scrape_status, last_scrape_metadata
FROM property_price
WHERE property_id = $1 AND listing_id = $2
LIMIT 1
`

type GetPropertyWithPriceParams struct {
	PropertyID int32 `json:"property_id"`
	ListingID  int32 `json:"listing_id"`
}

func (q *Queries) GetPropertyWithPrice(ctx context.Context, arg GetPropertyWithPriceParams) (PropertyPrice, error) {
	row := q.db.QueryRow(ctx, getPropertyWithPrice, arg.PropertyID, arg.ListingID)
	var i PropertyPrice
	err := row.Scan(
		&i.PropertyID,
		&i.ListingID,
		&i.Price,
		&i.URL,
		&i.Zipcode,
		&i.City,
		&i.State,
		&i.Location,
		&i.LastScrapeTS,
		&i.LastScrapeStatus,
		&i.LastScrapeMetadata,
	)
	return i, err
}

const listProperties = `-- name: ListProperties :many
SELECT property_id, listing_id, price, url, zipcode, city, state, location, last_scrape_ts, last_scrape_status, last_scrape_metadata
FROM property_price
ORDER BY property_id
`

func (q *Queries) ListProperties(ctx context.Context) ([]PropertyPrice, error) {
	rows, err := q.db.Query(ctx, listProperties)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []PropertyPrice
	for rows.Next() {
		var i PropertyPrice
		if err := rows.Scan(
			&i.PropertyID,
			&i.ListingID,
			&i.Price,
			&i.URL,
			&i.Zipcode,
			&i.City,
			&i.State,
			&i.Location,
			&i.LastScrapeTS,
			&i.LastScrapeStatus,
			&i.LastScrapeMetadata,
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

const listPropertiesPrices = `-- name: ListPropertiesPrices :many
SELECT property_id, listing_id, price, url, zipcode, city, state, location, last_scrape_ts, last_scrape_status, last_scrape_metadata
FROM property_price p
ORDER BY p.property_id
`

func (q *Queries) ListPropertiesPrices(ctx context.Context) ([]PropertyPrice, error) {
	rows, err := q.db.Query(ctx, listPropertiesPrices)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []PropertyPrice
	for rows.Next() {
		var i PropertyPrice
		if err := rows.Scan(
			&i.PropertyID,
			&i.ListingID,
			&i.Price,
			&i.URL,
			&i.Zipcode,
			&i.City,
			&i.State,
			&i.Location,
			&i.LastScrapeTS,
			&i.LastScrapeStatus,
			&i.LastScrapeMetadata,
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

const putProperty = `-- name: PutProperty :exec
UPDATE property
  SET url = $3,
  zipcode = $4,
  city = $5,
  state = $6,
  location = $7,
  last_scrape_ts = $8,
  last_scrape_status = $9,
  last_scrape_metadata = $10
WHERE property_id = $1 AND listing_id = $2
`

type PutPropertyParams struct {
	PropertyID         int32                        `json:"property_id"`
	ListingID          int32                        `json:"listing_id"`
	URL                pgtype.Text                  `json:"url"`
	Zipcode            pgtype.Text                  `json:"zipcode"`
	City               pgtype.Text                  `json:"city"`
	State              pgtype.Text                  `json:"state"`
	Location           *geometry.Geometry           `json:"location"`
	LastScrapeTS       pgtype.Timestamp             `json:"last_scrape_ts"`
	LastScrapeStatus   pgtype.Text                  `json:"last_scrape_status"`
	LastScrapeMetadata jsonb.PropertyScrapeMetadata `json:"last_scrape_metadata"`
}

func (q *Queries) PutProperty(ctx context.Context, arg PutPropertyParams) error {
	_, err := q.db.Exec(ctx, putProperty,
		arg.PropertyID,
		arg.ListingID,
		arg.URL,
		arg.Zipcode,
		arg.City,
		arg.State,
		arg.Location,
		arg.LastScrapeTS,
		arg.LastScrapeStatus,
		arg.LastScrapeMetadata,
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
