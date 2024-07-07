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
	"github.com/jackc/pgx/v5/pgxpool"
)

func handlePropertyEventsGet(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(events)
	}
}

func handlePropertyEventsPost(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var ps []dbgen.CreatePropertyEventParams
		err := decodeJSONBody(r, &ps)
		if err != nil {
			var mr *MalformedRequest
			var pr *time.ParseError
			if errors.As(err, &mr) {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(DefaultJSONResponse{Error: fmt.Sprintf("bad request payload: %s", err.Error())})
			} else if errors.As(err, &pr) {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(DefaultJSONResponse{Error: fmt.Sprintf("bad timestamp format: %s", err.Error())})
			} else {
				writeInternalError(l, w, err)
			}
			return
		}

		// validate each event, if any are invalid, return early with 400
		for _, p := range ps {
			if p.PropertyID == 0 || p.ListingID == 0 ||
				!p.EventDescription.Valid ||
				!p.EventTS.Valid {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "bad request payload: must set property_id, listing_id, event_description, and event_ts"})
				return
			}
		}
		// create and return status
		count, err := q.CreatePropertyEvent(r.Context(), ps)
		if err != nil {
			// Since this is a bulk creation, we may get some successful and some unsuccessful items,
			// so write a 400 response but include the count of successful events.
			if isPGError(err, pgErrorForeignKeyViolation) {
				writeBadRequestError(w, fmt.Errorf("event must map to an existing property (created %d / %d)", count, len(ps)))
				return
			}
			if isPGError(err, pgErrorNotNullViolation) {
				writeBadRequestError(w, fmt.Errorf("missing required field (created %d / %d)", count, len(ps)))
				return
			}
			// this case is expected for some clients and has its own status code
			if isPGError(err, pgErrorUniqueViolation) {
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(DefaultJSONResponse{Message: fmt.Sprintf("%d / %d", count, len(ps))})
				return
			}
			// default unhandled error
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DefaultJSONResponse{Message: fmt.Sprintf("%d / %d", count, len(ps))})
	}
}

func handlePropertyEventsPut(l *slog.Logger, p *pgxpool.Pool, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var ps []dbgen.CreatePropertyEventParams
		err := decodeJSONBody(r, &ps)
		if err != nil {
			var mr *MalformedRequest
			var pr *time.ParseError
			if errors.As(err, &mr) {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(DefaultJSONResponse{Error: fmt.Sprintf("bad request payload: %s", err.Error())})
			} else if errors.As(err, &pr) {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(DefaultJSONResponse{Error: fmt.Sprintf("bad timestamp format: %s", err.Error())})
			} else {
				writeInternalError(l, w, err)
			}
			return
		}

		if len(ps) == 0 {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Message: "ok"})
			return
		}

		// validate each event, if any are invalid, return early with 400
		pids := map[int32]struct{}{}
		for _, p := range ps {
			if p.PropertyID == 0 || p.ListingID == 0 ||
				!p.EventDescription.Valid ||
				!p.EventTS.Valid {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "bad request payload: must set property_id, listing_id, event_description, and event_ts"})
				return
			}
			pids[p.PropertyID] = struct{}{}
		}

		// the PUT route will delete first, and then bulk create
		tx, err := p.Begin(r.Context())
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		defer tx.Commit(r.Context())
		q = q.WithTx(tx)
		for pid, _ := range pids {
			err := q.DeletePropertyEventsByProperty(r.Context(), dbgen.DeletePropertyEventsByPropertyParams{PropertyID: pid})
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
		}

		// create and return status
		count, err := q.CreatePropertyEvent(r.Context(), ps)
		if err != nil {
			if isPGError(err, pgErrorForeignKeyViolation) {
				writeBadRequestError(w, fmt.Errorf("event must map to an existing property (created %d / %d)", count, len(ps)))
				return
			}
			if isPGError(err, pgErrorNotNullViolation) {
				writeBadRequestError(w, fmt.Errorf("missing required field (created %d / %d)", count, len(ps)))
				return
			}
			// on the PUT route, this should never really happen since we just cleared the events
			if isPGError(err, pgErrorUniqueViolation) {
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(DefaultJSONResponse{Message: fmt.Sprintf("%d / %d", count, len(ps))})
				return
			}
			// default unhandled error
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DefaultJSONResponse{Message: fmt.Sprintf("%d / %d", count, len(ps))})
	}
}

func handlePropertyEventsDelete(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		event_ids := r.URL.Query()["event_id"]
		if len(event_ids) == 0 {
			writeBadRequestError(w, fmt.Errorf("must supply at least one event_id"))
			return
		}

		ids := []int32{}
		for _, eid := range event_ids {
			id, err := strconv.Atoi(eid)
			if err != nil {
				writeBadRequestError(w, fmt.Errorf("bad event_id: %s", eid))
				return
			}
			ids = append(ids, int32(id))
		}
		err := q.DeletePropertyEvents(r.Context(), ids)
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DefaultJSONResponse{Message: "ok"})
	}
}
