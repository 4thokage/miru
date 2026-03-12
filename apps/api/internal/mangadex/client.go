package mangadex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

var logger = slog.Default()

const (
	MangaDexBaseURL   = "https://api.mangadex.org"
	MangaDexCoversURL = "https://uploads.mangadex.org/covers"
	TagCacheKey       = "mangadex:tags"
	SearchCacheTTL    = 5 * time.Minute
	ChapterCacheTTL   = 10 * time.Minute
)

type Client struct {
	httpClient *http.Client
	redis      *redis.Client
	tagCache   TagCache
	mu         sync.RWMutex
}

func NewClient(redisClient *redis.Client) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		redis:      redisClient,
		tagCache:   make(TagCache),
	}
}

func (c *Client) InitTags(ctx context.Context) error {
	cachedTags, err := c.loadTagCache(ctx)
	if err == nil && len(cachedTags) > 0 {
		c.mu.Lock()
		c.tagCache = cachedTags
		c.mu.Unlock()
		log.Printf("Loaded %d tags from Valkey cache", len(cachedTags))
		return nil
	}

	log.Println("Fetching tags from MangaDex API...")
	tags, err := c.fetchTags(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch tags: %w", err)
	}

	c.mu.Lock()
	c.tagCache = tags
	c.mu.Unlock()

	if err := c.saveTagCache(ctx, tags); err != nil {
		log.Printf("Warning: failed to save tag cache: %v", err)
	}

	log.Printf("Fetched and cached %d tags", len(tags))
	return nil
}

func (c *Client) fetchTags(ctx context.Context) (TagCache, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, MangaDexBaseURL+"/manga/tag", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Result string `json:"result"`
		Data   []struct {
			ID         string `json:"id"`
			Attributes struct {
				Name map[string]string `json:"name"`
			} `json:"attributes"`
		} `json:"data"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	tags := make(TagCache)
	for _, tag := range result.Data {
		name := tag.Attributes.Name["en"]
		if name != "" {
			tags[name] = tag.ID
		}
	}

	return tags, nil
}

func (c *Client) loadTagCache(ctx context.Context) (TagCache, error) {
	data, err := c.redis.Get(ctx, TagCacheKey).Bytes()
	if err != nil {
		return nil, err
	}

	var tags TagCache
	if err := json.Unmarshal(data, &tags); err != nil {
		return nil, err
	}

	return tags, nil
}

func (c *Client) saveTagCache(ctx context.Context, tags TagCache) error {
	data, err := json.Marshal(tags)
	if err != nil {
		return err
	}

	return c.redis.Set(ctx, TagCacheKey, data, 24*time.Hour).Err()
}

func (c *Client) GetTagUUID(name string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	uuid, ok := c.tagCache[name]
	return uuid, ok
}

func (c *Client) GetAllTags() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	tags := make([]string, 0, len(c.tagCache))
	for name := range c.tagCache {
		tags = append(tags, name)
	}
	return tags
}

func (c *Client) SearchManga(ctx context.Context, params SearchRequest) (*SearchResponse, error) {
	if params.ContentRating == nil {
		params.ContentRating = []string{"safe", "suggestive"}
	}
	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Limit > 100 {
		params.Limit = 100
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	cacheKey := c.buildCacheKey(params)
	cached, err := c.getCachedResults(ctx, cacheKey)
	if err == nil && cached != nil {
		return cached, nil
	}

	queryParams := c.buildQueryParams(params)
	searchURL := MangaDexBaseURL + "/manga?" + queryParams.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mangadex API error: %d - %s", resp.StatusCode, string(body))
	}

	var mdResp struct {
		Result   string `json:"result"`
		Response string `json:"response"`
		Total    int    `json:"total"`
		Offset   int    `json:"offset"`
		Limit    int    `json:"limit"`
		Data     []struct {
			ID         string `json:"id"`
			Attributes struct {
				Title         map[string]string `json:"title"`
				Description   map[string]string `json:"description"`
				LastChapter   string            `json:"lastChapter"`
				Year          *int              `json:"year"`
				ContentRating string            `json:"contentRating"`
				Tags          []struct {
					ID string `json:"id"`
				} `json:"tags"`
			} `json:"attributes"`
			Relationships []struct {
				ID         string `json:"id"`
				Type       string `json:"type"`
				Attributes struct {
					FileName string `json:"fileName"`
				} `json:"attributes"`
			} `json:"relationships"`
		} `json:"data"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(body, &mdResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := &SearchResponse{
		Data: make([]Manga, 0, len(mdResp.Data)),
		Pagination: Pagination{
			Limit:  params.Limit,
			Offset: params.Offset,
			Total:  mdResp.Total,
		},
	}

	for _, m := range mdResp.Data {
		title := ""
		if t, ok := m.Attributes.Title["en"]; ok {
			title = t
		} else {
			for _, t := range m.Attributes.Title {
				title = t
				break
			}
		}

		desc := ""
		if d, ok := m.Attributes.Description["en"]; ok {
			desc = d
		}

		coverID := ""
		coverFileName := ""
		for _, rel := range m.Relationships {
			if rel.Type == "cover_art" {
				coverID = rel.ID
				coverFileName = rel.Attributes.FileName
				break
			}
		}

		result.Data = append(result.Data, Manga{
			ID:            m.ID,
			Title:         title,
			Description:   desc,
			CoverArtID:    coverID,
			CoverFileName: coverFileName,
			LastChapter:   m.Attributes.LastChapter,
		})
	}

	c.cacheResults(ctx, cacheKey, result)

	return result, nil
}

func (c *Client) buildCacheKey(params SearchRequest) string {
	cacheParams := struct {
		Title         string   `json:"title"`
		IncludedTags  []string `json:"includedTags"`
		ExcludedTags  []string `json:"excludedTags"`
		Order         Order    `json:"order"`
		ContentRating []string `json:"contentRating"`
		Limit         int      `json:"limit"`
		Offset        int      `json:"offset"`
		Includes      []string `json:"includes"`
	}{
		Title:         params.Title,
		IncludedTags:  params.IncludedTags,
		ExcludedTags:  params.ExcludedTags,
		Order:         params.Order,
		ContentRating: params.ContentRating,
		Limit:         params.Limit,
		Offset:        params.Offset,
		Includes:      []string{"cover_art", "author", "artist"},
	}

	data, _ := json.Marshal(cacheParams)
	hash := sha256.Sum256(data)
	return "mangadex:search:" + hex.EncodeToString(hash[:])
}

func (c *Client) getCachedResults(ctx context.Context, key string) (*SearchResponse, error) {
	data, err := c.redis.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var result SearchResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) cacheResults(ctx context.Context, key string, result *SearchResponse) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return c.redis.Set(ctx, key, data, SearchCacheTTL).Err()
}

