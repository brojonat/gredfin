// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: realtor_query.sql

package dbgen

import (
	"context"

	jsonb "github.com/brojonat/gredfin/server/db/jsonb"
	"github.com/jackc/pgx/v5/pgtype"
	geometry "github.com/twpayne/go-geos/geometry"
)

const createRealtor = `-- name: CreateRealtor :exec
INSERT INTO realtor (
  name, company
) VALUES (
  $1, $2
) ON CONFLICT ON CONSTRAINT unique_person DO NOTHING
`

type CreateRealtorParams struct {
	Name    string `json:"name"`
	Company string `json:"company"`
}

func (q *Queries) CreateRealtor(ctx context.Context, arg CreateRealtorParams) error {
	_, err := q.db.Exec(ctx, createRealtor, arg.Name, arg.Company)
	return err
}

const createRealtorPropertyListing = `-- name: CreateRealtorPropertyListing :exec
INSERT INTO realtor_property_through (
  realtor_id, property_id, listing_id
) VALUES (
  $1, $2, $3
)
`

type CreateRealtorPropertyListingParams struct {
	RealtorID  int32 `json:"realtor_id"`
	PropertyID int32 `json:"property_id"`
	ListingID  int32 `json:"listing_id"`
}

func (q *Queries) CreateRealtorPropertyListing(ctx context.Context, arg CreateRealtorPropertyListingParams) error {
	_, err := q.db.Exec(ctx, createRealtorPropertyListing, arg.RealtorID, arg.PropertyID, arg.ListingID)
	return err
}

const deleteRealtor = `-- name: DeleteRealtor :exec
DELETE FROM realtor
WHERE realtor_id = $1
`

func (q *Queries) DeleteRealtor(ctx context.Context, realtorID int32) error {
	_, err := q.db.Exec(ctx, deleteRealtor, realtorID)
	return err
}

const deleteRealtorPropertyListing = `-- name: DeleteRealtorPropertyListing :exec
DELETE FROM realtor_property_through
WHERE realtor_id = $1 AND property_id = $2 AND listing_id = $3
`

type DeleteRealtorPropertyListingParams struct {
	RealtorID  int32 `json:"realtor_id"`
	PropertyID int32 `json:"property_id"`
	ListingID  int32 `json:"listing_id"`
}

func (q *Queries) DeleteRealtorPropertyListing(ctx context.Context, arg DeleteRealtorPropertyListingParams) error {
	_, err := q.db.Exec(ctx, deleteRealtorPropertyListing, arg.RealtorID, arg.PropertyID, arg.ListingID)
	return err
}

const getRealtor = `-- name: GetRealtor :one
SELECT realtor_id, name, company
FROM realtor
WHERE name = $1 AND company = $2
`

type GetRealtorParams struct {
	Name    string `json:"name"`
	Company string `json:"company"`
}

func (q *Queries) GetRealtor(ctx context.Context, arg GetRealtorParams) (Realtor, error) {
	row := q.db.QueryRow(ctx, getRealtor, arg.Name, arg.Company)
	var i Realtor
	err := row.Scan(&i.RealtorID, &i.Name, &i.Company)
	return i, err
}

const getRealtorProperties = `-- name: GetRealtorProperties :many
SELECT r.realtor_id, name, company, rp.realtor_id, rp.property_id, rp.listing_id, p.property_id, p.listing_id, price, url, zipcode, city, state, location, last_scrape_ts, last_scrape_status, last_scrape_metadata
FROM realtor r
INNER JOIN realtor_property_through rp
  ON r.realtor_id = rp.realtor_id
INNER JOIN property_price p
  ON rp.property_id = p.property_id AND rp.listing_id = p.listing_id
WHERE
  (rp.realtor_id = $1 OR $1 = 0) AND
  (r.name = $2 OR $2 = '' OR $2 IS NULL)
  -- FIXME: add a bunch more filters, this is the main query
ORDER BY r.name
`

type GetRealtorPropertiesParams struct {
	RealtorID int32  `json:"realtor_id"`
	Name      string `json:"name"`
}

type GetRealtorPropertiesRow struct {
	RealtorID          int32                        `json:"realtor_id"`
	Name               string                       `json:"name"`
	Company            string                       `json:"company"`
	RealtorID_2        int32                        `json:"realtor_id_2"`
	PropertyID         int32                        `json:"property_id"`
	ListingID          int32                        `json:"listing_id"`
	PropertyID_2       int32                        `json:"property_id_2"`
	ListingID_2        int32                        `json:"listing_id_2"`
	Price              int32                        `json:"price"`
	URL                pgtype.Text                  `json:"url"`
	Zipcode            pgtype.Text                  `json:"zipcode"`
	City               pgtype.Text                  `json:"city"`
	State              pgtype.Text                  `json:"state"`
	Location           *geometry.Geometry           `json:"location"`
	LastScrapeTS       pgtype.Timestamp             `json:"last_scrape_ts"`
	LastScrapeStatus   pgtype.Text                  `json:"last_scrape_status"`
	LastScrapeMetadata jsonb.PropertyScrapeMetadata `json:"last_scrape_metadata"`
}

