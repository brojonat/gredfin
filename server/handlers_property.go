package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
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

func handlePropertyGet(l *slog.Logger, q *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propertyID := r.URL.Query().Get("property_id")
		listingID := r.URL.Query().Get("listing_id")

		// no identifier, list properties
		if propertyID == "" && listingID == "" {
			props, err := q.ListPropertiesPrices(r.Context())
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
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "missing property_id"})
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
			props, err := q.GetPropertiesWithPrice(r.Context(), dbgen.GetPropertiesWithPriceParams{PropertyID: int32(pid)})
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
		prop, err := q.GetPropertyWithPrice(r.Context(), dbgen.GetPropertyWithPriceParams{PropertyID: int32(pid), ListingID: int32(lid)})
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
				json.NewEncoder(w).Encode(DefaultJSONResponse{Error: fmt.Sprintf("bad request payload: %s", err.Error())})
			} else {
				writeInternalError(l, w, err)
			}
			return
		}

		// check if this url is blocklisted, return early with a 204 if so
		bps, err := q.ListBlocklistedProperties(r.Context(), []string{p.URL.String})
		if err == nil && len(bps) > 0 {
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Message: "ok"})
			return
		}
		// this should return ErrNoRows in normal circumstances; exit early with 500 if not
		if err != nil && err != pgx.ErrNoRows {
			writeInternalError(l, w, err)
			return
		}

		// create the property, ignore "already exists" error
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
	return func(w http.ResponseWriter, r *http.Request) {
		var updateData dbgen.PutPropertyParams
		err := decodeJSONBody(r, &updateData)
		if err != nil {
			var mr *MalformedRequest
			if errors.As(err, &mr) {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(DefaultJSONResponse{Error: fmt.Sprintf("bad request payload: %s", err.Error())})
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

		// NOTE: this MUST use the "basic" property table and not the
		// property_price view, since the property_price view surfaces only
		// properties that have at least one property_event with a price.
		current, err := q.GetPropertyBasic(r.Context(), dbgen.GetPropertyBasicParams{
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
		// Default the update data to the existing data, then for each field, check
		// if the supplied value is not equal to the zero value. This makes it hard
		// to set the actual value to the zero value on this route, but I don't care.
		pd := dbgen.PutPropertyParams{
			PropertyID:         current.PropertyID,
			ListingID:          current.ListingID,
			URL:                current.URL,
			Zipcode:            current.Zipcode,
			City:               current.City,
			State:              current.State,
			Location:           current.Location,
			LastScrapeTS:       pgtype.Timestamp{Time: time.Now(), Valid: true},
			LastScrapeStatus:   current.LastScrapeStatus,
			LastScrapeMetadata: current.LastScrapeMetadata,
		}
		if updateData.URL.String != "" {
			pd.URL = updateData.URL
		}
		if updateData.Zipcode.String != "" {
			pd.Zipcode = updateData.Zipcode
		}
		if updateData.City.String != "" {
			pd.City = updateData.City
		}
		if updateData.State.String != "" {
			pd.State = updateData.State
		}
		if updateData.Location != nil {
			pd.Location = updateData.Location
		}
		if !updateData.LastScrapeTS.Time.IsZero() {
			pd.LastScrapeTS = updateData.LastScrapeTS
		}
		if updateData.LastScrapeStatus.String != "" {
			pd.LastScrapeStatus = updateData.LastScrapeStatus
		}
		if updateData.LastScrapeMetadata.InitialInfoHash != "" {
			pd.LastScrapeMetadata.InitialInfoHash = updateData.LastScrapeMetadata.InitialInfoHash
		}
		if updateData.LastScrapeMetadata.MLSHash != "" {
			pd.LastScrapeMetadata.MLSHash = updateData.LastScrapeMetadata.MLSHash
		}
		if updateData.LastScrapeMetadata.AVMHash != "" {
			pd.LastScrapeMetadata.AVMHash = updateData.LastScrapeMetadata.AVMHash
		}
		if updateData.LastScrapeMetadata.ImageURLs != nil {
			pd.LastScrapeMetadata.ImageURLs = updateData.LastScrapeMetadata.ImageURLs
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
