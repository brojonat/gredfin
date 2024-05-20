package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/brojonat/gredfin/server/dbgen"
	"github.com/jackc/pgx/v5/pgtype"
)

func handleRealtorGet(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		realtor_id := r.URL.Query().Get("realtor_id")
		realtor_name := r.URL.Query().Get("realtor_name")

		// no identifiers, return whole listing
		if realtor_id == "" && realtor_name == "" {
			rs, err := q.ListRealtors(r.Context())
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(rs)
			return
		}

		// realtor_id specified, return realtor entries under that ID
		if realtor_id != "" {
			id, err := strconv.Atoi(realtor_id)
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			rs, err := q.GetRealtorProperties(r.Context(), int32(id))
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(rs)
			return
		}

		// realtor_name specified, return realtor entries under that name
		// NOTE: there may be multiple realtors with the same name!
		if realtor_name != "" {
			rs, err := q.GetRealtorPropertiesByName(r.Context(), pgtype.Text{String: realtor_name})
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(rs)
			return
		}

	}
}

func handleRealtorPost(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p dbgen.CreateRealtorParams
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
		err = q.CreateRealtor(r.Context(), p)
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(defaultJSONResponse{Message: "ok"})
	}
}

func handleRealtorDelete(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		realtor_id := r.URL.Query().Get("realtor_id")
		rid, err := strconv.Atoi(realtor_id)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(defaultJSONResponse{Error: "bad value for realtor_id"})
			return
		}
		property_id := r.URL.Query().Get("property_id")
		listing_id := r.URL.Query().Get("listing_id")

		// delete all entries for this realtor
		if property_id != "" && listing_id == "" {
			err := q.DeleteRealtor(r.Context(), int32(rid))
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(defaultJSONResponse{Message: "ok"})
			return
		}

		// bad request
		if property_id == "" || listing_id == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(defaultJSONResponse{Error: "must supply property_id and listing_id"})
			return
		}

		// delete single entry
		err = q.DeletePropertyListing(r.Context(), dbgen.DeletePropertyListingParams{PropertyID: property_id, ListingID: listing_id})
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(defaultJSONResponse{Message: "ok"})
	}
}
