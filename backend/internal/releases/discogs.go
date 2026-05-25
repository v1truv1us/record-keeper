package releases

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type discogsSearch struct {
	Results []struct {
		ID          int      `json:"id"`
		Title       string   `json:"title"`
		Year        *int     `json:"year"`
		Label       []string `json:"label"`
		CoverImage  string   `json:"cover_image"`
		ResourceURL string   `json:"resource_url"`
	} `json:"results"`
}

// DiscogsClient searches releases via the Discogs API.
type DiscogsClient struct {
	client  *http.Client
	baseURL string
	key     string
	secret  string
}

// NewDiscogsClient creates a Discogs search client. Returns nil if credentials are missing.
func NewDiscogsClient(baseURL, key, secret string) *DiscogsClient {
	if key == "" || secret == "" {
		return nil
	}
	if baseURL == "" {
		baseURL = "https://api.discogs.com"
	}
	return &DiscogsClient{
		client:  &http.Client{Timeout: 6 * time.Second},
		baseURL: baseURL,
		key:     key,
		secret:  secret,
	}
}

func (d *DiscogsClient) Search(ctx context.Context, query string) ([]SearchResult, error) {
	return d.search(ctx, "q", query)
}

func (d *DiscogsClient) SearchBarcode(ctx context.Context, barcode string) ([]SearchResult, error) {
	return d.search(ctx, "barcode", barcode)
}

func (d *DiscogsClient) search(ctx context.Context, param, value string) ([]SearchResult, error) {
	reqURL := fmt.Sprintf("%s/database/search?type=release&per_page=8&%s=%s&key=%s&secret=%s",
		d.baseURL, param, url.QueryEscape(value), url.QueryEscape(d.key), url.QueryEscape(d.secret))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("discogs: build request: %w", err)
	}
	req.Header.Set("User-Agent", "AudioFile/0.2.0 +https://audiofile.app")

	res, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("discogs: request failed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discogs: unexpected status %d", res.StatusCode)
	}

	var discogs discogsSearch
	if err := json.NewDecoder(res.Body).Decode(&discogs); err != nil {
		return nil, fmt.Errorf("discogs: decode response: %w", err)
	}

	results := make([]SearchResult, 0, len(discogs.Results))
	for _, rel := range discogs.Results {
		artist, title := splitDiscogsTitle(rel.Title)
		results = append(results, SearchResult{
			MBID:     strconv.Itoa(rel.ID),
			Title:    title,
			Artist:   artist,
			Year:     rel.Year,
			Label:    firstString(rel.Label),
			CoverURL: rel.CoverImage,
		})
	}
	return results, nil
}

func splitDiscogsTitle(value string) (string, string) {
	parts := strings.SplitN(value, " - ", 2)
	if len(parts) != 2 {
		return "", value
	}
	return parts[0], parts[1]
}
