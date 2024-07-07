// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: search_query.sql

package dbgen

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createSearch = `-- name: CreateSearch :exec
INSERT INTO search (
  query
) VALUES (
  $1
)
`

func (q *Queries) CreateSearch(ctx context.Context, query pgtype.Text) error {
	_, err := q.db.Exec(ctx, createSearch, query)
	return err
}

const deleteSearch = `-- name: DeleteSearch :exec
DELETE FROM search
WHERE search_id = $1
`

func (q *Queries) DeleteSearch(ctx context.Context, searchID int32) error {
	_, err := q.db.Exec(ctx, deleteSearch, searchID)
	return err
}

const deleteSearchByQuery = `-- name: DeleteSearchByQuery :exec
DELETE FROM search
WHERE query = $1
`

func (q *Queries) DeleteSearchByQuery(ctx context.Context, query pgtype.Text) error {
	_, err := q.db.Exec(ctx, deleteSearchByQuery, query)
	return err
}

const getNNextSearchScrapeForUpdate = `-- name: GetNNextSearchScrapeForUpdate :one
SELECT search_id, query, last_scrape_ts, last_scrape_status FROM search
WHERE last_scrape_status = ANY($1::VARCHAR[])
ORDER BY NOW()::timestamp - last_scrape_ts DESC
LIMIT $2
FOR UPDATE
`

type GetNNextSearchScrapeForUpdateParams struct {
	Statuses []string `json:"statuses"`
	Count    int32    `json:"count"`
}

// Get the next N property entries that have a last_scrape_status in the
// supplied slice. Rows are locked for update; callers are expected to set
// status rows to PENDING after retrieving rows.
func (q *Queries) GetNNextSearchScrapeForUpdate(ctx context.Context, arg GetNNextSearchScrapeForUpdateParams) (Search, error) {
	row := q.db.QueryRow(ctx, getNNextSearchScrapeForUpdate, arg.Statuses, arg.Count)
	var i Search
	err := row.Scan(
		&i.SearchID,
		&i.Query,
		&i.LastScrapeTS,
		&i.LastScrapeStatus,
	)
	return i, err
}

const getSearch = `-- name: GetSearch :one
SELECT search_id, query, last_scrape_ts, last_scrape_status FROM search
WHERE search_id = $1 LIMIT 1
`

func (q *Queries) GetSearch(ctx context.Context, searchID int32) (Search, error) {
	row := q.db.QueryRow(ctx, getSearch, searchID)
	var i Search
	err := row.Scan(
		&i.SearchID,
		&i.Query,
		&i.LastScrapeTS,
		&i.LastScrapeStatus,
	)
	return i, err
}

const getSearchByQuery = `-- name: GetSearchByQuery :one
SELECT search_id, query, last_scrape_ts, last_scrape_status FROM search
WHERE query = $1
`

func (q *Queries) GetSearchByQuery(ctx context.Context, query pgtype.Text) (Search, error) {
	row := q.db.QueryRow(ctx, getSearchByQuery, query)
	var i Search
	err := row.Scan(
		&i.SearchID,
		&i.Query,
		&i.LastScrapeTS,
		&i.LastScrapeStatus,
	)
	return i, err
}

const listSearches = `-- name: ListSearches :many
SELECT search_id, query, last_scrape_ts, last_scrape_status FROM search
ORDER BY search_id
`

func (q *Queries) ListSearches(ctx context.Context) ([]Search, error) {
	rows, err := q.db.Query(ctx, listSearches)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Search
	for rows.Next() {
		var i Search
		if err := rows.Scan(
			&i.SearchID,
			&i.Query,
			&i.LastScrapeTS,
			&i.LastScrapeStatus,
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

const postSearch = `-- name: PostSearch :exec
UPDATE search
  SET query = $2,
  last_scrape_ts = $3,
  last_scrape_status = $4
WHERE search_id = $1
`

type PostSearchParams struct {
	SearchID         int32            `json:"search_id"`
	Query            pgtype.Text      `json:"query"`
	LastScrapeTS     pgtype.Timestamp `json:"last_scrape_ts"`
	LastScrapeStatus string           `json:"last_scrape_status"`
}

func (q *Queries) PostSearch(ctx context.Context, arg PostSearchParams) error {
	_, err := q.db.Exec(ctx, postSearch,
		arg.SearchID,
		arg.Query,
		arg.LastScrapeTS,
		arg.LastScrapeStatus,
	)
	return err
}

const updateSearchStatus = `-- name: UpdateSearchStatus :exec
UPDATE search
  SET last_scrape_ts = NOW()::timestamp,
  last_scrape_status = $2
WHERE search_id = $1
`

type UpdateSearchStatusParams struct {
	SearchID         int32  `json:"search_id"`
	LastScrapeStatus string `json:"last_scrape_status"`
}

func (q *Queries) UpdateSearchStatus(ctx context.Context, arg UpdateSearchStatusParams) error {
	_, err := q.db.Exec(ctx, updateSearchStatus, arg.SearchID, arg.LastScrapeStatus)
	return err
}
