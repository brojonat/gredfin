package redfin

import (
	"io"
	"net/http"
)

type Client interface {
	// url requests
	InitialInfo(url string, params map[string]string) ([]byte, error)
	PageTags(url string, params map[string]string) ([]byte, error)
	PrimaryRegion(url string, params map[string]string) ([]byte, error)

	// search
	Search(query string, params map[string]string) ([]byte, error)
	GISCSV(params map[string]string) ([]byte, error)

	// property id requests
	BelowTheFold(propertyID string, params map[string]string) ([]byte, error)
	HoodPhotos(propertyID string, params map[string]string) ([]byte, error)
	MoreResources(propertyID string, params map[string]string) ([]byte, error)
	PageHeader(propertyID string, params map[string]string) ([]byte, error)
	PropertyComments(propertyID string, params map[string]string) ([]byte, error)
	BuildingDetailsPage(propertyID string, params map[string]string) ([]byte, error)
	OwnerEstimate(propertyID string, params map[string]string) ([]byte, error)
	ClaimedHomeSellerData(propertyID string, params map[string]string) ([]byte, error)
	CostOfHomeOwnership(propertyID string, params map[string]string) ([]byte, error)

	// listing id requests
	FloorPlans(propertyID string, params map[string]string) ([]byte, error)
	TourListDatePicker(propertyID string, params map[string]string) ([]byte, error)

	// table id requests
	SharedRegion(tableID string, params map[string]string) ([]byte, error)

	// property requests
	SimilarListings(propertyID string, listingID string, params map[string]string) ([]byte, error)
	SimilarSold(propertyID string, listingID string, params map[string]string) ([]byte, error)
	NearbyHomes(propertyID string, listingID string, params map[string]string) ([]byte, error)
	AboveTheFold(propertyID string, listingID string, params map[string]string) ([]byte, error)
	PropertyParcel(propertyID string, listingID string, params map[string]string) ([]byte, error)
	Activity(propertyID string, listingID string, params map[string]string) ([]byte, error)
	CustomerConversionInfoOffMarket(propertyID string, listingID string, params map[string]string) ([]byte, error)
	RentalEstimate(propertyID string, listingID string, params map[string]string) ([]byte, error)
	AVMHistorical(propertyID string, listingID string, params map[string]string) ([]byte, error)
	InfoPanel(propertyID string, listingID string, params map[string]string) ([]byte, error)
	DescriptiveParagraph(propertyID string, listingID string, params map[string]string) ([]byte, error)
	AVMDetails(propertyID string, listingID string, params map[string]string) ([]byte, error)
	TourInsights(propertyID string, listingID string, params map[string]string) ([]byte, error)
	Stats(propertyID string, listingID string, regionID string, params map[string]string) ([]byte, error)
}

type client struct {
	baseURL   string
	userAgent string
}

func NewClient(baseURL, userAgent string) Client {
	return &client{baseURL: baseURL, userAgent: userAgent}
}

func (c *client) doPropertyRequest(path string, params map[string]string, page bool) ([]byte, error) {
	if page {
		params["pageType"] = "3"
	}
	params["accessLevel"] = "1"
	return c.doRequest("api/home/details"+path, params)
}

func (c *client) doRequest(url string, params map[string]string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+url, nil)
	req.Header.Add("User-Agent", c.userAgent)
	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	// possibly handle gzip encoding IF the server returns this for the gis query
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	// the responses are prefixed with {}& before valid JSON
	return b[4:], nil
}

// url requests
func (c *client) InitialInfo(url string, params map[string]string) ([]byte, error) {
	params["path"] = url
	return c.doRequest("api/home/details/initialInfo", params)
}

func (c *client) PageTags(url string, params map[string]string) ([]byte, error) {
	params["path"] = url
	return c.doRequest("api/home/details/v1/pagetagsinfo", params)
}

func (c *client) PrimaryRegion(url string, params map[string]string) ([]byte, error) {
	params["path"] = url
	return c.doRequest("api/home/details/primaryRegionInfo", params)
}

// search
func (c *client) Search(query string, params map[string]string) ([]byte, error) {
	params["location"] = query
	params["v"] = "2"
	return c.doRequest("do/location-autocomplete", params)
}

func (c *client) GISCSV(params map[string]string) ([]byte, error) {
	return c.doRequest("api/gis-csv", params)
}

