package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/brojonat/gredfin/server/dbgen"
	"github.com/jackc/pgx/v5"
)

// helper interface for searching realtors while we tinker with the underlying inmplementation
func searchRealtors(ctx context.Context, q *dbgen.Queries, s string) ([]dbgen.ListRealtorsRow, error) {
	if s == "" {
		return q.ListRealtors(ctx)
	}
	rs, err := q.ListRealtors(ctx)
	if err != nil {
		return nil, err
	}

	keep := []dbgen.ListRealtorsRow{}
	s = strings.ToLower(s)
	for _, r := range rs {
		normName := strings.ToLower(r.Name)
		normCompany := strings.ToLower(r.Company)
		// other strings like zipcode, month, etc...

		if strings.Contains(normName, s) {
			keep = append(keep, r)
			continue
		}
		if strings.Contains(normCompany, s) {
			keep = append(keep, r)
			continue
		}
		// other contains conditions...
	}
	return keep, nil
}

func handleRealtorGet(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		realtorID := r.URL.Query().Get("id")
		name := r.URL.Query().Get("name")
		search := r.URL.Query().Get("search")
		pi := r.URL.Query().Get("property_info")
		var details bool
		var err error
		if pi != "" {
			details, err = strconv.ParseBool(pi)
			if err != nil {
				writeBadRequestError(w, err)
				return
			}
		}

		// no identifiers, return whole listing
		if realtorID == "" && name == "" {
			rs, err := searchRealtors(r.Context(), q, search)
			if err == pgx.ErrNoRows || len(rs) == 0 {
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

		// realtor_id specified, return the specific row
		if realtorID != "" {
			rid, err := strconv.Atoi(realtorID)
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			rs, err := q.GetRealtorProperties(r.Context(), int32(rid))
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

		// Name specified, return realtor entries under that name. NOTE: there
		// may be multiple realtors with the same name; callers can filter on
		// company if necessary.
		if name != "" {
			if details {
				writeRealtorPropertiesFull(r.Context(), l, q, w, name)
				return
			}
			writeRealtorProperties(r.Context(), l, q, w, name)
			return
		}

	}
}

func writeRealtorProperties(ctx context.Context, l *slog.Logger, q *dbgen.Queries, w http.ResponseWriter, name string) {
	rs, err := q.GetRealtorPropertiesByName(ctx, name)
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

func writeRealtorPropertiesFull(ctx context.Context, l *slog.Logger, q *dbgen.Queries, w http.ResponseWriter, name string) {
	rs, err := q.GetRealtorPropertiesFullByName(ctx, name)
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

func handleRealtorPost(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p dbgen.CreateRealtorParams
		err := decodeJSONBody(r, &p)
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
		// Ignore conflicts quietly since we expect workers to spam this route
		// with possible duplicates. However, note that the implementation of
		// the realtor table is such that the unique index includes the price,
		// so by spamming this route continuously, we'll build up a price
		// history for each realtor for a particular property. Additionally, the
		// server implementation should use an "on conflict do nothing" clause,
		// so errors shouldn't be thrown anyway.
		err = q.CreateRealtor(r.Context(), p)
		if err != nil {
			if !isPGError(err, pgErrorUniqueViolation) {
				writeInternalError(l, w, err)
				return
			}
			l.Debug("duplicate key for realtor", "realtor", p.Name, "property_id", p.PropertyID, "listing_id", p.ListingID)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DefaultJSONResponse{Message: "ok"})
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
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Message: "ok"})
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
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DefaultJSONResponse{Message: "ok"})
	}
}

// returns HTML that clients can use to display a D3 plot.
func handleRealtorPriceDistPlot(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "must supply realtor name"})
			return
		}
		// // rs, err := q.GetRealtorPropertiesByName(r.Context(), name)
		// if err == pgx.ErrNoRows {
		// 	writeEmptyResultError(w)
		// 	return
		// }
		// if err != nil {
		// 	writeInternalError(l, w, err)
		// 	return
		// }
		res := []struct {
			X []float64
			Y []float64
		}{
			{
				[]float64{0, 1, 2, 3, 4, 5},
				[]float64{11, 12, 13, 14, 15},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(res)
	}
}
