// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0

package dbgen

import (
	jsonb "github.com/brojonat/gredfin/server/dbgen/jsonb"
	"github.com/jackc/pgx/v5/pgtype"
	geometry "github.com/twpayne/go-geos/geometry"
)

type LastPropertyPriceEvent struct {
	EventID          pgtype.Int4      `json:"event_id"`
	PropertyID       int32            `json:"property_id"`
	ListingID        int32            `json:"listing_id"`
	Price            int32            `json:"price"`
	EventDescription string           `json:"event_description"`
	Source           pgtype.Text      `json:"source"`
	SourceID         pgtype.Text      `json:"source_id"`
	EventTS          pgtype.Timestamp `json:"event_ts"`
}

type Property struct {
	PropertyID          int32                        `json:"property_id"`
	ListingID           int32                        `json:"listing_id"`
	URL                 pgtype.Text                  `json:"url"`
	Zipcode             pgtype.Text                  `json:"zipcode"`
	City                pgtype.Text                  `json:"city"`
	State               pgtype.Text                  `json:"state"`
	Location            *geometry.Geometry           `json:"location"`
	LastScrapeTS        pgtype.Timestamp             `json:"last_scrape_ts"`
	LastScrapeStatus    pgtype.Text                  `json:"last_scrape_status"`
	LastScrapeChecksums jsonb.PropertyScrapeMetadata `json:"last_scrape_checksums"`
}

type PropertyBlocklist struct {
	URL  string      `json:"url"`
	Expl pgtype.Text `json:"expl"`
}

type PropertyEvent struct {
	EventID          pgtype.Int4      `json:"event_id"`
	PropertyID       int32            `json:"property_id"`
	ListingID        int32            `json:"listing_id"`
	Price            int32            `json:"price"`
	EventDescription pgtype.Text      `json:"event_description"`
	Source           pgtype.Text      `json:"source"`
	SourceID         pgtype.Text      `json:"source_id"`
	EventTS          pgtype.Timestamp `json:"event_ts"`
}

type PropertyPrice struct {
	PropertyID          int32                        `json:"property_id"`
	ListingID           int32                        `json:"listing_id"`
	Price               int32                        `json:"price"`
	URL                 pgtype.Text                  `json:"url"`
	Zipcode             pgtype.Text                  `json:"zipcode"`
	City                pgtype.Text                  `json:"city"`
	State               pgtype.Text                  `json:"state"`
	Location            *geometry.Geometry           `json:"location"`
	LastScrapeTS        pgtype.Timestamp             `json:"last_scrape_ts"`
	LastScrapeStatus    pgtype.Text                  `json:"last_scrape_status"`
	LastScrapeChecksums jsonb.PropertyScrapeMetadata `json:"last_scrape_checksums"`
}

type Realtor struct {
	RealtorID int32  `json:"realtor_id"`
	Name      string `json:"name"`
	Company   string `json:"company"`
}

type RealtorPropertyThrough struct {
	RealtorID  int32 `json:"realtor_id"`
	PropertyID int32 `json:"property_id"`
	ListingID  int32 `json:"listing_id"`
}

type Search struct {
	SearchID         int32            `json:"search_id"`
	Query            pgtype.Text      `json:"query"`
	LastScrapeTS     pgtype.Timestamp `json:"last_scrape_ts"`
	LastScrapeStatus string           `json:"last_scrape_status"`
}
