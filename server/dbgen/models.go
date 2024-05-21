// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0

package dbgen

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type Property struct {
	PropertyID         string           `json:"property_id"`
	ListingID          string           `json:"listing_id"`
	Address            pgtype.Text      `json:"address"`
	Zipcode            pgtype.Text      `json:"zipcode"`
	State              pgtype.Text      `json:"state"`
	LastScrapeTs       pgtype.Timestamp `json:"last_scrape_ts"`
	LastScrapeStatus   pgtype.Text      `json:"last_scrape_status"`
	LastScrapeChecksum pgtype.Text      `json:"last_scrape_checksum"`
}

type Realtor struct {
	RealtorID     int32       `json:"realtor_id"`
	RealtorName   pgtype.Text `json:"realtor_name"`
	RealtorRegion pgtype.Text `json:"realtor_region"`
	PropertyID    string      `json:"property_id"`
	ListingID     string      `json:"listing_id"`
	ListPrice     pgtype.Int4 `json:"list_price"`
}

type Search struct {
	SearchID         int32            `json:"search_id"`
	Query            pgtype.Text      `json:"query"`
	LastScrapeTs     pgtype.Timestamp `json:"last_scrape_ts"`
	LastScrapeStatus pgtype.Text      `json:"last_scrape_status"`
}
