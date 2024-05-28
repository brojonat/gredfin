package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/brojonat/gredfin/server/dbgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PropertyScrapeMetadata struct {
	InitialInfoHash string `json:"initial_info_hash"`
	MLSHash         string `json:"mls_hash"`
	AVMHash         string `json:"AVMHash"`
}

func handlePropertyQueryGet(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propertyID := r.URL.Query().Get("property_id")
		listingID := r.URL.Query().Get("listing_id")

		// no identifier, list properties
		if propertyID == "" && listingID == "" {
			props, err := q.ListProperties(r.Context())
			if err == pgx.ErrNoRows {
				writeEmptyResultError(w)
				return
			}
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
			pid, err := strconv.Atoi(propertyID)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(defaultJSONResponse{Error: "bad value for property_id"})
				return
			}
			props, err := q.GetPropertiesByID(r.Context(), int32(pid))
			if err == pgx.ErrNoRows {
				writeEmptyResultError(w)
				return
			}
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(props)
			return
		}

		// return single entry
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
		prop, err := q.GetProperty(r.Context(), dbgen.GetPropertyParams{PropertyID: int32(pid), ListingID: int32(lid)})
		if err == pgx.ErrNoRows {
			writeEmptyResultError(w)
			return
		}
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
			pid, err := strconv.Atoi(propertyID)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(defaultJSONResponse{Error: "bad value for property_id"})
				return
			}
			err = q.DeletePropertyListingsByID(r.Context(), int32(pid))
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(defaultJSONResponse{Message: "ok"})
			return
		}

		// delete property listing
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
		err = q.DeletePropertyListing(
			r.Context(),
			dbgen.DeletePropertyListingParams{
				PropertyID: int32(pid),
				ListingID:  int32(lid),
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
				Limit: 1, Column2: []string{ScrapeStatusGood}},
		)
		if err == pgx.ErrNoRows {
			writeEmptyResultError(w)
			return
		}
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		q.UpdatePropertyStatus(
			r.Context(),
			dbgen.UpdatePropertyStatusParams{
				PropertyID:       prop.PropertyID,
				ListingID:        prop.ListingID,
				LastScrapeStatus: pgtype.Text{String: ScrapeStatusPending, Valid: true},
			},
		)
		tx.Commit(r.Context())
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(prop)
	}
}

func handlePropertySetStatus(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propertyID := r.URL.Query().Get("property_id")
		listingID := r.URL.Query().Get("listing_id")
		status := r.URL.Query().Get("status")

		if propertyID == "" || listingID == "" || status == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(defaultJSONResponse{Error: "must specify property_id, listing_id, and status"})
			return
		}

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

		err = q.UpdatePropertyStatus(
			r.Context(),
			dbgen.UpdatePropertyStatusParams{
				PropertyID:       int32(pid),
				ListingID:        int32(lid),
				LastScrapeStatus: pgtype.Text{String: status, Valid: true},
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

func handleGetPresignedPutURL(l *slog.Logger, s3c *s3.Client, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propertyID := r.URL.Query().Get("property_id")
		listingID := r.URL.Query().Get("listing_id")
		if propertyID == "" || listingID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(defaultJSONResponse{Error: "must supply property_id and listing_id"})
			return
		}
		bucket, err := getPropertyBucket()
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		pid, err := strconv.Atoi(propertyID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(defaultJSONResponse{Error: "invalid value for property_id"})
			return
		}
		lid, err := strconv.Atoi(listingID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(defaultJSONResponse{Error: "invalid value for listing_id"})
			return
		}
		key, err := getPropertyKey(r.Context(), int32(pid), int32(lid), q)
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		ps := s3.NewPresignClient(s3c)
		presignedPutRequest, err := ps.PresignPutObject(
			r.Context(),
			&s3.PutObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
			},
			func(opts *s3.PresignOptions) {
				opts.Expires = time.Duration(600 * int64(time.Second))
			})
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(defaultJSONResponse{Message: presignedPutRequest.URL})
	}
}

func getPropertyBucket() (string, error) {
	b := os.Getenv("S3_PROPERTY_BUCKET")
	if b == "" {
		return "", fmt.Errorf("s3 property bucket not set")
	}
	return b, nil
}

func getPropertyKey(ctx context.Context, pid, lid int32, q *dbgen.Queries) (string, error) {
	p, err := q.GetProperty(ctx, dbgen.GetPropertyParams{PropertyID: pid, ListingID: lid})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%d %d %d", p.URL.String, pid, lid, time.Now().Unix()), nil
}
