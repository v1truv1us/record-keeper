package releases

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type musicBrainzSearch struct {
	Releases []struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		Date         string `json:"date"`
		ArtistCredit []struct {
			Name string `json:"name"`
		} `json:"artist-credit"`
		LabelInfo []struct {
			Label *struct {
				Name string `json:"name"`
			} `json:"label"`
		} `json:"label-info"`
	} `json:"releases"`
}

// MusicBrainzClient searches releases via the MusicBrainz API.
type MusicBrainzClient struct {
	client  *http.Client
	baseURL string
}

// NewMusicBrainzClient creates a MusicBrainz search client.
func NewMusicBrainzClient(baseURL string) *MusicBrainzClient {
	if baseURL == "" {
		baseURL = "https://musicbrainz.org"
	}
	return &MusicBrainzClient{
		client:  &http.Client{Timeout: 6 * time.Second},
		baseURL: baseURL,
	}
}

func (m *MusicBrainzClient) Search(ctx context.Context, query string) ([]SearchResult, error) {
	mbQuery := buildMBQuery(query)
	reqURL := m.baseURL + "/ws/2/release/?fmt=json&limit=8&query=" + url.QueryEscape(mbQuery)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("musicbrainz: build request: %w", err)
	}
	req.Header.Set("User-Agent", "AudioFile/0.2.0 (https://github.com/v1truv1us/audiofile)")

	res, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("musicbrainz: request failed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("musicbrainz: unexpected status %d", res.StatusCode)
	}

	var mb musicBrainzSearch
	if err := json.NewDecoder(res.Body).Decode(&mb); err != nil {
		return nil, fmt.Errorf("musicbrainz: decode response: %w", err)
	}

	results := make([]SearchResult, 0, len(mb.Releases))
	for _, rel := range mb.Releases {
		results = append(results, SearchResult{
			MBID:     rel.ID,
			Title:    rel.Title,
			Artist:   artistName(rel.ArtistCredit),
			Year:     releaseYear(rel.Date),
			Label:    labelName(rel.LabelInfo),
			CoverURL: "https://coverartarchive.org/release/" + rel.ID + "/front-250",
		})
	}
	return results, nil
}

// SearchBarcode is not supported by MusicBrainz — returns empty results.
func (m *MusicBrainzClient) SearchBarcode(_ context.Context, _ string) ([]SearchResult, error) {
	return nil, nil
}

func buildMBQuery(q string) string {
	if parts := strings.SplitN(q, " - ", 2); len(parts) == 2 {
		artist := strings.TrimSpace(parts[0])
		title := strings.TrimSpace(parts[1])
		if artist != "" && title != "" {
			return fmt.Sprintf("artist:\"%s\" AND release:\"%s\"", artist, title)
		}
	}
	words := strings.Fields(q)
	if len(words) <= 2 {
		return fmt.Sprintf("artist:\"%s\" OR release:\"%s\"", q, q)
	}
	return q
}

func artistName(credits []struct {
	Name string `json:"name"`
}) string {
	parts := make([]string, 0, len(credits))
	for _, credit := range credits {
		if credit.Name != "" {
			parts = append(parts, credit.Name)
		}
	}
	return strings.Join(parts, "")
}

func labelName(labels []struct {
	Label *struct {
		Name string `json:"name"`
	} `json:"label"`
}) string {
	for _, info := range labels {
		if info.Label != nil && info.Label.Name != "" {
			return info.Label.Name
		}
	}
	return ""
}

func releaseYear(date string) *int {
	if len(date) < 4 {
		return nil
	}
	year := 0
	for _, ch := range date[:4] {
		if ch < '0' || ch > '9' {
			return nil
		}
		year = year*10 + int(ch-'0')
	}
	return &year
}
