package worker

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/brojonat/gredfin/redfin"
	"github.com/brojonat/gredfin/server"
	"github.com/brojonat/gredfin/server/dbgen"
	"github.com/brojonat/gredfin/server/dbgen/jsonb"
	"github.com/jackc/pgx/v5/pgtype"
)

func logPropertyError(l *slog.Logger, msg string, err error, p *dbgen.Property) {
	if p == nil {
		l.Error(msg, "error", err.Error())
		return
	}
	l.Error(msg, "error", err.Error(), "property_id", p.PropertyID, "listing_id", p.ListingID, "url", p.URL)
}

func hashBytes(b []byte) string {
	hash := md5.Sum(b)
	return hex.EncodeToString(hash[:])
}

// Default implementation of a Property scrape worker.
func MakePropertyWorkerFunc(
	end string,
	authToken string,
	grc redfin.Client,
) func(context.Context, *slog.Logger) {
	f := func(ctx context.Context, l *slog.Logger) {
		h := server.GetDefaultServerHeaders(authToken)
		p, err := claimProperty(end, h)
		if err != nil {
			logPropertyError(l, "error claiming property from server", err, p)
			return
		}
		pid := p.PropertyID
		lid := p.ListingID
		url := p.URL.String
		l.Info("running scrape worker", "property_id", pid, "listing_id", lid, "url", url)

		// The following is a bit verbose with logging statements so here's a
		// high level overview. Make Redfin client calls for the InitialInfo,
		// BelowTheFold (MLSInfo), and AVMDetails data. For each call, if an
		// error is returned, then log it and mark the scrape bad. Marking bad
		// can also fail (e.g., due to network reasons), so also log an error in
		// that case too before returning. If the client call doesn't return an
		// error, then parse the response. If that returns an error, do the same
		// error handling as above. Finally, check the error code/response
		// message in the Redfin response itself and if that doesn't indicate
		// success, then do the same error handling as above.

		var iiRes redfin.RedfinResponse
		var mlsRes redfin.RedfinResponse
		var avmRes redfin.RedfinResponse

		// pull InitialInfo bytes
		params := map[string]string{}
		iib, err := grc.InitialInfo(p.URL.String, params)
		if err != nil {
			logPropertyError(l, "error getting InitialInfo, marking scrape bad", err, p)
			if err = markPropertyScrapeBad(end, h, pid, lid); err != nil {
				logPropertyError(l, "error marking scrape bad", err, p)
			}
			return
		}
		if err = json.Unmarshal(iib, &iiRes); err != nil {
			logPropertyError(l, "error serializing InitialInfo response, marking scrape bad", err, p)
			if err = markPropertyScrapeBad(end, h, pid, lid); err != nil {
				logPropertyError(l, "error marking scrape bad", err, p)
			}
			return
		}
		if err = checkRedfinResponse(iiRes); err != nil {
			logPropertyError(l, "error with InitialInfo response, marking scrape bad", err, p)
			if err = markPropertyScrapeBad(end, h, pid, lid); err != nil {
				logPropertyError(l, "error marking scrape bad", err, p)
			}
			return
		}

		// pull MLS (below-the-fold) bytes
		property_id := strconv.Itoa(int(p.PropertyID))
		listing_id := strconv.Itoa(int(p.ListingID))
		mlsb, err := grc.BelowTheFold(property_id, params)
		if err != nil {
			logPropertyError(l, "error getting BelowTheFold (MLS) data, marking scrape bad", err, p)
			if err = markPropertyScrapeBad(end, h, pid, lid); err != nil {
				logPropertyError(l, "error marking scrape bad", err, p)
			}
			return
		}
		if err = json.Unmarshal(mlsb, &mlsRes); err != nil {
			logPropertyError(l, "error serializing mls response, marking scrape bad", err, p)
			if err = markPropertyScrapeBad(end, h, pid, lid); err != nil {
				logPropertyError(l, "error marking scrape bad", err, p)
			}
			return
		}
		if err = checkRedfinResponse(mlsRes); err != nil {
			l.Info(string(mlsb))
			logPropertyError(l, "error with mls (below the fold) response, marking scrape bad", err, p)
			if err = markPropertyScrapeBad(end, h, pid, lid); err != nil {
				logPropertyError(l, "error marking scrape bad", err, p)
			}
			return
		}

		// pull AVM bytes
		avmb, err := grc.AVMDetails(property_id, listing_id, params)
		if err != nil {
			logPropertyError(l, "error getting avm info, marking scrape bad", err, p)
			if err = markPropertyScrapeBad(end, h, pid, lid); err != nil {
				logPropertyError(l, "error marking scrape bad", err, p)
			}
			return
		}
		if err = json.Unmarshal(avmb, &avmRes); err != nil {
			logPropertyError(l, "error serializing avm response, marking scrape bad", err, p)
			if err = markPropertyScrapeBad(end, h, pid, lid); err != nil {
				logPropertyError(l, "error marking scrape bad", err, p)
			}
			return
		}
		if err = checkRedfinResponse(avmRes); err != nil {
			logPropertyError(l, "error with avm response, marking scrape bad", err, p)
			if err = markPropertyScrapeBad(end, h, pid, lid); err != nil {
				logPropertyError(l, "error marking scrape bad", err, p)
			}
			return
		}

		// At this point, we've successfully fetched all the bytes from Redfin
		// pertaining to this property. Now we just have to do something with
		// them. Pass them to handlePropertyBytes which will process them (extract
		// useful data, make some queries to the server, etc).
		err = handlePropertyBytes(end, h, l, p, iiRes.Payload, mlsRes.Payload, avmRes.Payload)
		if err != nil {
			logPropertyError(l, "error handling property data, marking scrape bad", err, p)
			if err = markPropertyScrapeBad(end, h, pid, lid); err != nil {
				logPropertyError(l, "error marking scrape bad", err, p)
			}
			return
		}

		// Mark the scrape status as good on the server
		payload := dbgen.PutPropertyParams{
			PropertyID:       p.PropertyID,
			ListingID:        p.ListingID,
			LastScrapeStatus: pgtype.Text{String: server.ScrapeStatusGood, Valid: true},
			LastScrapeChecksums: jsonb.PropertyScrapeMetadata{
				InitialInfoHash: hashBytes(iiRes.Payload),
				MLSHash:         hashBytes(mlsRes.Payload),
				AVMHash:         hashBytes(avmRes.Payload),
			},
		}
		b, err := json.Marshal(payload)
		if err != nil {
			logPropertyError(l, "error serializing property update payload", err, p)
			if err = markPropertyScrapeBad(end, h, pid, lid); err != nil {
				logPropertyError(l, "error marking scrape bad", err, p)
			}
			return
		}
		if err = updateProperty(end, h, b); err != nil {
			logPropertyError(l, "error updating property scrape metadata", err, p)
			if err = markPropertyScrapeBad(end, h, pid, lid); err != nil {
				logPropertyError(l, "error marking scrape bad", err, p)
			}
			return
		}
	}
	return f
}

