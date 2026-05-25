package releases

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	searchers []ReleaseSearcher
	discogs   *DiscogsClient
}

type searchCacheEntry struct {
	results []SearchResult
	expires time.Time
}

var searchCache = struct {
	sync.Mutex
	items map[string]searchCacheEntry
}{items: map[string]searchCacheEntry{}}

// NewHandler creates a release handler with the given search sources.
func NewHandler(searchers ...ReleaseSearcher) *Handler {
	return &Handler{
		searchers:      searchers,
	}
}

// SetDiscogs sets the Discogs client for barcode scanning.
func (h *Handler) SetDiscogs(d *DiscogsClient) {
	h.discogs = d
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/search", h.search)
	r.Post("/scan", h.scan)
	return r
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		http.Error(w, "q is required", http.StatusBadRequest)
		return
	}

	key := strings.ToLower(q)
	now := time.Now()
	searchCache.Lock()
	entry, ok := searchCache.items[key]
	if ok && now.Before(entry.expires) {
		searchCache.Unlock()
		writeJSON(w, entry.results)
		return
	}
	searchCache.Unlock()

	results, ok := h.searchResults(w, r, q)
	if !ok {
		return
	}
	cacheResults(key, results, now)
	writeJSON(w, results)
}

func (h *Handler) scan(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Barcode string `json:"barcode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "barcode is required", http.StatusBadRequest)
		return
	}
	barcode := cleanBarcode(payload.Barcode)
	if barcode == "" {
		http.Error(w, "barcode is required", http.StatusBadRequest)
		return
	}
	if h.discogs == nil {
		http.Error(w, "barcode scanning requires Discogs credentials", http.StatusServiceUnavailable)
		return
	}
	results, err := h.discogs.SearchBarcode(r.Context(), barcode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]any{"barcode": barcode, "results": results})
}

func (h *Handler) searchResults(w http.ResponseWriter, r *http.Request, q string) ([]SearchResult, bool) {
	var lastErr error
	for _, s := range h.searchers {
		results, err := s.Search(r.Context(), q)
		if err != nil {
			lastErr = err
			continue
		}
		if len(results) > 0 {
			return results, true
		}
	}

	if lastErr != nil {
		http.Error(w, "release search failed. Try again later or enter the release details manually.", http.StatusBadGateway)
		return nil, false
	}

	return nil, true
}

func cacheResults(key string, results []SearchResult, now time.Time) {
	searchCache.Lock()
	searchCache.items[key] = searchCacheEntry{results: results, expires: now.Add(12 * time.Hour)}
	searchCache.Unlock()
}

func cleanBarcode(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(value)
}
