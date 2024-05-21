package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/brojonat/gredfin/server/dbgen"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func handleSearchQueryGet(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		search_id := r.URL.Query().Get("search_id")
		search_query := r.URL.Query().Get("search_query")

		// no identifier supplied, return listing
		if search_id == "" && search_query == "" {
			ss, err := q.ListSearches(r.Context())
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
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			json.NewEncoder(w).Encode(s)
			return
		}

		// search_query specified, return that search
		if search_query != "" {
			s, err := q.GetSearchByQuery(r.Context(), pgtype.Text{String: search_query})
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			json.NewEncoder(w).Encode(s)
			return
		}

	}
}

func handleSearchQueryPost(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p pgtype.Text
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
		err = q.CreateSearch(r.Context(), p)
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(defaultJSONResponse{Message: "ok"})
	}
}

func handleSearchQueryDelete(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		search_id := r.URL.Query().Get("search_id")
		search_query := r.URL.Query().Get("search_query")

		// no identifier supplied, error out
		if search_id == "" && search_query == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(defaultJSONResponse{Error: "must supply search_id or search_query"})
			return
		}

		// search_query specified, delete that search
		if search_query != "" {
			err := q.DeleteSearchByQuery(r.Context(), pgtype.Text{String: search_query})
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			json.NewEncoder(w).Encode(defaultJSONResponse{Message: "ok"})
			return
		}

		// search_id specified, delete that search
		sid, err := strconv.Atoi(search_id)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(defaultJSONResponse{Error: "bad value for search_id"})
			return
		}
		err = q.DeleteSearch(r.Context(), int32(sid))
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(defaultJSONResponse{Message: "ok"})
	}
}

func handleSearchQueryClaimNext(l *slog.Logger, p *pgxpool.Pool, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tx, err := p.Begin(r.Context())
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		q = q.WithTx(tx)
		s, err := q.GetNNextSearchScrapeForUpdate(
			r.Context(),
			dbgen.GetNNextSearchScrapeForUpdateParams{
				Limit: 1, Column2: []string{scrapeStatusGood}},
		)
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		q.UpdateSearchStatus(
			r.Context(),
			dbgen.UpdateSearchStatusParams{
				SearchID:         s.SearchID,
				LastScrapeStatus: pgtype.Text{String: scrapeStatusPending},
			},
		)
		tx.Commit(r.Context())
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(s)
	}
}

func handleSearchQuerySetStatus(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		search_id := r.URL.Query().Get("search_id")
		status := r.URL.Query().Get("status")

		if search_id == "" || status == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(defaultJSONResponse{Error: "must specify search_id and status"})
			return
		}

		sid, err := strconv.Atoi(search_id)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(defaultJSONResponse{Error: "bad value for search_id"})
			return
		}

		err = q.UpdateSearchStatus(
			r.Context(),
			dbgen.UpdateSearchStatusParams{
				SearchID:         int32(sid),
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
