package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/brojonat/gredfin/server/dbgen"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func handlePropertyQueryGet(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propertyID := r.URL.Query().Get("property_id")
		listingID := r.URL.Query().Get("listing_id")

		// no identifier, list properties
		if propertyID == "" && listingID == "" {
			props, err := q.ListProperties(r.Context())
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(props)
			return
		}

		// no propertyID with a listingID is a bad request
		if propertyID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(defaultJSONResponse{Error: "missing listing_id"})
			return
		}

		// no listingID, return a listing of properties
		if listingID == "" {
			props, err := q.GetPropertiesByID(r.Context(), propertyID)
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(props)
			return
		}

		// return single entry
		prop, err := q.GetProperty(r.Context(), dbgen.GetPropertyParams{PropertyID: propertyID, ListingID: listingID})
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(prop)
	}
}

func handlePropertyQueryPost(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p dbgen.CreatePropertyParams
		err := decodeJSONBody(r, &p)
		if err != nil {
			var mr *MalformedRequest
			if errors.As(err, &mr) {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(defaultJSONResponse{Error: "bad request"})
			} else {
				writeInternalError(l, w, err)
			}
			return
		}
		err = q.CreateProperty(r.Context(), p)
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(defaultJSONResponse{Message: "ok"})
	}
}

func handlePropertyQueryDelete(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propertyID := r.URL.Query().Get("property_id")
		listingID := r.URL.Query().Get("listing_id")

		if propertyID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(defaultJSONResponse{Error: "must supply property_id"})
			return
		}

		// no listingID, delete all property entries under the ID
		if listingID == "" {
			err := q.DeletePropertyListingsByID(r.Context(), propertyID)
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(defaultJSONResponse{Message: "ok"})
			return
		}

		// delete property listing
		err := q.DeletePropertyListing(
			r.Context(),
			dbgen.DeletePropertyListingParams{
				PropertyID: propertyID,
				ListingID:  listingID,
			},
		)
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(defaultJSONResponse{Message: "ok"})
	}
}

// claims the next property to be scraped and sets the status to pending
func handlePropertyQueryClaimNext(l *slog.Logger, p *pgxpool.Pool, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tx, err := p.Begin(r.Context())
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		q = q.WithTx(tx)
		prop, err := q.GetNNextPropertyScrapeForUpdate(
			r.Context(),
			dbgen.GetNNextPropertyScrapeForUpdateParams{
				Limit: 1, Column2: []string{scrapeStatusGood}},
		)
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		q.UpdatePropertyStatus(
			r.Context(),
			dbgen.UpdatePropertyStatusParams{
				PropertyID:       prop.PropertyID,
				ListingID:        prop.ListingID,
				LastScrapeStatus: pgtype.Text{String: scrapeStatusPending},
			},
		)
		tx.Commit(r.Context())
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(prop)
	}
}

func handlePropertySetStatus(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		property_id := r.URL.Query().Get("property_id")
		listing_id := r.URL.Query().Get("listing_id")
		status := r.URL.Query().Get("status")

		if property_id == "" || listing_id == "" || status == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(defaultJSONResponse{Error: "must specify property_id, listing_id, and status"})
			return
		}

		err := q.UpdatePropertyStatus(
			r.Context(),
			dbgen.UpdatePropertyStatusParams{
				PropertyID:       property_id,
				ListingID:        listing_id,
				LastScrapeStatus: pgtype.Text{String: status},
			},
		)
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(defaultJSONResponse{Message: "ok"})
	}
}
