package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/brojonat/gredfin/server/db/dbgen"
	"github.com/brojonat/histogram"
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
		// This case returns a [(x1, y1), ...] that can be used in StepLineSeries
		case "", "1":
			// get the data
			ps, err := q.GetRealtorProperties(r.Context(), dbgen.GetRealtorPropertiesParams{Name: name})
			if ps == nil || err == pgx.ErrNoRows {
				writeEmptyResultError(w)
				return
			}
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			// histogram the prices
			prices := []float64{}
			for _, p := range ps {
				prices = append(prices, float64(p.Price))
			}
			bs, err := histogram.BSExactSpan(10)(prices)
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			h, err := histogram.Hist(prices, bs, histogram.DefaultBucketer)
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			// write the output
			type bin struct {
				Price float64 `json:"price"`
				Count int     `json:"count"`
			}
			bins := []bin{}
			for _, b := range h.Buckets {
				bins = append(bins, bin{Price: b.Min, Count: b.Count})
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(bins)
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
				X string `json:"timestamp"`
				Y int32  `json:"price"`
			}
			res := []chartData{}
			for _, e := range events {
				if e.Price == 0 {
					continue
				}
				evt := chartData{
					X: e.EventTS.Time.Format(time.RFC3339),
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