func (c *Client) buildQueryParams(params SearchRequest) url.Values {
	q := url.Values{}

	if params.Title != "" {
		q.Set("title", params.Title)
	}

	for _, tagName := range params.IncludedTags {
		if uuid, ok := c.GetTagUUID(tagName); ok {
			q.Add("includedTags[]", uuid)
		}
	}

	for _, tagName := range params.ExcludedTags {
		if uuid, ok := c.GetTagUUID(tagName); ok {
			q.Add("excludedTags[]", uuid)
		}
	}

	if params.Order.Rating != "" {
		q.Set("order[rating]", params.Order.Rating)
	}
	if params.Order.Follows != "" {
		q.Set("order[followedCount]", params.Order.Follows)
	}
	if params.Order.LastChapter != "" {
		q.Set("order[lastChapter]", params.Order.LastChapter)
	}
	if params.Order.Title != "" {
		q.Set("order[title]", params.Order.Title)
	}
	if params.Order.Year != "" {
		q.Set("order[year]", params.Order.Year)
	}

	for _, cr := range params.ContentRating {
		q.Add("contentRating[]", cr)
	}

	q.Set("limit", fmt.Sprintf("%d", params.Limit))
	q.Set("offset", fmt.Sprintf("%d", params.Offset))
	q.Add("includes[]", "cover_art")
	q.Add("includes[]", "author")
	q.Add("includes[]", "artist")

	return q
}

func GetCoverURL(mangaID, fileName string, size int) string {
	if fileName == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s/%s.%d.jpg", MangaDexCoversURL, mangaID, fileName, size)
}

func GetEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func (c *Client) GetChapterPages(ctx context.Context, chapterID string) (*ChapterPages, error) {
	cacheKey := fmt.Sprintf("chapter:%s", chapterID)

	cached, err := c.getCachedChapter(ctx, cacheKey)
	if err == nil && cached != nil {
		logger.Info("chapter pages cache hit", slog.String("chapter_id", chapterID))
		return cached, nil
	}

	logger.Info("fetching chapter pages from MangaDex", slog.String("chapter_id", chapterID))

	url := fmt.Sprintf("%s/at-home/server/%s", MangaDexBaseURL, chapterID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chapter: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mangadex API error: %d - %s", resp.StatusCode, string(body))
	}

	var mdResp struct {
		Result  string `json:"result"`
		BaseURL string `json:"baseUrl"`
		Chapter struct {
			Hash    string   `json:"hash"`
			Data    []string `json:"data"`
			MangaID string   `json:"mangaId"`
		} `json:"chapter"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(body, &mdResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	images := make([]string, len(mdResp.Chapter.Data))
	for i, filename := range mdResp.Chapter.Data {
		images[i] = fmt.Sprintf("%s/data/%s/%s", mdResp.BaseURL, mdResp.Chapter.Hash, filename)
	}

	result := &ChapterPages{
		BaseURL: mdResp.BaseURL,
		Hash:    mdResp.Chapter.Hash,
		Images:  images,
		MangaID: mdResp.Chapter.MangaID,
	}

	if err := c.cacheChapter(ctx, cacheKey, result); err != nil {
		logger.Warn("failed to cache chapter pages", slog.String("error", err.Error()))
	}

	logger.Info("chapter pages fetched successfully",
		slog.String("chapter_id", chapterID),
		slog.Int("image_count", len(images)))

	return result, nil
}

func (c *Client) getCachedChapter(ctx context.Context, key string) (*ChapterPages, error) {
	data, err := c.redis.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var chapter ChapterPages
	if err := json.Unmarshal(data, &chapter); err != nil {
		return nil, err
	}

	return &chapter, nil
}

func (c *Client) cacheChapter(ctx context.Context, key string, chapter *ChapterPages) error {
	data, err := json.Marshal(chapter)
	if err != nil {
		return err
	}

	return c.redis.Set(ctx, key, data, ChapterCacheTTL).Err()
}

func (c *Client) GetMangaDetails(ctx context.Context, mangaID string) (*MangaDetails, error) {
	mangaURL := fmt.Sprintf("%s/manga/%s?includes[]=cover_art&includes[]=author&includes[]=artist", MangaDexBaseURL, mangaID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mangaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manga: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mangadex API error: %d - %s", resp.StatusCode, string(body))
	}

	var mangaResp struct {
		Result   string `json:"result"`
		Response string `json:"response"`
		Data     struct {
			ID         string `json:"id"`
			Attributes struct {
				Title       map[string]string `json:"title"`
				Description map[string]string `json:"description"`
			} `json:"attributes"`
			Relationships []struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"relationships"`
		} `json:"data"`
		Included []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				FileName string `json:"fileName"`
			} `json:"attributes"`
		} `json:"included"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(body, &mangaResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	m := mangaResp.Data
	title := ""
	if t, ok := m.Attributes.Title["en"]; ok {
		title = t
	} else {
		for _, t := range m.Attributes.Title {
			title = t
			break
		}
	}

	desc := ""
	if d, ok := m.Attributes.Description["en"]; ok {
		desc = d
	}

	coverID := ""
	for _, rel := range m.Relationships {
		if rel.Type == "cover_art" {
			coverID = rel.ID
			break
		}
	}

	coverFileName := ""
	for _, inc := range mangaResp.Included {
		if inc.Type == "cover_art" && inc.ID == coverID {
			coverFileName = inc.Attributes.FileName
			break
		}
	}

	manga := Manga{
		ID:            m.ID,
		Title:         title,
		Description:   desc,
		CoverArtID:    coverID,
		CoverFileName: coverFileName,
	}

	chapters, err := c.GetMangaChapters(ctx, mangaID)
	if err != nil {
		logger.Warn("failed to fetch chapters", slog.String("error", err.Error()))
		chapters = []Chapter{}
	}

	return &MangaDetails{
		Manga:    manga,
		Chapters: chapters,
	}, nil
}

func (c *Client) GetMangaChapters(ctx context.Context, mangaID string) ([]Chapter, error) {
	feedURL := fmt.Sprintf("%s/manga/%s/feed?limit=100&includes[]=scanlation_group&order[chapter]=desc", MangaDexBaseURL, mangaID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chapters: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mangadex API error: %d - %s", resp.StatusCode, string(body))
	}

	var feedResp struct {
		Result   string `json:"result"`
		Response string `json:"response"`
		Data     []struct {
			ID         string `json:"id"`
			Attributes struct {
				Chapter   string `json:"chapter"`
				Title     string `json:"title"`
				Volume    string `json:"volume"`
				Pages     int    `json:"pages"`
				Published string `json:"publishAt"`
			} `json:"attributes"`
		} `json:"data"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(body, &feedResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	chapters := make([]Chapter, 0, len(feedResp.Data))
	for _, ch := range feedResp.Data {
		chapters = append(chapters, Chapter{
			ID:        ch.ID,
			Chapter:   ch.Attributes.Chapter,
			Title:     ch.Attributes.Title,
			Volume:    ch.Attributes.Volume,
			Pages:     ch.Attributes.Pages,
			Published: ch.Attributes.Published,
		})
	}

	return chapters, nil
}
