package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/brojonat/gredfin/server/dbgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func handleRealtorGet(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		realtor_id := r.URL.Query().Get("realtor_id")
		realtor_name := r.URL.Query().Get("realtor_name")

		// no identifiers, return whole listing
		if realtor_id == "" && realtor_name == "" {
			rs, err := q.ListRealtors(r.Context())
			if err == pgx.ErrNoRows {
				writeEmptyResultError(w)
				return
			}
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
			if err == pgx.ErrNoRows {
				writeEmptyResultError(w)
				return
			}
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
			rs, err := q.GetRealtorPropertiesByName(r.Context(), pgtype.Text{String: realtor_name, Valid: true})
			if err == pgx.ErrNoRows {
				writeEmptyResultError(w)
				return
			}
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
		propertyID := r.URL.Query().Get("property_id")
		listingID := r.URL.Query().Get("listing_id")

		// delete all entries for this realtor
		if propertyID != "" && listingID == "" {
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
		if propertyID == "" || listingID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(defaultJSONResponse{Error: "must supply property_id and listing_id"})
			return
		}

		// delete single entry
		pid, err := strconv.Atoi(propertyID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(defaultJSONResponse{Error: "bad value for property_id"})
			return
		}
		lid, err := strconv.Atoi(listingID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(defaultJSONResponse{Error: "bad value for listing_id"})
			return
		}
		err = q.DeletePropertyListing(r.Context(), dbgen.DeletePropertyListingParams{PropertyID: int32(pid), ListingID: int32(lid)})
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(defaultJSONResponse{Message: "ok"})
	}
}
