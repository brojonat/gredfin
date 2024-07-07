package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/brojonat/gredfin/server/db/dbgen"
	"github.com/jackc/pgx/v5"
)

// Writes a list of { price, count } objects representing realtor's binned prices
func handlePlotDataRealtorPrices(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v := r.URL.Query().Get("version")
		name := r.URL.Query().Get("name")
		if name == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "must supply realtor name"})
			return
		}

		switch v {
		// This case returns a list of { price } objects representing the
		// realtor's prices.
		case "", "1":
			events, err := q.GetRealtorProperties(r.Context(), dbgen.GetRealtorPropertiesParams{Name: name})
			if err != nil {
				if err == pgx.ErrNoRows {
					w.WriteHeader(http.StatusNotFound)
					json.NewEncoder(w).Encode(DefaultJSONResponse{Error: fmt.Sprintf("no properties for realtor %s", name)})
					return
				}
				writeInternalError(l, w, err)
				return
			}
			type rr struct {
				Price float64 `json:"price"`
			}
			res := []rr{}
			for _, e := range events {
				res = append(res, rr{float64(e.Price)})
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(res)
			return
		default:
			writeBadRequestError(w, fmt.Errorf("unsupported version: %s", v))
			return
		}
	}
}

// Writes a list of { datetime, price } objects representing historical property prices. This should
// be well suited to the flutterflow and SF_CartesianChart API.
func handlePlotDataPropertyPrices(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v := r.URL.Query().Get("version")
		propertyID := r.URL.Query().Get("property_id")
		pid, err := strconv.Atoi(propertyID)
		if err != nil {
			writeBadRequestError(w, fmt.Errorf("bad value for property_id"))
			return
		}
		events, err := q.GetPropertyEvents(r.Context(), dbgen.GetPropertyEventsParams{PropertyID: int32(pid)})
		if err != nil {
			writeInternalError(l, w, err)
			return
		}

		switch v {
		case "", "1":
			// This data version will iterate over the events and send the
			// timestamp and price if the price is non-zero.
			type chartData struct {
				X time.Time `json:"timestamp"`
				Y int32     `json:"price"`
			}
			res := []chartData{}
			for _, e := range events {
				if e.Price == 0 {
					continue
				}
				evt := chartData{
					X: e.EventTS.Time,
					Y: e.Price,
				}
				res = append(res, evt)
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(res)
			return
		default:
			writeBadRequestError(w, fmt.Errorf("unsupported version: %s", v))
			return
		}
	}
}
