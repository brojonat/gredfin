// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0

package dbgen

import (
	jsonb "github.com/brojonat/gredfin/server/dbgen/jsonb"
	"github.com/jackc/pgx/v5/pgtype"
	geos "github.com/twpayne/go-geos"
)

type Property struct {
	PropertyID          int32                        `json:"property_id"`
	ListingID           int32                        `json:"listing_id"`
	URL                 pgtype.Text                  `json:"url"`
	Zipcode             pgtype.Text                  `json:"zipcode"`
	City                pgtype.Text                  `json:"city"`
	State               pgtype.Text                  `json:"state"`
	Location            *geos.Geom                   `json:"location"`
	ListPrice           int                          `json:"list_price"`
	LastScrapeTs        pgtype.Timestamp             `json:"last_scrape_ts"`
	LastScrapeStatus    pgtype.Text                  `json:"last_scrape_status"`
	LastScrapeChecksums jsonb.PropertyScrapeMetadata `json:"last_scrape_checksums"`
}

type Realtor struct {
	RealtorID  int32            `json:"realtor_id"`
	Name       string           `json:"name"`
	Company    string           `json:"company"`
	PropertyID int32            `json:"property_id"`
	ListingID  int32            `json:"listing_id"`
	CreatedTs  pgtype.Timestamp `json:"created_ts"`
}

type Search struct {
	SearchID         int32            `json:"search_id"`
	Query            pgtype.Text      `json:"query"`
	LastScrapeTs     pgtype.Timestamp `json:"last_scrape_ts"`
	LastScrapeStatus string           `json:"last_scrape_status"`
}
