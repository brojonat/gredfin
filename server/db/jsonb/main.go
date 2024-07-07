package jsonb

type PropertyScrapeMetadata struct {
	ThumbnailURLs   []string `json:"thumbnail_urls"`
	ImageURLs       []string `json:"image_urls"`
	InitialInfoHash string   `json:"initial_info_hash"`
	MLSHash         string   `json:"mls_hash"`
	AVMHash         string   `json:"avm_hash"`
}
