package server

type DefaultJSONResponse struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type PostRealtorBody struct {
	Name       string `json:"name"`
	Company    string `json:"company"`
	PropertyID int32  `json:"property_id"`
	ListingID  int32  `json:"listing_id"`
}