// property id requests
func (c *client) BelowTheFold(propertyID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	return c.doPropertyRequest("belowTheFold", params, true)
}

func (c *client) HoodPhotos(propertyID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	return c.doRequest("api/home/details/hood-photos", params)
}

func (c *client) MoreResources(propertyID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	return c.doRequest("api/home/details/moreResourcesInfo", params)
}

func (c *client) PageHeader(propertyID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	return c.doRequest("api/home/details/homeDetailsPageHeaderInfo", params)
}

func (c *client) PropertyComments(propertyID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	return c.doRequest("api/v1/home/details/propertyCommentsInfo", params)
}

func (c *client) BuildingDetailsPage(propertyID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	return c.doRequest("api/building/details-page/v1", params)
}

func (c *client) OwnerEstimate(propertyID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	return c.doRequest("api/home/details/owner-estimate", params)
}

func (c *client) ClaimedHomeSellerData(propertyID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	return c.doRequest("api/home/details/claimedHomeSellerData", params)
}

func (c *client) CostOfHomeOwnership(propertyID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	return c.doRequest("do/api/costOfHomeOwnershipDetails", params)
}

// listing id requests
func (c *client) FloorPlans(listingID string, params map[string]string) ([]byte, error) {
	params["listingId"] = listingID
	return c.doRequest("api/home/details/listing/floorplans", params)
}

func (c *client) TourListDatePicker(listingID string, params map[string]string) ([]byte, error) {
	params["listingId"] = listingID
	return c.doRequest("do/tourlist/getDatePickerData", params)
}

// table id requests
func (c *client) SharedRegion(tableID string, params map[string]string) ([]byte, error) {
	params["tableId"] = tableID
	params["regionTypeId"] = "2"
	params["mapPageTypeId"] = "1"
	return c.doRequest("api/region/shared-region-info", params)
}

// property requests
func (c *client) SimilarListings(propertyID string, listingID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	params["listingId"] = listingID
	return c.doPropertyRequest("/similars/listings", params, false)
}

func (c *client) SimilarSold(propertyID string, listingID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	params["listingId"] = listingID
	return c.doPropertyRequest("/similars/solds", params, false)
}

func (c *client) NearbyHomes(propertyID string, listingID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	params["listingId"] = listingID
	return c.doPropertyRequest("/nearbyhomes", params, false)
}

func (c *client) AboveTheFold(propertyID string, listingID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	params["listingId"] = listingID
	return c.doPropertyRequest("/aboveTheFold", params, false)
}

func (c *client) PropertyParcel(propertyID string, listingID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	params["listingId"] = listingID
	return c.doPropertyRequest("/propertyParcelInfo", params, true)
}

func (c *client) Activity(propertyID string, listingID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	params["listingId"] = listingID
	return c.doPropertyRequest("/activityInfo", params, false)
}

func (c *client) CustomerConversionInfoOffMarket(propertyID string, listingID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	params["listingId"] = listingID
	return c.doPropertyRequest("/customerConversionInfo/offMarket", params, true)
}

func (c *client) RentalEstimate(propertyID string, listingID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	params["listingId"] = listingID
	return c.doPropertyRequest("/rental-estimate", params, false)
}

func (c *client) AVMHistorical(propertyID string, listingID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	params["listingId"] = listingID
	return c.doPropertyRequest("/avmHistoricalData", params, false)
}

func (c *client) InfoPanel(propertyID string, listingID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	params["listingId"] = listingID
	return c.doPropertyRequest("/mainHouseInfoPanelInfo", params, false)
}

func (c *client) DescriptiveParagraph(propertyID string, listingID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	params["listingId"] = listingID
	return c.doPropertyRequest("/descriptiveParagraph", params, false)
}

func (c *client) AVMDetails(propertyID string, listingID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	params["listingId"] = listingID
	return c.doPropertyRequest("/avm", params, false)
}

func (c *client) TourInsights(propertyID string, listingID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	params["listingId"] = listingID
	return c.doPropertyRequest("/tourInsights", params, true)
}

func (c *client) Stats(propertyID string, listingID string, regionID string, params map[string]string) ([]byte, error) {
	params["propertyId"] = propertyID
	params["listingId"] = listingID
	params["regionId"] = regionID
	return c.doPropertyRequest("/stats", params, false)
}
