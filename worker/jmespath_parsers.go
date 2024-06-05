package worker

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jmespath/go-jmespath"
)

// parse the MLSInfo bytes and extract the realtor name and company
func parseRealtorInfo(b []byte) (string, string, error) {
	var data interface{}
	if err := json.Unmarshal(b, &data); err != nil {
		return "", "", fmt.Errorf("error parsing MLS data for realtor: %w", err)
	}
	result, err := jmespath.Search("propertyHistoryInfo.mediaBrowserInfoBySourceId.*.photoAttribution | [0]", data)
	if err != nil {
		return "", "", fmt.Errorf("error searching MLS data for realtor: %w", err)
	}
	if result == nil {
		return "", "", fmt.Errorf("null result searching MLS data for realtor")
	}
	parts := strings.Split(result.(string), "•")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("unexpected format for realtor name/company: `%s`", result.(string))
	}
	// remove leading/trailing whitespace and an expected prefix
	name := strings.ReplaceAll(parts[0], "Listed by ", "")
	name = strings.TrimLeft(name, " ")
	name = strings.TrimRight(name, " ")
	// remove leading/trailing whitespace and a trailing period
	company := strings.TrimLeft(parts[1], " ")
	company = strings.TrimRight(company, " .")
	return name, company, nil
}

// parse the MLSInfo bytes and extract the price
func parseListingPrice(b []byte) (int32, error) {
	var data interface{}
	if err := json.Unmarshal(b, &data); err != nil {
		return 0, err
	}
	p, err := jmespath.Search("propertyHistoryInfo.events[0].price", data)
	if err != nil {
		return 0, err
	}
	price, ok := p.(float64)
	if !ok {
		return 0, fmt.Errorf("invalid value for price: %s", p)
	}
	return int32(price), nil
}
