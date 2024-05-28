package redfin

import "encoding/json"

type RedfinResponse struct {
	Version      int             `json:"version"`
	ErrorMessage string          `json:"errorMessage"`
	ResultCode   int             `json:"resultCode"`
	Payload      json.RawMessage `json:"payload"`
}
type InitialInfoPayload struct {
	ResponseCode int `json:"responseCode"`
	ListingID    int `json:"listingId"`
	PropertyID   int `json:"propertyId"`
}
type SearchPayload struct {
	Sections         []Sections   `json:"sections"`
	ExactMatch       ExactMatch   `json:"exactMatch"`
	ExtraResults     ExtraResults `json:"extraResults"`
	ResponseTime     int          `json:"responseTime"`
	HasFakeResults   bool         `json:"hasFakeResults"`
	IsGeocoded       bool         `json:"isGeocoded"`
	IsRedfinServiced bool         `json:"isRedfinServiced"`
}
type Sections struct {
	Rows []Rows `json:"rows"`
	Name string `json:"name"`
}
type Rows struct {
	ID                string `json:"id"`
	Type              string `json:"type"`
	Name              string `json:"name"`
	SubName           string `json:"subName"`
	URL               string `json:"url"`
	URLV2             string `json:"urlV2"`
	Active            bool   `json:"active"`
	ClaimedHome       bool   `json:"claimedHome"`
	InvalidMRS        bool   `json:"invalidMRS"`
	BusinessMarketIds []int  `json:"businessMarketIds"`
	CountryCode       string `json:"countryCode"`
	SearchStatusID    int    `json:"searchStatusId"`
	HasRental         bool   `json:"hasRental"`
}
type ExactMatch struct {
	ID                string `json:"id"`
	Type              string `json:"type"`
	Name              string `json:"name"`
	SubName           string `json:"subName"`
	URL               string `json:"url"`
	URLV2             string `json:"urlV2"`
	Active            bool   `json:"active"`
	ClaimedHome       bool   `json:"claimedHome"`
	InvalidMRS        bool   `json:"invalidMRS"`
	BusinessMarketIds []int  `json:"businessMarketIds"`
	CountryCode       string `json:"countryCode"`
	SearchStatusID    int    `json:"searchStatusId"`
	HasRental         bool   `json:"hasRental"`
}
type ExtraResults struct {
}
