package jsonb

type PropertyScrapeMetadata struct {
	InitialInfoHash string `json:"initial_info_hash"`
	MLSHash         string `json:"mls_hash"`
	AVMHash         string `json:"avm_hash"`
}
