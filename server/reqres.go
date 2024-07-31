package server

import "github.com/brojonat/gredfin/server/db/dbgen"

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

type Location struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

type CreatePropertyParams struct {
	dbgen.CreatePropertyParams
	Location Location `json:"location"`
}

// This needs to check for any types that contain *geom.Point fields and make
// them serializable because it has proven to be annoyingly difficult to have
// sqlc, pgx, and go-geom to use a serializable type location data. This should
// be refactored at some point, but for now, this is worth it to unblock
// progress. In order to fix this, we should be able to implement our own
// pgtype.Codec and register that on the connection instead of using the
// existing pgxgeom.Register function. Or we could make a PR for adding a
// Geometry package to go-geom.
func makeLocationSerializable(v any) any {
	switch qr := v.(type) {
	// query response was a single Property
	case dbgen.Property:
		type serialPrice struct {
			dbgen.Property
			Location Location `json:"location"`
		}
		return serialPrice{qr, Location{Type: "Point", Coordinates: qr.Location.Coords()}}
	// query response was a single PropertyPrice
	case dbgen.PropertyPrice:
		type serialPrice struct {
			dbgen.PropertyPrice
			Location Location `json:"location"`
		}
		return serialPrice{qr, Location{Type: "Point", Coordinates: qr.Location.Coords()}}
	// query response was a list of PropertyPrice
	case []dbgen.PropertyPrice:
		type serialPrice struct {
			dbgen.PropertyPrice
			Location Location `json:"location"`
		}
		res := []serialPrice{}
		for _, p := range qr {
			res = append(res, serialPrice{p, Location{Type: "Point", Coordinates: p.Location.Coords()}})
		}
		return res
	default:
		return v
	}
}
