package server

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/brojonat/gredfin/server/dbgen"
)

func getPropertyBucket() (string, error) {
	b := os.Getenv("S3_PROPERTY_BUCKET")
	if b == "" {
		return "", fmt.Errorf("s3 property bucket not set")
	}
	return b, nil
}

func getPropertyKey(ctx context.Context, q *dbgen.Queries, pid, lid int32, basename string) (string, error) {
	p, err := q.GetPropertyBasic(ctx, dbgen.GetPropertyBasicParams{PropertyID: pid, ListingID: lid})
	if err != nil {
		return "", err
	}
	addr := strings.TrimPrefix(p.URL.String, "https://www.redfin.com/")
	if addr == p.URL.String {
		return "", fmt.Errorf("unable to parse url to address %s", p.URL.String)
	}
	return fmt.Sprintf("property/%s/%d_%d_%d_%s", addr, pid, lid, time.Now().Unix(), basename), nil
}