func checkRedfinResponse(r redfin.RedfinResponse) error {
	if r.ResultCode != 0 || r.ErrorMessage != "Success" {
		return fmt.Errorf("bad redfin response: (code: %d, message: %s)", r.ResultCode, r.ErrorMessage)
	}
	return nil
}

// Parse property scrape bytes and upload relevant data.
func handlePropertyBytes(end string, h http.Header, l *slog.Logger, p *dbgen.Property, iib, mlsb, avmb []byte) error {

	// first parse the bytes into an empty interface for jmespath search
	var jmesMLS interface{}
	err := json.Unmarshal(mlsb, &jmesMLS)
	if err != nil {
		return fmt.Errorf("error parsing MLS bytes")
	}

	// Helper closure to parse and upload basic property data. This sets the
	// data in the property table.
	parseUploadProperty := func() error {

		// parse zipcode
		zipcode, err := jmesParseMLSParams("zipcode", jmesMLS)
		if err != nil {
			return fmt.Errorf("error searching for zipcode: %w", err)
		}
		if zipcode == nil {
			return fmt.Errorf("null result extracting zipcode")
		}

		// parse city
		city, err := jmesParseMLSParams("city", jmesMLS)
		if err != nil {
			return fmt.Errorf("error searching for city %w", err)
		}
		if city == nil {
			return fmt.Errorf("null result extracting city")
		}

		// parse state
		state, err := jmesParseMLSParams("state", jmesMLS)
		if err != nil {
			return fmt.Errorf("error searching for state %w", err)
		}
		if state == nil {
			return fmt.Errorf("null result extracting state")
		}

		// parse listing price
		lp, err := jmesParseMLSParams("price", jmesMLS)
		if err != nil {
			return fmt.Errorf("error extracting list price: %w", err)
		}
		if lp == nil {
			return fmt.Errorf("null result extracting list price")
		}
		// upload
		np := dbgen.PutPropertyParams{
			PropertyID: p.PropertyID,
			ListingID:  p.ListingID,
			URL:        p.URL,
			Zipcode:    pgtype.Text{String: zipcode.(string), Valid: true},
			City:       pgtype.Text{String: city.(string), Valid: true},
			State:      pgtype.Text{String: state.(string), Valid: true},
		}
		b, err := json.Marshal(np)
		if err != nil {
			return fmt.Errorf("error serializing property (property_id: %d, listing_id: %d): %w", p.PropertyID, p.ListingID, err)
		}
		if err = updateProperty(end, h, b); err != nil {
			return fmt.Errorf("error uploading property: %w", err)
		}
		return nil
	}

	// helper closure to parse and upload property history events. This sets the
	// data in the property_events table AND the property_events_property_through table.
	parseUploadPropertyEvents := func() error {
		hevents, err := jmesParseMLSParams("events", jmesMLS)
		if err != nil {
			return err
		}
		events := []dbgen.CreatePropertyEventParams{}
		for _, he := range hevents.([]historyEvent) {
			events = append(events, dbgen.CreatePropertyEventParams{
				PropertyID:       p.PropertyID,
				ListingID:        p.ListingID,
				Price:            he.Price,
				EventDescription: pgtype.Text{String: he.EventDescription, Valid: true},
				Source:           pgtype.Text{String: he.Source, Valid: true},
				SourceID:         pgtype.Text{String: he.SourceID, Valid: true},
				EventTS:          pgtype.Timestamp{Time: he.EventTS, Valid: true},
			})
		}

		b, err := json.Marshal(events)
		if err != nil {
			return fmt.Errorf("error serializing property history events (property_id: %d, listing_id: %d): %w", p.PropertyID, p.ListingID, err)
		}
		if err = createPropertyEvents(end, h, b); err != nil {
			return fmt.Errorf("error uploading property history events: %w", err)
		}
		return nil
	}

	// Helper closure to parse and upload realtor data. This sets the data in
	// the realtor-property through table.
	parseUploadRealtor := func() error {
		// parse and upload the realtor data to the server
		name, company, err := parseRealtorInfo(mlsb)
		if err != nil {
			return fmt.Errorf("error extracting realtor: %w", err)
		}
		r := dbgen.CreateRealtorParams{
			PropertyID: p.PropertyID,
			ListingID:  p.ListingID,
			Name:       name,
			Company:    company,
		}

		b, err := json.Marshal(r)
		if err != nil {
			return fmt.Errorf("could not serialize realtor for create request: %w", err)
		}
		if err = createRealtor(end, h, b); err != nil {
			return fmt.Errorf("error uploading realtor: %w", err)
		}
		return nil

	}

	// Helper closure to upload bytes to S3 if the hash is different from the
	// last scrape. This sets the data in the object storage.
	maybeS3Upload := func(b []byte, hash string, basename string) error {
		if hashBytes(b) == hash || true {
			l.Debug("skipping scrape upload, bytes unchanged", "property_id", p.PropertyID, "listing_id", p.ListingID, "basename", basename)
			return nil
		}
		url, err := getPresignedPutURL(end, h, p, basename)
		if err != nil {
			return err
		}
		req, err := http.NewRequest(
			http.MethodPut,
			url,
			bytes.NewReader(b),
		)
		if err != nil {
			return err
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf(res.Status)
		}
		return nil
	}

	// parse and upload the property data to the server
	if err := parseUploadProperty(); err != nil {
		return fmt.Errorf("error uploading property: %w", err)
	}

	// parse and upload the property data to the server
	if err := parseUploadPropertyEvents(); err != nil {
		return fmt.Errorf("error uploading property history events: %w", err)
	}

	// parse and upload the realtor data for this property to the server
	if err := parseUploadRealtor(); err != nil {
		return fmt.Errorf("error uploading realtor: %w", err)
	}

	// now (maybe) do S3 uploads to the cloud object store
	if err := maybeS3Upload(iib, p.LastScrapeChecksums.InitialInfoHash, "initial_info.json"); err != nil {
		return fmt.Errorf("error uploading InitialInfo bytes: %w", err)
	}
	if err := maybeS3Upload(mlsb, p.LastScrapeChecksums.MLSHash, "mls_info.json"); err != nil {
		return fmt.Errorf("error uploading MLSInfo bytes: %w", err)
	}
	if err := maybeS3Upload(avmb, p.LastScrapeChecksums.AVMHash, "avm_info.json"); err != nil {
		return fmt.Errorf("error uploading AVMInfo bytes: %w", err)
	}
	return nil
}

