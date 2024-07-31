package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/brojonat/gredfin/server/db/dbgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func handleRealtorGet(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		realtorID := r.URL.Query().Get("id")
		name := r.URL.Query().Get("name")
		search := r.URL.Query().Get("search")

		// no identifiers, return listings based on search (if unspecified, all
		// rows will be returned)
		if realtorID == "" && name == "" {
			rs, err := q.SearchRealtorProperties(r.Context(), search)
			if rs == nil || err == pgx.ErrNoRows {
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

		// realtor_id specified, return listings under that id
		if realtorID != "" {
			rid, err := strconv.Atoi(realtorID)
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			rs, err := q.GetRealtorProperties(r.Context(), dbgen.GetRealtorPropertiesParams{RealtorID: int32(rid)})
			if rs == nil || err == pgx.ErrNoRows {
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

		// name specified, return listings under that name
		rs, err := q.GetRealtorProperties(r.Context(), dbgen.GetRealtorPropertiesParams{Name: name})
		if rs == nil || err == pgx.ErrNoRows {
			writeEmptyResultError(w)
			return
		}
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(makeLocationSerializable(rs))
	}
}

func handleRealtorPost(l *slog.Logger, p *pgxpool.Pool, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var data PostRealtorBody
		err := decodeJSONBody(r, &data)
		if err != nil {
			var mr *MalformedRequest
			if errors.As(err, &mr) {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "bad request"})
			} else {
				writeInternalError(l, w, err)
			}
			return
		}

		// start a transaction to create the realtor and the through table entry
		tx, err := p.Begin(r.Context())
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		defer tx.Commit(r.Context())
		q = q.WithTx(tx)

		// Ignore conflicts quietly since we expect workers to spam this route
		// with possible duplicates. This query should "do nothing" on conflicts
		// anyway.
		realtor, err := q.GetRealtor(r.Context(), dbgen.GetRealtorParams{Name: data.Name, Company: data.Company})
		if err == pgx.ErrNoRows {
			err = q.CreateRealtor(r.Context(), dbgen.CreateRealtorParams{Name: data.Name, Company: data.Company})
			if err != nil && !isPGError(err, pgErrorUniqueViolation) {
				writeInternalError(l, w, err)
				return
			}
		}
		realtor, err = q.GetRealtor(r.Context(), dbgen.GetRealtorParams{Name: data.Name, Company: data.Company})
		if err != nil {
			writeInternalError(l, w, err)
			return
		}

		err = q.CreateRealtorPropertyListing(r.Context(), dbgen.CreateRealtorPropertyListingParams{
			RealtorID: realtor.RealtorID, PropertyID: data.PropertyID, ListingID: data.ListingID})
		if err != nil && !isPGError(err, pgErrorUniqueViolation) {
			writeInternalError(l, w, err)
			return
		}
		writeOK(w)
	}

}

func handleRealtorDelete(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		realtor_id := r.URL.Query().Get("realtor_id")
		rid, err := strconv.Atoi(realtor_id)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "bad value for realtor_id"})
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
			writeOK(w)
			return
		}

		// bad request
		if propertyID == "" || listingID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "must supply property_id and listing_id"})
			return
		}

		// delete single entry
		pid, err := strconv.Atoi(propertyID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "bad value for property_id"})
			return
		}
		lid, err := strconv.Atoi(listingID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "bad value for listing_id"})
			return
		}
		err = q.DeletePropertyListing(r.Context(), dbgen.DeletePropertyListingParams{PropertyID: int32(pid), ListingID: int32(lid)})
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		writeOK(w)
	}
}
