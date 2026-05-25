package releases

import "context"

// SearchResult represents a release match from any search source.
type SearchResult struct {
	MBID     string `json:"mbid"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Year     *int   `json:"year,omitempty"`
	Label    string `json:"label,omitempty"`
	CoverURL string `json:"coverUrl,omitempty"`
}

// ReleaseSearcher searches for releases by text query or barcode.
type ReleaseSearcher interface {
	Search(ctx context.Context, query string) ([]SearchResult, error)
	SearchBarcode(ctx context.Context, barcode string) ([]SearchResult, error)
}
