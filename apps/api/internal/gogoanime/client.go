package gogoanime

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
	"regexp"
	"strconv"
	"strings"
	"time"

	"miru-api/internal/scraper"

	"github.com/PuerkitoBio/goquery"
	"github.com/redis/go-redis/v9"
)

const (
	BaseURL          = "https://gogocdn.net"
	MainURL          = "https://gogoanime.sk"
	AjaxURL          = "https://ajax.gogocdn.net"
	SearchCacheTTL   = 5 * time.Minute
	DetailsCacheTTL  = 15 * time.Minute
	EpisodesCacheTTL = 10 * time.Minute
	StreamCacheTTL   = 2 * time.Minute
)

var logger = log.Default()

type Client struct {
	httpClient *http.Client
	redis      *redis.Client
}

func NewClient(redisClient *redis.Client) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		redis: redisClient,
	}
}

func (c *Client) Name() string {
	return "gogoanime"
}

func (c *Client) Search(ctx context.Context, query string, page int) (*scraper.SearchResult, error) {
	cacheKey := fmt.Sprintf("gogoanime:search:%s:%d", hashQuery(query), page)
	if cached, ok := c.getCache(ctx, cacheKey).(*scraper.SearchResult); ok {
		return cached, nil
	}

	searchURL := fmt.Sprintf("%s/search.html?keyword=%s&page=%d", MainURL, url.QueryEscape(query), page)
	doc, err := c.fetchDoc(ctx, searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch search page: %w", err)
	}

	var animes []scraper.Anime
	doc.Find(".last_episodes .anime-card").Each(func(i int, s *goquery.Selection) {
		link := s.Find("a").AttrOr("href", "")
		id := extractAnimeID(link)
		image := s.Find("img").AttrOr("src", "")
		title := s.Find(".anime-title").Text()
		year := s.Find(".released").Text()

		if id != "" && title != "" {
			animes = append(animes, scraper.Anime{
				ID:    id,
				Title: strings.TrimSpace(title),
				Image: fixImageURL(image),
				Year:  extractYear(strings.TrimSpace(year)),
			})
		}
	})

	hasNext := len(animes) >= 20

	result := &scraper.SearchResult{
		Animes:  animes,
		Page:    page,
		HasNext: hasNext,
	}

	c.setCache(ctx, cacheKey, result, SearchCacheTTL)
	return result, nil
}

func (c *Client) GetAnimeDetails(ctx context.Context, id string) (*scraper.AnimeDetails, error) {
	cacheKey := fmt.Sprintf("gogoanime:details:%s", id)
	if cached, ok := c.getCache(ctx, cacheKey).(*scraper.AnimeDetails); ok {
		return cached, nil
	}

	detailsURL := fmt.Sprintf("%s/category/%s", MainURL, id)
	doc, err := c.fetchDoc(ctx, detailsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch anime details: %w", err)
	}

	title := doc.Find(".anime-info-title").Text()
	image := doc.Find(".anime-info-body img").AttrOr("src", "")
	description := doc.Find(".description").Text()
	status := doc.Find(".status").Text()

	var genres []string
	doc.Find(".genres a").Each(func(i int, s *goquery.Selection) {
		genre := strings.TrimSpace(s.Text())
		if genre != "" {
			genres = append(genres, genre)
		}
	})

	episodes := c.extractEpisodes(ctx, id, doc)

	releasedYear := ""
	if yearMatch := regexp.MustCompile(`(\d{4})`).FindString(description); yearMatch != "" {
		releasedYear = yearMatch
	}

	details := &scraper.AnimeDetails{
		ID:            id,
		Title:         strings.TrimSpace(title),
		Image:         fixImageURL(image),
		Description:   cleanText(description),
		Status:        cleanText(status),
		Genres:        genres,
		Episodes:      episodes,
		TotalEpisodes: len(episodes),
		ReleasedYear:  releasedYear,
	}

	c.setCache(ctx, cacheKey, details, DetailsCacheTTL)
	return details, nil
}

func (c *Client) GetEpisodes(ctx context.Context, animeID string) ([]scraper.Episode, error) {
	cacheKey := fmt.Sprintf("gogoanime:episodes:%s", animeID)
	if cached, ok := c.getCache(ctx, cacheKey).([]scraper.Episode); ok {
		return cached, nil
	}

	details, err := c.GetAnimeDetails(ctx, animeID)
	if err != nil {
		return nil, err
	}

	c.setCache(ctx, cacheKey, details.Episodes, EpisodesCacheTTL)
	return details.Episodes, nil
}

