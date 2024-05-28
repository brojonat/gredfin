package worker

import "net/http"

const (
	// default region/type for Columbus, OH
	defaultRegionID   = "18063"
	defaultRegionType = "2" // 2 is _maybe_ zipcode?
)

// Return the default headers to use to make queries against the server
func getDefaultServerHeaders(authToken string) http.Header {
	h := http.Header{}
	h.Add("Authorization", authToken)
	h.Add("Content-Type", "application/json")
	return h
}

// Return the default parameters for the Redfin GIS-CSV query. Note that callers
// still need to override the "region_id" and "region_type" fields. The default
// values are set for demo/testing purposes and correspond to Columbus, OH.
func getDefaultGISCSVParams() map[string]string {
	params := map[string]string{}
	params["al"] = "1"
	params["has_att_fiber"] = "false"
	params["has_deal"] = "false"
	params["has_dishwasher"] = "false"
	params["has_laundry_facility"] = "false"
	params["has_laundry_hookups"] = "false"
	params["has_parking"] = "false"
	params["has_pool"] = "false"
	params["has_short_term_lease"] = "false"
	params["include_pending_homes"] = "false"
	params["isRentals"] = "false"
	params["is_furnished"] = "false"
	params["is_income_restricted"] = "false"
	params["is_senior_living"] = "false"
	params["num_homes"] = "350"
	params["ord"] = "redfin-recommended-asc"
	params["page_number"] = "1"
	params["pool"] = "false"
	params["region_id"] = defaultRegionID
	params["region_type"] = defaultRegionType
	params["sf"] = "1,2,3,5,6,7"
	params["status"] = "9"
	params["travel_with_traffic"] = "false"
	params["travel_within_region"] = "false"
	params["uipt"] = "1,2,3,4,5,6,7,8"
	params["utilities_included"] = "false"
	params["v"] = "8"
	return params
}

// Return the default parameters for the Redfin search query
func getDefaultSearchParams() map[string]string {
	params := map[string]string{}
	return params
}
