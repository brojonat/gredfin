package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/brojonat/gredfin/server/db/dbgen"
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
			if err == pgx.ErrNoRows {
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
		err = q.CreateSearch(r.Context(), p)
		if err != nil {
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

		if search_id == "" || status == "" || !isValidStatus(status) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "must specify search_id and valid status"})
			return
		}

		sid, err := strconv.Atoi(search_id)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "bad value for search_id"})
			return
		}

		err = q.UpdateSearchStatus(
			r.Context(),
			dbgen.UpdateSearchStatusParams{
				SearchID:         int32(sid),
				LastScrapeStatus: status,
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