func (c *Client) extractEpisodes(ctx context.Context, id string, doc *goquery.Document) []scraper.Episode {
	var episodes []scraper.Episode

	episodesList := doc.Find("#episode_related li a")
	if episodesList.Length() > 0 {
		episodesList.Each(func(i int, s *goquery.Selection) {
			href := s.AttrOr("href", "")
			epID := extractEpisodeID(href)
			epNum := s.Find(".name").Text()
			num := extractEpisodeNumber(epNum)

			if epID != "" {
				episodes = append(episodes, scraper.Episode{
					ID:     epID,
					Number: num,
					Title:  strings.TrimSpace(epNum),
				})
			}
		})
	}

	if len(episodes) == 0 {
		ajaxURL := fmt.Sprintf("%s/ajax/load-list-episode?ep_start=0&ep_end=1000&id=%s&alias=%s", AjaxURL, id, id)
		doc, err := c.fetchDoc(ctx, ajaxURL)
		if err == nil {
			doc.Find("li a").Each(func(i int, s *goquery.Selection) {
				href := s.AttrOr("href", "")
				epID := extractEpisodeID(href)
				epNum := s.Find(".name").Text()
				num := extractEpisodeNumber(epNum)

				if epID != "" {
					episodes = append(episodes, scraper.Episode{
						ID:     epID,
						Number: num,
						Title:  strings.TrimSpace(epNum),
					})
				}
			})
		}
	}

	for i, j := 0, len(episodes)-1; i < j; i, j = i+1, j-1 {
		episodes[i], episodes[j] = episodes[j], episodes[i]
	}

	return episodes
}

func (c *Client) GetStreamingLinks(ctx context.Context, episodeID string) ([]scraper.StreamSource, error) {
	cacheKey := fmt.Sprintf("gogoanime:stream:%s", episodeID)
	if cached, ok := c.getCache(ctx, cacheKey).([]scraper.StreamSource); ok {
		return cached, nil
	}

	episodeURL := fmt.Sprintf("%s/%s", MainURL, episodeID)
	doc, err := c.fetchDoc(ctx, episodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch episode page: %w", err)
	}

	var sources []scraper.StreamSource

	doc.Find(".anime_muti_link a").Each(func(i int, s *goquery.Selection) {
		server := s.AttrOr("title", "")
		embedData := s.AttrOr("data-value", "")

		if embedData == "" {
			embedData = s.AttrOr("href", "")
		}

		streamURL, quality := c.extractStreamURL(ctx, embedData, server)
		if streamURL != "" {
			sources = append(sources, scraper.StreamSource{
				Server:  cleanServerName(server),
				URL:     streamURL,
				Quality: quality,
				IsM3U8:  strings.HasSuffix(streamURL, ".m3u8"),
			})
		}
	})

	if len(sources) == 0 {
		script := doc.Find("script").Text()
		if match := regexp.MustCompile(`sources\s*:\s*\[(.*?)\]`).FindStringSubmatch(script); len(match) > 1 {
			logger.Printf("Found inline sources in script for episode %s", episodeID)
		}
	}

	sources = prioritizeServers(sources)

	c.setCache(ctx, cacheKey, sources, StreamCacheTTL)
	return sources, nil
}

func (c *Client) extractStreamURL(ctx context.Context, embedData, server string) (string, string) {
	server = cleanServerName(server)

	if strings.Contains(server, "streamsb") || strings.Contains(embedData, "streamsb") {
		return c.extractStreamsb(ctx, embedData)
	}

	if strings.Contains(server, "doodstream") || strings.Contains(embedData, "doodstream") {
		return c.extractDoodstream(ctx, embedData)
	}

	if strings.Contains(server, "voe") || strings.Contains(embedData, "voe") {
		return c.extractVoe(ctx, embedData)
	}

	parsedURL := extractURL(embedData)
	if parsedURL == "" {
		return "", ""
	}

	if strings.Contains(parsedURL, "streamsb") {
		return c.extractStreamsb(ctx, parsedURL)
	}

	if strings.Contains(parsedURL, "doodstream") {
		return c.extractDoodstream(ctx, parsedURL)
	}

	if strings.Contains(parsedURL, "voe") {
		return c.extractVoe(ctx, parsedURL)
	}

	return embedData, "unknown"
}

