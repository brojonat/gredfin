package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/brojonat/gredfin/server/db/dbgen"
	"github.com/brojonat/gredfin/server/db/jsonb"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func handleSearchGet(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		search_id := r.URL.Query().Get("search_id")
		search_query := r.URL.Query().Get("search_query")

		// no identifier supplied, return listing
		if search_id == "" && search_query == "" {
			ss, err := q.ListSearches(r.Context())
			if ss == nil || err == pgx.ErrNoRows {
				writeEmptyResultError(w)
				return
			}
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			json.NewEncoder(w).Encode(ss)
			return
		}

		// search_id specified, return that search
		if search_id != "" {
			sid, err := strconv.Atoi(search_id)
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			s, err := q.GetSearch(r.Context(), int32(sid))
			if err == pgx.ErrNoRows {
				writeEmptyResultError(w)
				return
			}
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			json.NewEncoder(w).Encode(s)
			return
		}

		// search_query specified, return that search
		if search_query != "" {
			s, err := q.GetSearchByQuery(r.Context(), pgtype.Text{String: search_query, Valid: true})
			if err == pgx.ErrNoRows {
				writeEmptyResultError(w)
				return
			}
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			json.NewEncoder(w).Encode(s)
			return
		}

	}
}

func handleSearchPost(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p pgtype.Text
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

		// create the search entry, ignore "already exists" error
		err = q.CreateSearch(r.Context(), p)
		if err != nil && !isPGError(err, pgErrorUniqueViolation) {
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DefaultJSONResponse{Message: "ok"})
	}
}

func handleSearchDelete(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		search_id := r.URL.Query().Get("search_id")
		search_query := r.URL.Query().Get("search_query")

		// no identifier supplied, error out
		if search_id == "" && search_query == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "must supply search_id or search_query"})
			return
		}

		// search_query specified, delete that search
		if search_query != "" {
			err := q.DeleteSearchByQuery(r.Context(), pgtype.Text{String: search_query, Valid: true})
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			json.NewEncoder(w).Encode(DefaultJSONResponse{Message: "ok"})
			return
		}

		// search_id specified, delete that search
		sid, err := strconv.Atoi(search_id)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "bad value for search_id"})
			return
		}
		err = q.DeleteSearch(r.Context(), int32(sid))
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DefaultJSONResponse{Message: "ok"})
	}
}

func handleSearchClaimNext(l *slog.Logger, p *pgxpool.Pool, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tx, err := p.Begin(r.Context())
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		defer tx.Commit(r.Context())
		q = q.WithTx(tx)

		s, err := q.GetNNextSearchScrapeForUpdate(
			r.Context(),
			dbgen.GetNNextSearchScrapeForUpdateParams{
				Count: 1, Statuses: []string{ScrapeStatusGood}},
		)
		if err == pgx.ErrNoRows {
			writeEmptyResultError(w)
			return
		}
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		// FIXME: verify that this doesn't overwrite the metadata to {}
		err = q.UpdateSearchStatus(
			r.Context(),
			dbgen.UpdateSearchStatusParams{
				SearchID:         s.SearchID,
				LastScrapeStatus: ScrapeStatusPending,
			},
		)
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(s)
	}
}

func handleSearchSetStatus(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		search_id := r.URL.Query().Get("search_id")
		status := r.URL.Query().Get("status")
		success_count := r.URL.Query().Get("success_count")
		error_count := r.URL.Query().Get("error_count")

		if search_id == "" || status == "" || !isValidStatus(status) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "must specify search_id, valid status, and property_count"})
			return
		}

		sid, err := strconv.Atoi(search_id)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "bad value for search_id"})
			return
		}

		sc, err := strconv.Atoi(success_count)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "bad value for success_count"})
			return
		}
		ec, err := strconv.Atoi(error_count)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "bad value for error_count"})
			return
		}

		err = q.UpdateSearchStatus(
			r.Context(),
			dbgen.UpdateSearchStatusParams{
				SearchID:           int32(sid),
				LastScrapeStatus:   status,
				LastScrapeMetadata: &jsonb.SearchScrapeMetadata{SuccessCount: sc, ErrorCount: ec},
			},
		)
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DefaultJSONResponse{Message: "ok"})
	}
}

func handleGetRecentSearchScrapeStats(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dp := r.URL.Query().Get("duration")
		if dp == "" {
			dp = "5m"
		}
		dur, err := time.ParseDuration(dp)
		if err != nil {
			writeBadRequestError(w, fmt.Errorf("could not parse duration %s", dp))
			return
		}
		ts := time.Now().Add(-dur)
		res, err := q.GetRecentSearchScrapeStats(r.Context(), pgtype.Timestamp{Time: ts, Valid: true})
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(res)
	}
}
