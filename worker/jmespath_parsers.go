package worker

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jmespath/go-jmespath"
)

// NOTE: callers pass in just they payload object from the response body
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

// NOTE: callers pass in just the payload object from the response body
func jmesParseInitialInfoParams(p string, data interface{}) (interface{}, error) {
	switch p {
	case "property_id":
		return jmespath.Search("propertyId", data)
	case "listing_id":
		return jmespath.Search("listingId", data)
	case "latitude":
		return jmespath.Search("latLong.latitude", data)
	case "longitude":
		return jmespath.Search("latLong.longitude", data)
	default:
		return "", fmt.Errorf("unsupported param: %s", p)
	}
}

// NOTE: callers pass in just the payload object from the response body
func jmesParseMLSParams(p string, data interface{}) (interface{}, error) {
	switch p {
	case "zipcode":
		return jmespath.Search("publicRecordsInfo.addressInfo.zip", data)
	case "city":
		return jmespath.Search("publicRecordsInfo.addressInfo.city", data)
	case "state":
		return jmespath.Search("publicRecordsInfo.addressInfo.state", data)
	case "price":
		// first get the number of events
		nevents, err := jmespath.Search("length(propertyHistoryInfo.events)", data)
		if err != nil {
			return nil, fmt.Errorf("could not parse events")
		}
		// iterate over the events and grab the first price entry
		for i := range int(math.Round(nevents.(float64))) {
			path := fmt.Sprintf("propertyHistoryInfo.events[%d].price", i)
			res, err := jmespath.Search(path, data)
			if err != nil {
				return "", fmt.Errorf("error accessing property history events: %w", err)
			}
			if res != nil {
				return res, nil
			}
		}
		return nil, fmt.Errorf("no price found in property history events")
	case "events":
		events := []historyEvent{}
		nevents, err := jmespath.Search("length(propertyHistoryInfo.events)", data)
		if err != nil {
			return nil, fmt.Errorf("could not parse events")
		}
		if nevents == nil {
			return events, nil
		}
		for i := range int(math.Round(nevents.(float64))) {
			pricePath := fmt.Sprintf("propertyHistoryInfo.events[%d].price", i)
			descPath := fmt.Sprintf("propertyHistoryInfo.events[%d].eventDescription", i)
			srcPath := fmt.Sprintf("propertyHistoryInfo.events[%d].source", i)
			srcIDPath := fmt.Sprintf("propertyHistoryInfo.events[%d].sourceId", i)
			tsPath := fmt.Sprintf("propertyHistoryInfo.events[%d].eventDate", i)

			// ignore errors here since they're unimportant
			price, _ := jmespath.Search(pricePath, data)
			desc, _ := jmespath.Search(descPath, data)
			src, _ := jmespath.Search(srcPath, data)
			srcID, _ := jmespath.Search(srcIDPath, data)
			ts, _ := jmespath.Search(tsPath, data)
			pe := historyEvent{}
			if price != nil {
				pe.Price = int32(math.Round(price.(float64)))
			}
			if desc != nil {
				pe.EventDescription = desc.(string)
			}
			if src != nil {
				pe.Source = src.(string)
			}
			if srcID != nil {
				pe.SourceID = srcID.(string)
			}
			if ts != nil {
				pe.EventTS = time.Unix(0, int64(time.Millisecond)*int64(math.Round(ts.(float64))))
			}
			events = append(events, pe)
		}
		return events, nil
	case "image_urls":
		res, err := jmespath.Search("propertyHistoryInfo.mediaBrowserInfoBySourceId.*.photos[].thumbnailData.thumbnailUrl", data)
		if err != nil {
			return nil, err
		}
		ress, ok := res.([]interface{})
		if !ok {
			return nil, fmt.Errorf("could not handle jmes return type")
		}
		urls := []string{}
		for _, rv := range ress {
			urls = append(urls, rv.(string))
		}
		return urls, nil
	default:
		return nil, fmt.Errorf("unsupported param: %s", p)
	}
}

type historyEvent struct {
	Price            int32
	EventDescription string
	Source           string
	SourceID         string
	EventTS          time.Time
}