func getPresignedPutURL(end string, h http.Header, p *dbgen.Property, basename string) (string, error) {
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/property-query/get-presigned-put-url", end),
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("error making presigned url request: %w", err)
	}
	req.Header = h
	q := req.URL.Query()
	q.Add("property_id", strconv.Itoa(int(p.PropertyID)))
	q.Add("listing_id", strconv.Itoa(int(p.ListingID)))
	q.Add("basename", basename)
	req.URL.RawQuery = q.Encode()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error doing presigned url request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", fmt.Errorf("error with presigned url response: %s", res.Status)
	}

	var data server.DefaultJSONResponse
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("error reading presigned url response: %w", err)
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return "", fmt.Errorf("error parsing presigned url response: %w", err)
	}
	return data.Message, nil
}

func markPropertyScrapeBad(endpoint string, h http.Header, pid, lid int32) error {
	payload := dbgen.UpdatePropertyStatusParams{
		PropertyID:       pid,
		ListingID:        lid,
		LastScrapeStatus: pgtype.Text{String: server.ScrapeStatusBad, Valid: true},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return updateProperty(endpoint, h, b)
}

func claimProperty(endpoint string, headers http.Header) (*dbgen.Property, error) {
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/property-query/claim-next", endpoint),
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header = headers
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(res.Status)
	}

	var p dbgen.Property
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func createProperty(endpoint string, h http.Header, b []byte) error {
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/property", endpoint),
		bytes.NewReader(b),
	)
	if err != nil {
		return fmt.Errorf("error doing create Property request: %w", err)
	}
	req.Header = h
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error doing create Property request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK &&
		res.StatusCode != http.StatusCreated &&
		res.StatusCode != http.StatusAccepted {
		return fmt.Errorf(res.Status)
	}
	return nil
}

