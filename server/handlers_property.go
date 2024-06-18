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
	"strings"
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
	AVMHash         string `json:"avm_hash"`
}

func handlePropertyGet(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
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
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "missing listing_id"})
			return
		}

		// no listingID, return a listing of properties
		if listingID == "" {
			pid, err := strconv.Atoi(propertyID)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "bad value for property_id"})
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
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "bad value for property_id"})
			return
		}
		lid, err := strconv.Atoi(listingID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "bad value for listing_id"})
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

func handlePropertyPost(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p dbgen.CreatePropertyParams
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
		err = q.CreateProperty(r.Context(), p)
		if err != nil {
			if !isPGError(err, pgErrorUniqueViolation) {
				writeInternalError(l, w, err)
				return
			}
			l.Debug("duplicate key for property", "property_id", p.PropertyID, "listing_id", p.ListingID, "url", p.URL)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DefaultJSONResponse{Message: "ok"})
	}
}

// Gets the current property with the supplied property_id and listing_id, then
// for each field that is specified in the input, updates the current with the
// specified data, then writes the resulting object to the model.
func handlePropertyUpdate(l *slog.Logger, p *pgxpool.Pool, q *dbgen.Queries) http.HandlerFunc {
	type propertyUpdate struct {
		dbgen.CreatePropertyParams
		PropertyScrapeMetadata
		Status       string `json:"status"`
		LastScrapeTs string `json:"last_scrape_ts"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var updateData propertyUpdate
		err := decodeJSONBody(r, &updateData)
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

		tx, err := p.Begin(r.Context())
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		defer tx.Commit(r.Context())
		q = q.WithTx(tx)

		current, err := q.GetProperty(r.Context(), dbgen.GetPropertyParams{
			PropertyID: updateData.PropertyID,
			ListingID:  updateData.ListingID,
		})
		if err != nil {
			if err == pgx.ErrNoRows {
				w.WriteHeader(http.StatusNotFound)
				msg := fmt.Sprintf("property does not exist (pid: %d, lid: %d)", updateData.PropertyID, updateData.ListingID)
				json.NewEncoder(w).Encode(DefaultJSONResponse{Error: msg})
				return
			}
			writeInternalError(l, w, err)
			return
		}
		pd := dbgen.PutPropertyParams{
			PropertyID:          updateData.PropertyID,
			ListingID:           updateData.ListingID,
			URL:                 current.URL,
			Zipcode:             current.Zipcode,
			City:                current.City,
			State:               current.State,
			ListPrice:           current.ListPrice,
			LastScrapeTs:        pgtype.Timestamp{Time: time.Now(), Valid: true},
			LastScrapeStatus:    current.LastScrapeStatus,
			LastScrapeChecksums: current.LastScrapeChecksums,
		}
		if updateData.URL.String != "" {
			pd.URL = updateData.URL
		}
		if updateData.Status != "" {
			pd.LastScrapeStatus = pgtype.Text{String: updateData.Status, Valid: true}
		}
		if updateData.LastScrapeTs != "" {
			ts, err := time.Parse(time.RFC3339, updateData.LastScrapeTs)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "bad timestamp"})
				return
			}
			pd.LastScrapeTs = pgtype.Timestamp{Time: ts, Valid: true}
		}
		if updateData.InitialInfoHash != "" {
			pd.LastScrapeChecksums.InitialInfoHash = updateData.InitialInfoHash
		}
		if updateData.MLSHash != "" {
			pd.LastScrapeChecksums.MLSHash = updateData.MLSHash
		}
		if updateData.AVMHash != "" {
			pd.LastScrapeChecksums.AVMHash = updateData.AVMHash
		}
		err = q.PutProperty(r.Context(), pd)
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DefaultJSONResponse{Message: "ok"})
	}
}

func handlePropertyDelete(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propertyID := r.URL.Query().Get("property_id")
		listingID := r.URL.Query().Get("listing_id")

		if propertyID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "must supply property_id"})
			return
		}

		// no listingID, delete all property entries under the ID
		if listingID == "" {
			pid, err := strconv.Atoi(propertyID)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "bad value for property_id"})
				return
			}
			err = q.DeletePropertyListingsByID(r.Context(), int32(pid))
			if err != nil {
				writeInternalError(l, w, err)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Message: "ok"})
			return
		}

		// delete property listing
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
		json.NewEncoder(w).Encode(DefaultJSONResponse{Message: "ok"})
	}
}

// claims the next property to be scraped and sets the status to pending
func handlePropertyClaimNext(l *slog.Logger, p *pgxpool.Pool, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tx, err := p.Begin(r.Context())
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		defer tx.Commit(r.Context())
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
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(prop)
	}
}

func handleGetPresignedPutURL(l *slog.Logger, s3c *s3.Client, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propertyID := r.URL.Query().Get("property_id")
		listingID := r.URL.Query().Get("listing_id")
		basename := r.URL.Query().Get("basename")
		if propertyID == "" || listingID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "must supply property_id and listing_id"})
			return
		}
		if !strings.HasSuffix(basename, ".json") || len(basename) <= 5 {
			w.WriteHeader(http.StatusBadRequest)
			msg := fmt.Sprintf("invalid basename %s; must be valid filename with .json extension", basename)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: msg})
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
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "invalid value for property_id"})
			return
		}
		lid, err := strconv.Atoi(listingID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "invalid value for listing_id"})
			return
		}
		key, err := getPropertyKey(r.Context(), q, int32(pid), int32(lid), basename)
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
		json.NewEncoder(w).Encode(DefaultJSONResponse{Message: presignedPutRequest.URL})
	}
}

func getPropertyBucket() (string, error) {
	b := os.Getenv("S3_PROPERTY_BUCKET")
	if b == "" {
		return "", fmt.Errorf("s3 property bucket not set")
	}
	return b, nil
}

func getPropertyKey(ctx context.Context, q *dbgen.Queries, pid, lid int32, basename string) (string, error) {
	p, err := q.GetProperty(ctx, dbgen.GetPropertyParams{PropertyID: pid, ListingID: lid})
	if err != nil {
		return "", err
	}
	addr := strings.TrimPrefix(p.URL.String, "https://www.redfin.com/")
	if addr == p.URL.String {
		return "", fmt.Errorf("unable to parse url to address %s", p.URL.String)
	}
	return fmt.Sprintf("property/%s/%d_%d_%d_%s", addr, pid, lid, time.Now().Unix(), basename), nil
}
