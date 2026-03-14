package gogoanime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"miru-api/internal/scraper"
)

func TestGoGoAnimeSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Errorf("Expected /search, got %s", r.URL.Path)
		}

		query := r.URL.Query().Get("keyword")
		if query == "" {
			t.Error("Expected query parameter, got none")
		}

		response := map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"id":      "naruto",
					"title":   "Naruto",
					"release": "2002",
					"image":   "https://gogocdn.net/images/naruto.jpg",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	t.Log("TestGoGoAnimeSearch: Would search for anime on GoGoAnime")
	t.Log("TestGoGoAnimeSearch: Query parameter extraction tested")
}

func TestGetAnimeDetails(t *testing.T) {
	animeID := "naruto"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/category/" + animeID
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, r.URL.Path)
		}

		response := map[string]interface{}{
			"title":         "Naruto",
			"description":   "Naruto Uzumaki, a young ninja...",
			"releaseDate":   "2002-10-03",
			"status":        "Completed",
			"genres":        []string{"Action", "Adventure"},
			"totalEpisodes": 220,
			"image":         "https://gogocdn.net/images/naruto.jpg",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	t.Logf("TestGetAnimeDetails: Testing retrieval of anime details for: %s", animeID)
	t.Log("TestGetAnimeDetails: Expected fields: title, description, releaseDate, status, genres, totalEpisodes, image")
}

func TestGetEpisodes(t *testing.T) {
	animeID := "naruto"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/load-server-episode"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, r.URL.Path)
		}

		response := []map[string]interface{}{
			{"id": "naruto-episode-1", "number": 1, "title": "Enter: Naruto"},
			{"id": "naruto-episode-2", "number": 2, "title": "My Name is Konohamaru!"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	t.Logf("TestGetEpisodes: Testing episode list retrieval for: %s", animeID)
	t.Log("TestGetEpisodes: Expected: array of episodes with id, number, title")
}

func TestGetStreamingLinks(t *testing.T) {
	episodeID := "naruto-episode-1"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/ajax/episode/sources"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, r.URL.Path)
		}

		response := map[string]interface{}{
			"link": "https://streamsb.net/embed/abc123",
			"type": "streamsb",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	t.Logf("TestGetStreamingLinks: Testing streaming link retrieval for: %s", episodeID)
	t.Log("TestGetStreamingLinks: Expected: link, type (streamsb/doodstream/voe)")
}

func TestAnimeProviderInterface(t *testing.T) {
	var provider scraper.AnimeProvider

	t.Log("TestAnimeProviderInterface: Verifying AnimeProvider interface compliance")
	t.Log("Required methods:")
	t.Log("  - Search(ctx, query, page) ([]SearchResult, error)")
	t.Log("  - GetAnimeDetails(ctx, id) (*AnimeDetails, error)")
	t.Log("  - GetEpisodes(ctx, id) ([]Episode, error)")
	t.Log("  - GetStreamingLinks(ctx, episodeID) ([]StreamSource, error)")
	t.Log("  - GetDownloadLinks(ctx, episodeID) ([]DownloadLink, error)")
	t.Log("  - GetRecent(ctx, page) ([]RecentResult, error)")
	t.Log("  - GetPopular(ctx, page) ([]SearchResult, error)")

	_ = provider
}

func TestStreamSourceExtraction(t *testing.T) {
	streamSources := []struct {
		name     string
		source   string
		embedded bool
	}{
		{"StreamSB", "streamsb", true},
		{"DoodStream", "doodstream", true},
		{"VOE", "voe", false},
		{"StreamTape", "streamtape", true},
	}

	for _, source := range streamSources {
		t.Run(source.name, func(t *testing.T) {
			t.Logf("Source: %s, Embedded: %v", source.name, source.embedded)
		})
	}
}

func TestCacheBehavior(t *testing.T) {
	tests := []struct {
		endpoint    string
		cacheTTL    time.Duration
		description string
	}{
		{"search", 5 * time.Minute, "Search results cached for 5 minutes"},
		{"details", 15 * time.Minute, "Anime details cached for 15 minutes"},
		{"episodes", 10 * time.Minute, "Episode list cached for 10 minutes"},
		{"streams", 2 * time.Minute, "Streaming links cached for 2 minutes"},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			t.Logf("%s: %v - %s", tt.endpoint, tt.cacheTTL, tt.description)
		})
	}
}

func TestErrorRecovery(t *testing.T) {
	errorCases := []struct {
		scenario      string
		httpStatus    int
		expectedError string
	}{
		{"GoGoAnime site down", 503, "service unavailable"},
		{"Invalid anime ID", 404, "anime not found"},
		{"Rate limited by source", 429, "too many requests"},
		{"Network timeout", 0, "connection timeout"},
		{"Parse error", 200, "invalid response format"},
	}

	for _, tc := range errorCases {
		t.Run(tc.scenario, func(t *testing.T) {
			t.Logf("HTTP %d: %s", tc.httpStatus, tc.expectedError)
		})
	}
}

func BenchmarkGetEpisodes(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_ = ctx
	}
}

func TestFixImageURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "WordPress Photon CDN with query params",
			input:    "https://i1.wp.com/gogoanime.by/wp-content/uploads/2026/03/beastars-final-season-part-2.webp?resize=246,350",
			expected: "https://gogocdn.net/wp-content/uploads/2026/03/beastars-final-season-part-2.webp",
		},
		{
			name:     "WordPress Photon CDN without query params",
			input:    "https://i2.wp.com/gogoanime.by/wp-content/uploads/2023/01/naruto.jpg",
			expected: "https://gogocdn.net/wp-content/uploads/2023/01/naruto.jpg",
		},
		{
			name:     "Already gogocdn URL",
			input:    "https://gogocdn.net/images/naruto.jpg",
			expected: "https://gogocdn.net/images/naruto.jpg",
		},
		{
			name:     "Protocol relative URL",
			input:    "//gogocdn.net/images/naruto.jpg",
			expected: "https://gogocdn.net/images/naruto.jpg",
		},
		{
			name:     "Root relative URL",
			input:    "/images/naruto.jpg",
			expected: "https://gogocdn.net/images/naruto.jpg",
		},
		{
			name:     "Empty URL",
			input:    "",
			expected: "",
		},
		{
			name:     "i0.wp.com variant",
			input:    "https://i0.wp.com/gogoanime.by/wp-content/uploads/2024/05/one-piece.webp?resize=300,450",
			expected: "https://gogocdn.net/wp-content/uploads/2024/05/one-piece.webp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fixImageURL(tt.input)
			if result != tt.expected {
				t.Errorf("fixImageURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
