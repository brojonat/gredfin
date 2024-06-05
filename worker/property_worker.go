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
	"time"

	"github.com/brojonat/gredfin/redfin"
	"github.com/brojonat/gredfin/server"
	"github.com/brojonat/gredfin/server/dbgen"
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
		// them. Pass them to handlePropertyBytes which will process them.
		err = handlePropertyBytes(end, h, l, p, iiRes.Payload, mlsRes.Payload, avmRes.Payload)
		if err != nil {
			logPropertyError(l, "error handling property data, marking scrape bad", err, p)
			if err = markPropertyScrapeBad(end, h, pid, lid); err != nil {
				logPropertyError(l, "error marking scrape bad", err, p)
			}
			return
		}

		// Done processing this property, so update this property on the server
		// with this scrape's metadata (status, timestamp, and hashes). The same
		// error handling as above is done here too. Note that if an error is
		// encountered here the scrape will be marked bad even though we've
		// already handled the data returned by the Redfin client.
		payload := struct {
			dbgen.CreatePropertyParams
			server.PropertyScrapeMetadata
			Status       string `json:"status"`
			LastScrapeTs string `json:"last_scrape_ts"`
		}{
			dbgen.CreatePropertyParams{PropertyID: p.PropertyID, ListingID: p.ListingID, URL: p.URL},
			server.PropertyScrapeMetadata{
				InitialInfoHash: hashBytes(iiRes.Payload),
				MLSHash:         hashBytes(mlsRes.Payload),
				AVMHash:         hashBytes(avmRes.Payload),
			},
			server.ScrapeStatusGood,
			time.Now().Format(time.RFC3339),
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

	// helper function to upload realtor data
	uploadRealtor := func(r *dbgen.CreateRealtorParams) error {
		b, err := json.Marshal(r)
		if err != nil {
			return fmt.Errorf("could not serialize realtor for create request: %w", err)
		}
		req, err := http.NewRequest(
			http.MethodPost,
			fmt.Sprintf("%s/realtor", end),
			bytes.NewReader(b),
		)
		if err != nil {
			return fmt.Errorf("error constructing create realtor request: %w", err)
		}
		req.Header = h
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("error doing create realtor request: %w", err)
		}
		defer res.Body.Close()
		b, err = io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("error reading response body: %w", err)
		}
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("error with create realtor response: %s", string(b))
		}
		return nil
	}

	// helper function to upload bytes to S3 if the hash is different from the last scrape
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

	// Here we can do a number of things. We can parse the data and upload
	// aspects of it to the server. For instance, we can upload the realtor and
	// price info. Additionally, we can upload to S3. The maybeUpload function
	// will do that, but first it will check the hash of the bytes and only
	// upload if the bytes have changed from the last scrape. However, note that
	// some response bodies change on just about every request, so this on its
	// own isn't super effective at reducing storage. Instead, we can extract
	// certain parts of the response that are likely to change frequently and
	// only hash/upload the "stable" parts of the response.

	// parse and upload the realtor data to the server
	name, company, err := parseRealtorInfo(mlsb)
	if err != nil {
		return fmt.Errorf("error extracting realtor: %w", err)
	}
	lp, err := parseListingPrice(mlsb)
	if err != nil {
		return fmt.Errorf("error extracting list price: %w", err)
	}
	r := dbgen.CreateRealtorParams{
		PropertyID: p.PropertyID,
		ListingID:  p.ListingID,
		ListPrice:  lp,
		Name:       name,
		Company:    company,
	}
	if err := uploadRealtor(&r); err != nil {
		return fmt.Errorf("error uploading realtor: %w", err)
	}

	// now do S3 uploads
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
	payload := struct {
		PropertyID int32  `json:"property_id"`
		ListingID  int32  `json:"listing_id"`
		Status     string `json:"status"`
	}{
		pid, lid, server.ScrapeStatusBad,
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

func createProperty(endpoint string, h http.Header, c *dbgen.CreatePropertyParams) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/property-query", endpoint),
		bytes.NewReader(b),
	)
	if err != nil {
		return err
	}
	req.Header = h
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return fmt.Errorf(res.Status)
	}
	return nil
}

func updateProperty(endpoint string, h http.Header, b []byte) error {
	req, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("%s/property-query", endpoint),
		bytes.NewReader(b),
	)
	if err != nil {
		return err
	}
	req.Header = h
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf(res.Status)
	}
	return nil
}