func updateProperty(endpoint string, h http.Header, b []byte) error {
	req, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("%s/property", endpoint),
		bytes.NewReader(b),
	)
	if err != nil {
		return fmt.Errorf("error creating update Property request: %w", err)
	}
	req.Header = h
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error doing update Property request: %w", err)
	}
	defer res.Body.Close()
	b, err = io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("could not read update Property response body: %w", err)
	}

	var body server.DefaultJSONResponse
	err = json.Unmarshal(b, &body)
	if err != nil {
		return fmt.Errorf("could not parse update Property response body: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response code for update Property: %s (%s)", res.Status, body.Error)
	}
	return nil
}

// helper function to POST /realtor
func createRealtor(end string, h http.Header, b []byte) error {
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/realtor", end),
		bytes.NewReader(b),
	)
	if err != nil {
		return fmt.Errorf("error constructing create Realtor request: %w", err)
	}
	req.Header = h
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error doing create Realtor request: %w", err)
	}
	defer res.Body.Close()
	b, err = io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("error reading create Realtor response body: %w", err)
	}
	var body server.DefaultJSONResponse
	err = json.Unmarshal(b, &body)
	if err != nil {
		return fmt.Errorf("could not parse create Realtor response body: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response code for create Realtor: %s (%s)", res.Status, body.Error)
	}
	return nil
}

// helper function to POST /property-events
func createPropertyEvents(end string, h http.Header, b []byte) error {
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/property-events", end),
		bytes.NewReader(b),
	)
	if err != nil {
		return fmt.Errorf("error constructing create PropertyEvents request: %w", err)
	}
	req.Header = h
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error doing create PropertyEvents request: %w", err)
	}
	defer res.Body.Close()
	b, err = io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("error reading create PropertyEvents response body: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("error with create PropertyEvents response: %s", string(b))
	}
	return nil
}