func (c *Client) extractStreamsb(ctx context.Context, embedURL string) (string, string) {
	parsedURL := extractURL(embedURL)
	if parsedURL == "" {
		return "", ""
	}

	doc, err := c.fetchDoc(ctx, parsedURL)
	if err != nil {
		return "", ""
	}

	script := doc.Find("script").Text()

	if m3u8Match := regexp.MustCompile(`["']([^"']+\.m3u8[^"']*)["']`).FindStringSubmatch(script); len(m3u8Match) > 1 {
		return m3u8Match[1], "1080p"
	}

	if m3u8Match := regexp.MustCompile(`sources\s*:\s*\[(.*?)\]`).FindStringSubmatch(script); len(m3u8Match) > 1 {
		if urlMatch := regexp.MustCompile(`file["']?\s*:\s*["']([^"']+\.m3u8[^"']*)["']`).FindStringSubmatch(m3u8Match[1]); len(urlMatch) > 1 {
			return urlMatch[1], "720p"
		}
	}

	apiURL := strings.Replace(parsedURL, "/e/", "/sources/", 1)
	apiDoc, err := c.fetchDoc(ctx, apiURL)
	if err == nil {
		apiScript := apiDoc.Find("script").Text()
		if m3u8Match := regexp.MustCompile(`["']([^"']+\.m3u8[^"']*)["']`).FindStringSubmatch(apiScript); len(m3u8Match) > 1 {
			return m3u8Match[1], "720p"
		}
	}

	return "", ""
}

func (c *Client) extractDoodstream(ctx context.Context, embedURL string) (string, string) {
	parsedURL := extractURL(embedURL)
	if parsedURL == "" {
		return "", ""
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", parsedURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	md5Match := regexp.MustCompile(`'/([a-f0-9]+)/[^"']*'`).FindStringSubmatch(html)
	if len(md5Match) > 1 {
		md5 := md5Match[1]
		doodURL := fmt.Sprintf("https://doodstream.com/e/%s", md5)
		return doodURL, "1080p"
	}

	return "", ""
}

func (c *Client) extractVoe(ctx context.Context, embedURL string) (string, string) {
	parsedURL := extractURL(embedURL)
	if parsedURL == "" {
		return "", ""
	}

	doc, err := c.fetchDoc(ctx, parsedURL)
	if err != nil {
		return "", ""
	}

	video := doc.Find("video source")
	src := video.AttrOr("src", "")
	if src != "" {
		return src, "1080p"
	}

	script := doc.Find("script").Text()
	if m3u8Match := regexp.MustCompile(`["']([^"']+\.m3u8[^"']*)["']`).FindStringSubmatch(script); len(m3u8Match) > 1 {
		return m3u8Match[1], "1080p"
	}

	return "", ""
}

func (c *Client) GetRecent(ctx context.Context, page int) (*scraper.RecentResult, error) {
	cacheKey := fmt.Sprintf("gogoanime:recent:%d", page)
	if cached, ok := c.getCache(ctx, cacheKey).(*scraper.RecentResult); ok {
		return cached, nil
	}

	recentURL := fmt.Sprintf("%s/home.html?page=%d", MainURL, page)
	doc, err := c.fetchDoc(ctx, recentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recent page: %w", err)
	}

	var episodes []scraper.RecentEpisode
	doc.Find(".last_episodes .anime-card").Each(func(i int, s *goquery.Selection) {
		link := s.Find("a").AttrOr("href", "")
		epID := extractEpisodeID(link)
		animeID := extractAnimeID(link)
		image := s.Find("img").AttrOr("src", "")
		title := s.Find(".anime-title").Text()
		epNumStr := s.Find(".episode-number").Text()

		epNum := extractEpisodeNumber(epNumStr)

		subOrDub := "sub"
		if strings.Contains(strings.ToLower(epNumStr), "dub") {
			subOrDub = "dub"
		}

		if epID != "" && animeID != "" {
			episodes = append(episodes, scraper.RecentEpisode{
				ID:         epID,
				AnimeID:    animeID,
				AnimeTitle: strings.TrimSpace(title),
				Image:      fixImageURL(image),
				Episode:    epNum,
				SubOrDub:   subOrDub,
			})
		}
	})

	hasNext := len(episodes) >= 20

	result := &scraper.RecentResult{
		Episodes: episodes,
		Page:     page,
		HasNext:  hasNext,
	}

	c.setCache(ctx, cacheKey, result, SearchCacheTTL)
	return result, nil
}

