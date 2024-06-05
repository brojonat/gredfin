// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0
// source: realtor_query.sql

package dbgen

import (
	"context"
)

const createRealtor = `-- name: CreateRealtor :exec
INSERT INTO realtor (
  name, company, property_id, listing_id, list_price, created_ts
) VALUES (
  $1, $2, $3, $4, $5, NOW()::timestamp
) ON CONFLICT ON CONSTRAINT realtor_pkey DO NOTHING
`

type CreateRealtorParams struct {
	Name       string `json:"name"`
	Company    string `json:"company"`
	PropertyID int32  `json:"property_id"`
	ListingID  int32  `json:"listing_id"`
	ListPrice  int32  `json:"list_price"`
}

func (q *Queries) CreateRealtor(ctx context.Context, arg CreateRealtorParams) error {
	_, err := q.db.Exec(ctx, createRealtor,
		arg.Name,
		arg.Company,
		arg.PropertyID,
		arg.ListingID,
		arg.ListPrice,
	)
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

const deleteRealtorListing = `-- name: DeleteRealtorListing :exec
DELETE FROM realtor
WHERE realtor_id = $1 AND property_id = $2 AND listing_id = $3
`

type DeleteRealtorListingParams struct {
	RealtorID  int32 `json:"realtor_id"`
	PropertyID int32 `json:"property_id"`
	ListingID  int32 `json:"listing_id"`
}

func (q *Queries) DeleteRealtorListing(ctx context.Context, arg DeleteRealtorListingParams) error {
	_, err := q.db.Exec(ctx, deleteRealtorListing, arg.RealtorID, arg.PropertyID, arg.ListingID)
	return err
}

const getRealtorProperties = `-- name: GetRealtorProperties :many
SELECT realtor_id, name, company, property_id, listing_id, list_price, created_ts FROM realtor
WHERE realtor_id = $1
`

func (q *Queries) GetRealtorProperties(ctx context.Context, realtorID int32) ([]Realtor, error) {
	rows, err := q.db.Query(ctx, getRealtorProperties, realtorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Realtor
	for rows.Next() {
		var i Realtor
		if err := rows.Scan(
			&i.RealtorID,
			&i.Name,
			&i.Company,
			&i.PropertyID,
			&i.ListingID,
			&i.ListPrice,
			&i.CreatedTs,
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

const getRealtorPropertiesByName = `-- name: GetRealtorPropertiesByName :many
SELECT realtor_id, name, company, property_id, listing_id, list_price, created_ts FROM realtor
WHERE name = $1
`

func (q *Queries) GetRealtorPropertiesByName(ctx context.Context, name string) ([]Realtor, error) {
	rows, err := q.db.Query(ctx, getRealtorPropertiesByName, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Realtor
	for rows.Next() {
		var i Realtor
		if err := rows.Scan(
			&i.RealtorID,
			&i.Name,
			&i.Company,
			&i.PropertyID,
			&i.ListingID,
			&i.ListPrice,
			&i.CreatedTs,
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

const listRealtors = `-- name: ListRealtors :many
SELECT realtor_id, name, company, property_id, listing_id, list_price, created_ts FROM realtor
ORDER BY name
`

func (q *Queries) ListRealtors(ctx context.Context) ([]Realtor, error) {
	rows, err := q.db.Query(ctx, listRealtors)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Realtor
	for rows.Next() {
		var i Realtor
		if err := rows.Scan(
			&i.RealtorID,
			&i.Name,
			&i.Company,
			&i.PropertyID,
			&i.ListingID,
			&i.ListPrice,
			&i.CreatedTs,
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

const postRealtor = `-- name: PostRealtor :exec
UPDATE realtor
  SET name = $2,
  company = $3,
  property_id = $4,
  listing_id = $5,
  list_price = $6
WHERE realtor_id = $1
`

type PostRealtorParams struct {
	RealtorID  int32  `json:"realtor_id"`
	Name       string `json:"name"`
	Company    string `json:"company"`
	PropertyID int32  `json:"property_id"`
	ListingID  int32  `json:"listing_id"`
	ListPrice  int32  `json:"list_price"`
}

func (q *Queries) PostRealtor(ctx context.Context, arg PostRealtorParams) error {
	_, err := q.db.Exec(ctx, postRealtor,
		arg.RealtorID,
		arg.Name,
		arg.Company,
		arg.PropertyID,
		arg.ListingID,
		arg.ListPrice,
	)
	return err
}