func (q *Queries) GetRealtorProperties(ctx context.Context, arg GetRealtorPropertiesParams) ([]GetRealtorPropertiesRow, error) {
	rows, err := q.db.Query(ctx, getRealtorProperties, arg.RealtorID, arg.Name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetRealtorPropertiesRow
	for rows.Next() {
		var i GetRealtorPropertiesRow
		if err := rows.Scan(
			&i.RealtorID,
			&i.Name,
			&i.Company,
			&i.RealtorID_2,
			&i.PropertyID,
			&i.ListingID,
			&i.PropertyID_2,
			&i.ListingID_2,
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

const searchRealtorProperties = `-- name: SearchRealtorProperties :many
SELECT name, company, property_count, avg_price, median_price, zipcodes
FROM (
	SELECT
		rp.name, rp.company,
		COUNT(*)::INT AS "property_count",
		AVG(rp.price)::INT AS "avg_price",
		PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY rp.price)::INT AS "median_price",
		STRING_AGG(DISTINCT rp.zipcode, ',')::TEXT AS "zipcodes"
	FROM (
		SELECT pp.property_id, pp.listing_id, price, url, zipcode, city, state, location, last_scrape_ts, last_scrape_status, last_scrape_metadata, rpt.realtor_id, rpt.property_id, rpt.listing_id, r.realtor_id, name, company
		FROM property_price pp
		LEFT JOIN realtor_property_through rpt ON pp.property_id = rpt.property_id AND pp.listing_id = rpt.listing_id
		LEFT JOIN realtor r ON rpt.realtor_id = r.realtor_id
		WHERE r.name IS NOT NULL AND r.company IS NOT NULL AND pp.zipcode IS NOT NULL
	) rp
	GROUP BY rp.name, rp.company
) AS rs
WHERE
  (POSITION(LOWER($1) IN LOWER(rs.name)) > 0) OR
  (POSITION(LOWER($1) IN LOWER(rs.company)) > 0) OR
  (POSITION($1 IN rs.zipcodes) > 0)
ORDER BY rs.property_count DESC
LIMIT 100
`

type SearchRealtorPropertiesRow struct {
	Name          string `json:"name"`
	Company       string `json:"company"`
	PropertyCount int32  `json:"property_count"`
	AvgPrice      int32  `json:"avg_price"`
	MedianPrice   int32  `json:"median_price"`
	Zipcodes      string `json:"zipcodes"`
}

// List realtors with some useful aggregate data. This is like the "realtor
// stats" handler. This lets us do more aggregation on the backend and reduce
// bandwidth.
func (q *Queries) SearchRealtorProperties(ctx context.Context, search string) ([]SearchRealtorPropertiesRow, error) {
	rows, err := q.db.Query(ctx, searchRealtorProperties, search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []SearchRealtorPropertiesRow
	for rows.Next() {
		var i SearchRealtorPropertiesRow
		if err := rows.Scan(
			&i.Name,
			&i.Company,
			&i.PropertyCount,
			&i.AvgPrice,
			&i.MedianPrice,
			&i.Zipcodes,
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

const searchRealtorPropertiesLegacyREMOVEME = `-- name: SearchRealtorPropertiesLegacyREMOVEME :many
SELECT r.realtor_id, name, company, rp.realtor_id, rp.property_id, rp.listing_id, p.property_id, p.listing_id, price, url, zipcode, city, state, location, last_scrape_ts, last_scrape_status, last_scrape_metadata
FROM realtor r
INNER JOIN realtor_property_through rp
  ON r.realtor_id = rp.realtor_id
INNER JOIN property_price p
  ON rp.property_id = p.property_id AND rp.listing_id = p.listing_id
WHERE
  (POSITION(LOWER($1) IN LOWER(r.name)) > 0) OR
  (POSITION(LOWER($1) IN LOWER(r.company)) > 0)
ORDER BY r.name
LIMIT 100
`

type SearchRealtorPropertiesLegacyREMOVEMERow struct {
	RealtorID          int32                        `json:"realtor_id"`
	Name               string                       `json:"name"`
	Company            string                       `json:"company"`
	RealtorID_2        int32                        `json:"realtor_id_2"`
	PropertyID         int32                        `json:"property_id"`
	ListingID          int32                        `json:"listing_id"`
	PropertyID_2       int32                        `json:"property_id_2"`
	ListingID_2        int32                        `json:"listing_id_2"`
	Price              int32                        `json:"price"`
	URL                pgtype.Text                  `json:"url"`
	Zipcode            pgtype.Text                  `json:"zipcode"`
	City               pgtype.Text                  `json:"city"`
	State              pgtype.Text                  `json:"state"`
	Location           *geometry.Geometry           `json:"location"`
	LastScrapeTS       pgtype.Timestamp             `json:"last_scrape_ts"`
	LastScrapeStatus   pgtype.Text                  `json:"last_scrape_status"`
	LastScrapeMetadata jsonb.PropertyScrapeMetadata `json:"last_scrape_metadata"`
}

func (q *Queries) SearchRealtorPropertiesLegacyREMOVEME(ctx context.Context, search string) ([]SearchRealtorPropertiesLegacyREMOVEMERow, error) {
	rows, err := q.db.Query(ctx, searchRealtorPropertiesLegacyREMOVEME, search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []SearchRealtorPropertiesLegacyREMOVEMERow
	for rows.Next() {
		var i SearchRealtorPropertiesLegacyREMOVEMERow
		if err := rows.Scan(
			&i.RealtorID,
			&i.Name,
			&i.Company,
			&i.RealtorID_2,
			&i.PropertyID,
			&i.ListingID,
			&i.PropertyID_2,
			&i.ListingID_2,
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