func (c *Client) GetPopular(ctx context.Context, page int) (*scraper.SearchResult, error) {
	cacheKey := fmt.Sprintf("gogoanime:popular:%d", page)
	if cached, ok := c.getCache(ctx, cacheKey).(*scraper.SearchResult); ok {
		return cached, nil
	}

	popularURL := fmt.Sprintf("%s/popular.html?page=%d", MainURL, page)
	doc, err := c.fetchDoc(ctx, popularURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch popular page: %w", err)
	}

	var animes []scraper.Anime
	doc.Find(".last_episodes .anime-card").Each(func(i int, s *goquery.Selection) {
		link := s.Find("a").AttrOr("href", "")
		id := extractAnimeID(link)
		image := s.Find("img").AttrOr("src", "")
		title := s.Find(".anime-title").Text()
		year := s.Find(".released").Text()

		if id != "" && title != "" {
			animes = append(animes, scraper.Anime{
				ID:    id,
				Title: strings.TrimSpace(title),
				Image: fixImageURL(image),
				Year:  extractYear(strings.TrimSpace(year)),
			})
		}
	})

	hasNext := len(animes) >= 20

	result := &scraper.SearchResult{
		Animes:  animes,
		Page:    page,
		HasNext: hasNext,
	}

	c.setCache(ctx, cacheKey, result, SearchCacheTTL)
	return result, nil
}

func (c *Client) fetchDoc(ctx context.Context, url string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 302 {
		redirect := resp.Header.Get("Location")
		if redirect != "" {
			return c.fetchDoc(ctx, redirect)
		}
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

func (c *Client) getCache(ctx context.Context, key string) interface{} {
	if c.redis == nil {
		return nil
	}

	data, err := c.redis.Get(ctx, key).Bytes()
	if err != nil {
		return nil
	}

	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}

	return result
}

func (c *Client) setCache(ctx context.Context, key string, value interface{}, ttl time.Duration) {
	if c.redis == nil {
		return
	}

	data, err := json.Marshal(value)
	if err != nil {
		return
	}

	c.redis.Set(ctx, key, data, ttl)
}

func hashQuery(query string) string {
	hash := sha256.Sum256([]byte(query))
	return hex.EncodeToString(hash[:])[:16]
}

func extractAnimeID(link string) string {
	if link == "" {
		return ""
	}
	parts := strings.Split(strings.TrimSuffix(link, "/"), "/")
	return parts[len(parts)-1]
}

func extractEpisodeID(link string) string {
	re := regexp.MustCompile(`-episode-\d+`)
	match := re.FindString(link)
	if match == "" {
		re := regexp.MustCompile(`/(\w+-\d+)$`)
		match = re.FindString(link)
	}
	return match
}

func extractEpisodeNumber(text string) int {
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(text)
	if match == "" {
		return 0
	}
	num, _ := strconv.Atoi(match)
	return num
}

func extractYear(text string) string {
	re := regexp.MustCompile(`\d{4}`)
	match := re.FindString(text)
	return match
}

func extractURL(text string) string {
	re := regexp.MustCompile(`https?://[^\s"'>]+`)
	match := re.FindString(text)
	return match
}

func fixImageURL(url string) string {
	if url == "" {
		return ""
	}
	if strings.HasPrefix(url, "//") {
		return "https:" + url
	}
	if strings.HasPrefix(url, "/") {
		return BaseURL + url
	}
	return url
}

func cleanText(text string) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.TrimSpace(text)
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return text
}

func cleanServerName(name string) string {
	name = strings.ToLower(name)
	name = strings.TrimSpace(name)
	return name
}

func prioritizeServers(sources []scraper.StreamSource) []scraper.StreamSource {
	order := []string{"streamsb", "doodstream", "voe", "streamtape", "mixdrop"}

	var result []scraper.StreamSource
	var others []scraper.StreamSource

	for _, s := range sources {
		found := false
		for _, p := range order {
			if strings.Contains(strings.ToLower(s.Server), p) {
				result = append(result, s)
				found = true
				break
			}
		}
		if !found {
			others = append(others, s)
		}
	}

	return append(result, others...)
}
