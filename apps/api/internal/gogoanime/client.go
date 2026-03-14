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
	MainURL          = "https://gogoanime.by"
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
			// Allow redirects up to 10 times (default behavior)
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

	searchURL := fmt.Sprintf("%s/?s=%s&page=%d", MainURL, url.QueryEscape(query), page)
	doc, err := c.fetchDoc(ctx, searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch search page: %w", err)
	}

	var animes []scraper.Anime
	doc.Find(".listupd article.bs, .listupd article.bsx").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Find("a").Attr("href")

		// Try episode first, then series
		id := extractAnimeIDFromEpisode(link)
		if id == "" {
			id = extractSeriesID(link)
		}

		image, _ := s.Find(".limit img").Attr("src")

		// Get title - try h2 first (for search results), then .ttt > .tt (for recent/popular)
		title := ""
		h2 := s.Find("h2")
		if h2.Length() > 0 {
			title = strings.TrimSpace(h2.First().Text())
		} else {
			titleDiv := s.Find(".ttt > .tt")
			titleDiv.Contents().Each(func(i int, sel *goquery.Selection) {
				if sel.Is("h2") {
					return
				}
				title += sel.Text()
			})
			title = cleanText(title)
		}

		if id != "" && title != "" {
			animes = append(animes, scraper.Anime{
				ID:    id,
				Title: title,
				Image: fixImageURL(image),
			})
		}
	})

	hasNext := doc.Find(".hpage .r").Length() > 0

	result := &scraper.SearchResult{
		Animes:  animes,
		Page:    page,
		HasNext: hasNext,
	}

	c.setCache(ctx, cacheKey, result, SearchCacheTTL)
	return result, nil
}

func (c *Client) GetAnimeDetails(ctx context.Context, id string) (*scraper.AnimeDetails, error) {
	logger.Printf("[DEBUG] GetAnimeDetails: Starting for id=%s", id)
	start := time.Now()

	select {
	case <-ctx.Done():
		logger.Printf("[DEBUG] GetAnimeDetails: Context cancelled for id=%s", id)
		return nil, ctx.Err()
	default:
	}

	cacheKey := fmt.Sprintf("gogoanime:details:%s", id)
	if cached, ok := c.getCache(ctx, cacheKey).(*scraper.AnimeDetails); ok {
		logger.Printf("[DEBUG] GetAnimeDetails: Cache hit for id=%s", id)
		return cached, nil
	}
	logger.Printf("[DEBUG] GetAnimeDetails: Cache miss for id=%s", id)

	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	detailsURL := fmt.Sprintf("%s/series/%s", MainURL, id)
	logger.Printf("[DEBUG] GetAnimeDetails: Fetching %s", detailsURL)

	doc, err := c.fetchDoc(timeoutCtx, detailsURL)
	if err != nil {
		logger.Printf("[DEBUG] GetAnimeDetails: Failed to fetch details for id=%s: %v", id, err)
		return nil, fmt.Errorf("failed to fetch anime details: %w", err)
	}
	logger.Printf("[DEBUG] GetAnimeDetails: Successfully fetched document for id=%s", id)

	title := doc.Find(".entry-title").Text()
	image := doc.Find(".thumb img").AttrOr("src", "")
	description := doc.Find(".ninfo").Text()
	status := doc.Find(".spe").Text()
	logger.Printf("[DEBUG] GetAnimeDetails: Extracted basic info for id=%s, title=%s", id, title)

	var genres []string
	doc.Find(".genxed a").Each(func(i int, s *goquery.Selection) {
		genre := strings.TrimSpace(s.Text())
		if genre != "" {
			genres = append(genres, genre)
		}
	})

	logger.Printf("[DEBUG] GetAnimeDetails: Extracting episodes for id=%s", id)
	episodes := c.extractEpisodes(timeoutCtx, id, doc)
	logger.Printf("[DEBUG] GetAnimeDetails: Extracted %d episodes for id=%s", len(episodes), id)

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
	logger.Printf("[DEBUG] GetAnimeDetails: Completed for id=%s, found %d episodes (took %v)", id, len(episodes), time.Since(start))
	return details, nil
}

func (c *Client) GetEpisodes(ctx context.Context, animeID string) ([]scraper.Episode, error) {
	logger.Printf("[DEBUG] GetEpisodes: Starting for animeID=%s", animeID)
	start := time.Now()

	select {
	case <-ctx.Done():
		logger.Printf("[DEBUG] GetEpisodes: Context cancelled for animeID=%s", animeID)
		return nil, ctx.Err()
	default:
	}

	cacheKey := fmt.Sprintf("gogoanime:episodes:%s", animeID)
	if cached, ok := c.getCache(ctx, cacheKey).([]scraper.Episode); ok {
		logger.Printf("[DEBUG] GetEpisodes: Cache hit for animeID=%s", animeID)
		return cached, nil
	}
	logger.Printf("[DEBUG] GetEpisodes: Cache miss for animeID=%s", animeID)

	logger.Printf("[DEBUG] GetEpisodes: Calling GetAnimeDetails for animeID=%s", animeID)
	details, err := c.GetAnimeDetails(ctx, animeID)
	if err != nil {
		logger.Printf("[DEBUG] GetEpisodes: GetAnimeDetails failed for animeID=%s: %v", animeID, err)
		return nil, err
	}

	logger.Printf("[DEBUG] GetEpisodes: Got %d episodes from details for animeID=%s (took %v)", len(details.Episodes), animeID, time.Since(start))
	c.setCache(ctx, cacheKey, details.Episodes, EpisodesCacheTTL)
	return details.Episodes, nil
}

func (c *Client) extractEpisodes(ctx context.Context, id string, doc *goquery.Document) []scraper.Episode {
	logger.Printf("[DEBUG] extractEpisodes: Starting for id=%s", id)
	start := time.Now()

	select {
	case <-ctx.Done():
		logger.Printf("[DEBUG] extractEpisodes: Context cancelled for id=%s", id)
		return nil
	default:
	}

	var episodes []scraper.Episode

	episodesList := doc.Find(".episodes-container div a")
	logger.Printf("[DEBUG] extractEpisodes: Found %d episodes in container for id=%s", episodesList.Length(), id)

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

	logger.Printf("[DEBUG] extractEpisodes: Found %d episodes from main container for id=%s", len(episodes), id)

	if len(episodes) == 0 {
		logger.Printf("[DEBUG] extractEpisodes: Trying AJAX fallback for id=%s", id)
		ajaxURL := fmt.Sprintf("%s/ajax/load-list-episode?ep_start=0&ep_end=1000&id=%s&alias=%s", AjaxURL, id, id)
		ajaxDoc, err := c.fetchDoc(ctx, ajaxURL)
		if err != nil {
			logger.Printf("[DEBUG] extractEpisodes: AJAX fallback failed for id=%s: %v", id, err)
		} else {
			ajaxList := ajaxDoc.Find("li a")
			logger.Printf("[DEBUG] extractEpisodes: AJAX returned %d items for id=%s", ajaxList.Length(), id)
			ajaxList.Each(func(i int, s *goquery.Selection) {
				select {
				case <-ctx.Done():
					return
				default:
				}

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
			logger.Printf("[DEBUG] extractEpisodes: Found %d episodes from AJAX for id=%s", len(episodes), id)
		}
	}

	for i, j := 0, len(episodes)-1; i < j; i, j = i+1, j-1 {
		episodes[i], episodes[j] = episodes[j], episodes[i]
	}

	logger.Printf("[DEBUG] extractEpisodes: Completed for id=%s, found %d episodes (took %v)", id, len(episodes), time.Since(start))
	return episodes
}

func (c *Client) GetStreamingLinks(ctx context.Context, episodeID string) ([]scraper.StreamSource, error) {
	logger.Printf("[DEBUG] GetStreamingLinks: Starting for episodeID=%s", episodeID)
	start := time.Now()

	select {
	case <-ctx.Done():
		logger.Printf("[DEBUG] GetStreamingLinks: Context cancelled for episodeID=%s", episodeID)
		return nil, ctx.Err()
	default:
	}

	cacheKey := fmt.Sprintf("gogoanime:stream:%s", episodeID)
	if cached, ok := c.getCache(ctx, cacheKey).([]scraper.StreamSource); ok {
		logger.Printf("[DEBUG] GetStreamingLinks: Cache hit for episodeID=%s", episodeID)
		return cached, nil
	}
	logger.Printf("[DEBUG] GetStreamingLinks: Cache miss for episodeID=%s", episodeID)

	timeoutCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	episodeURL := fmt.Sprintf("%s/%s", MainURL, episodeID)
	logger.Printf("[DEBUG] GetStreamingLinks: Fetching episode page %s", episodeURL)

	doc, err := c.fetchDoc(timeoutCtx, episodeURL)
	if err != nil {
		logger.Printf("[DEBUG] GetStreamingLinks: Failed to fetch episode page for %s: %v", episodeID, err)
		return nil, fmt.Errorf("failed to fetch episode page: %w", err)
	}
	logger.Printf("[DEBUG] GetStreamingLinks: Successfully fetched episode page for %s", episodeID)

	var sources []scraper.StreamSource

	playerLinks := doc.Find(".player-type-link")
	logger.Printf("[DEBUG] GetStreamingLinks: Found %d player-type-link elements for %s", playerLinks.Length(), episodeID)

	playerLinks.Each(func(i int, s *goquery.Selection) {
		select {
		case <-timeoutCtx.Done():
			logger.Printf("[DEBUG] GetStreamingLinks: Context timeout during player-type-link iteration %d for %s", i, episodeID)
			return
		default:
		}

		server := s.Text()
		if server == "" {
			server = s.AttrOr("data-type", "")
		}

		encURL1 := s.AttrOr("data-encrypted-url1", "")
		encURL2 := s.AttrOr("data-encrypted-url2", "")
		encURL3 := s.AttrOr("data-encrypted-url3", "")
		plainURL := s.AttrOr("data-plain-url", "")
		dataType := s.AttrOr("data-type", "")

		logger.Printf("[DEBUG] GetStreamingLinks: Processing server=%s, type=%s for %s", server, dataType, episodeID)

		if encURL1 != "" {
			logger.Printf("[DEBUG] GetStreamingLinks: Extracting encrypted stream URL for server=%s", server)
			streamURL, quality := c.extractEncryptedStreamURL(timeoutCtx, encURL1, encURL2, encURL3, dataType, plainURL)
			if streamURL != "" {
				logger.Printf("[DEBUG] GetStreamingLinks: Got encrypted stream URL for server=%s: %s", server, streamURL)
				sources = append(sources, scraper.StreamSource{
					Server:  cleanServerName(server),
					URL:     streamURL,
					Quality: quality,
					IsM3U8:  strings.HasSuffix(streamURL, ".m3u8"),
				})
			} else {
				logger.Printf("[DEBUG] GetStreamingLinks: Failed to extract encrypted stream URL for server=%s", server)
			}
		} else if plainURL != "" {
			logger.Printf("[DEBUG] GetStreamingLinks: Using plain URL for server=%s: %s", server, plainURL)
			sources = append(sources, scraper.StreamSource{
				Server:  cleanServerName(server),
				URL:     plainURL,
				Quality: "unknown",
				IsM3U8:  strings.HasSuffix(plainURL, ".m3u8"),
			})
		}
	})

	if len(sources) == 0 {
		mutiLinks := doc.Find(".anime_muti_link a")
		logger.Printf("[DEBUG] GetStreamingLinks: No player-type-link sources, trying %d anime_muti_link elements for %s", mutiLinks.Length(), episodeID)

		mutiLinks.Each(func(i int, s *goquery.Selection) {
			select {
			case <-timeoutCtx.Done():
				logger.Printf("[DEBUG] GetStreamingLinks: Context timeout during anime_muti_link iteration %d for %s", i, episodeID)
				return
			default:
			}

			server := s.AttrOr("title", "")
			embedData := s.AttrOr("data-value", "")

			if embedData == "" {
				embedData = s.AttrOr("href", "")
			}

			logger.Printf("[DEBUG] GetStreamingLinks: Processing anime_muti_link server=%s for %s", server, episodeID)
			streamURL, quality := c.extractStreamURL(timeoutCtx, embedData, server)
			if streamURL != "" {
				logger.Printf("[DEBUG] GetStreamingLinks: Got stream URL from anime_muti_link for server=%s: %s", server, streamURL)
				sources = append(sources, scraper.StreamSource{
					Server:  cleanServerName(server),
					URL:     streamURL,
					Quality: quality,
					IsM3U8:  strings.HasSuffix(streamURL, ".m3u8"),
				})
			} else {
				logger.Printf("[DEBUG] GetStreamingLinks: Failed to extract stream URL from anime_muti_link for server=%s", server)
			}
		})
	}

	if len(sources) == 0 {
		script := doc.Find("script").Text()
		if match := regexp.MustCompile(`sources\s*:\s*\[(.*?)\]`).FindStringSubmatch(script); len(match) > 1 {
			logger.Printf("[DEBUG] GetStreamingLinks: Found inline sources in script for episode %s", episodeID)
		}
	}

	sources = prioritizeServers(sources)
	logger.Printf("[DEBUG] GetStreamingLinks: Completed for episodeID=%s, found %d sources (took %v)", episodeID, len(sources), time.Since(start))

	c.setCache(ctx, cacheKey, sources, StreamCacheTTL)
	return sources, nil
}

func (c *Client) GetDownloadLinks(ctx context.Context, episodeID string) ([]scraper.DownloadLink, error) {
	logger.Printf("[DEBUG] GetDownloadLinks: Starting for episodeID=%s", episodeID)
	start := time.Now()

	select {
	case <-ctx.Done():
		logger.Printf("[DEBUG] GetDownloadLinks: Context cancelled for episodeID=%s", episodeID)
		return nil, ctx.Err()
	default:
	}

	cacheKey := fmt.Sprintf("gogoanime:download:%s", episodeID)
	if cached, ok := c.getCache(ctx, cacheKey).([]scraper.DownloadLink); ok {
		logger.Printf("[DEBUG] GetDownloadLinks: Cache hit for episodeID=%s", episodeID)
		return cached, nil
	}
	logger.Printf("[DEBUG] GetDownloadLinks: Cache miss for episodeID=%s", episodeID)

	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	episodeURL := fmt.Sprintf("%s/%s", MainURL, episodeID)
	logger.Printf("[DEBUG] GetDownloadLinks: Fetching episode page %s", episodeURL)

	doc, err := c.fetchDoc(timeoutCtx, episodeURL)
	if err != nil {
		logger.Printf("[DEBUG] GetDownloadLinks: Failed to fetch episode page for %s: %v", episodeID, err)
		return nil, fmt.Errorf("failed to fetch episode page: %w", err)
	}
	logger.Printf("[DEBUG] GetDownloadLinks: Successfully fetched episode page for %s", episodeID)

	var downloads []scraper.DownloadLink

	// Look for download box containers
	dlBox := doc.Find(".dlbox")
	logger.Printf("[DEBUG] GetDownloadLinks: Found %d dlbox elements for %s", dlBox.Length(), episodeID)

	if dlBox.Length() > 0 {
		dlBox.Find("ul li").Each(func(i int, s *goquery.Selection) {
			select {
			case <-timeoutCtx.Done():
				return
			default:
			}

			server := s.Find("span").Text()
			linkElem := s.Find("a")
			href := linkElem.AttrOr("href", "")
			quality := linkElem.AttrOr("title", "")

			if href != "" {
				logger.Printf("[DEBUG] GetDownloadLinks: Found download link for server=%s", server)
				downloads = append(downloads, scraper.DownloadLink{
					Server:  cleanServerName(server),
					URL:     href,
					Quality: quality,
				})
			}
		})
	}

	// Also check for any direct download links
	if len(downloads) == 0 {
		doc.Find("a[href*='download'], a[href*='.mp4'], a[href*='.mkv']").Each(func(i int, s *goquery.Selection) {
			select {
			case <-timeoutCtx.Done():
				return
			default:
			}

			href := s.AttrOr("href", "")
			text := strings.TrimSpace(s.Text())

			if href != "" && (strings.Contains(href, ".mp4") || strings.Contains(href, ".mkv") || strings.Contains(text, "Download")) {
				logger.Printf("[DEBUG] GetDownloadLinks: Found direct download link: %s", href)
				downloads = append(downloads, scraper.DownloadLink{
					Server:  "download",
					URL:     href,
					Quality: "unknown",
				})
			}
		})
	}

	logger.Printf("[DEBUG] GetDownloadLinks: Completed for episodeID=%s, found %d download links (took %v)", episodeID, len(downloads), time.Since(start))

	c.setCache(ctx, cacheKey, downloads, StreamCacheTTL)
	return downloads, nil
}

func (c *Client) extractStreamURL(ctx context.Context, embedData, server string) (string, string) {
	select {
	case <-ctx.Done():
		return "", ""
	default:
	}

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

func (c *Client) extractEncryptedStreamURL(ctx context.Context, enc1, enc2, enc3, serverType, plainURL string) (string, string) {
	logger.Printf("[DEBUG] extractEncryptedStreamURL: Starting for serverType=%s", serverType)
	start := time.Now()

	select {
	case <-ctx.Done():
		logger.Printf("[DEBUG] extractEncryptedStreamURL: Context cancelled")
		return "", ""
	default:
	}

	directTypes := []string{"embed", "kiwi"}

	for _, dt := range directTypes {
		if strings.EqualFold(serverType, dt) && plainURL != "" {
			logger.Printf("[DEBUG] extractEncryptedStreamURL: Direct type %s with plainURL, returning immediately", dt)
			return plainURL, "unknown"
		}
	}

	apiURL := "https://9animetv.be/wp-content/plugins/video-player/includes/player/player.php"
	logger.Printf("[DEBUG] extractEncryptedStreamURL: Calling API %s with serverType=%s", apiURL, serverType)

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, "GET", apiURL, nil)
	if err != nil {
		logger.Printf("[DEBUG] extractEncryptedStreamURL: Failed to create request: %v", err)
		return "", ""
	}

	q := req.URL.Query()
	q.Set(serverType, enc1)
	if enc2 != "" {
		q.Set("url2", enc2)
	}
	if enc3 != "" {
		q.Set("url3", enc3)
	}
	q.Set("user_agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.3 Safari/605.1.15")
	q.Set("ref", "gogoanime.by")
	req.URL.RawQuery = q.Encode()

	logger.Printf("[DEBUG] extractEncryptedStreamURL: Sending request to API")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Printf("[DEBUG] extractEncryptedStreamURL: API request failed: %v (took %v)", err, time.Since(start))
		return "", ""
	}
	defer resp.Body.Close()

	logger.Printf("[DEBUG] extractEncryptedStreamURL: API response status=%d (took %v)", resp.StatusCode, time.Since(start))

	if resp.StatusCode != 200 {
		logger.Printf("[DEBUG] extractEncryptedStreamURL: API returned non-200 status: %d", resp.StatusCode)
		return "", ""
	}

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	logger.Printf("[DEBUG] extractEncryptedStreamURL: Response body length=%d", len(body))

	// Look for iframe src specifically - the API returns an iframe with the actual player
	// The response may also contain <script src="..."> tags (like JW Player library),
	// so we need to be specific about finding the iframe src
	if iframeMatch := regexp.MustCompile(`<iframe[^>]+src=["']([^"']+)["']`).FindStringSubmatch(html); len(iframeMatch) > 1 {
		logger.Printf("[DEBUG] extractEncryptedStreamURL: Found iframe src: %s (took %v)", iframeMatch[1], time.Since(start))
		return iframeMatch[1], "unknown"
	}

	// Fallback: look for any m3u8 URLs in the response
	if m3u8Match := regexp.MustCompile(`["']([^"']+\.m3u8[^"]*)["']`).FindStringSubmatch(html); len(m3u8Match) > 1 {
		logger.Printf("[DEBUG] extractEncryptedStreamURL: Found m3u8 URL: %s (took %v)", m3u8Match[1], time.Since(start))
		return m3u8Match[1], "1080p"
	}

	logger.Printf("[DEBUG] extractEncryptedStreamURL: No stream URL found in response (took %v)", time.Since(start))
	return "", ""
}

func (c *Client) extractStreamsb(ctx context.Context, embedURL string) (string, string) {
	select {
	case <-ctx.Done():
		return "", ""
	default:
	}

	parsedURL := extractURL(embedURL)
	if parsedURL == "" {
		return "", ""
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	doc, err := c.fetchDoc(timeoutCtx, parsedURL)
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
	apiDoc, err := c.fetchDoc(timeoutCtx, apiURL)
	if err == nil {
		apiScript := apiDoc.Find("script").Text()
		if m3u8Match := regexp.MustCompile(`["']([^"']+\.m3u8[^"']*)["']`).FindStringSubmatch(apiScript); len(m3u8Match) > 1 {
			return m3u8Match[1], "720p"
		}
	}

	return "", ""
}

func (c *Client) extractDoodstream(ctx context.Context, embedURL string) (string, string) {
	select {
	case <-ctx.Done():
		return "", ""
	default:
	}

	parsedURL := extractURL(embedURL)
	if parsedURL == "" {
		return "", ""
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(timeoutCtx, "GET", parsedURL, nil)
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
	select {
	case <-ctx.Done():
		return "", ""
	default:
	}

	parsedURL := extractURL(embedURL)
	if parsedURL == "" {
		return "", ""
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	doc, err := c.fetchDoc(timeoutCtx, parsedURL)
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

	recentURL := fmt.Sprintf("%s/?page=%d", MainURL, page)
	doc, err := c.fetchDoc(ctx, recentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recent page: %w", err)
	}

	var episodes []scraper.RecentEpisode
	doc.Find(".listupd article.bs").Each(func(i int, s *goquery.Selection) {
		link := s.Find("a").AttrOr("href", "")
		epID := extractEpisodeID(link)
		animeID := extractAnimeIDFromEpisode(link)
		image := s.Find(".limit img").AttrOr("src", "")

		// Get anime title from the div.tt text (not including h2)
		titleDiv := s.Find(".ttt > .tt")
		title := ""
		titleDiv.Contents().Each(func(i int, sel *goquery.Selection) {
			if sel.Is("h2") {
				return
			}
			title += sel.Text()
		})
		title = cleanText(title)

		epNumStr := s.Find(".limit .bt .epx").Text()
		epNum := extractEpisodeNumber(epNumStr)

		subOrDub := "sub"
		if strings.Contains(strings.ToLower(s.Find(".limit .bt .sb").Text()), "dub") {
			subOrDub = "dub"
		}

		if epID != "" && animeID != "" {
			episodes = append(episodes, scraper.RecentEpisode{
				ID:         epID,
				AnimeID:    animeID,
				AnimeTitle: title,
				Image:      fixImageURL(image),
				Episode:    epNum,
				SubOrDub:   subOrDub,
			})
		}
	})

	hasNext := doc.Find(".hpage .r").Length() > 0

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

	// Popular page no longer exists, use homepage
	popularURL := fmt.Sprintf("%s/?page=%d", MainURL, page)
	doc, err := c.fetchDoc(ctx, popularURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch popular page: %w", err)
	}

	var animes []scraper.Anime
	doc.Find(".listupd article.bs").Each(func(i int, s *goquery.Selection) {
		link := s.Find("a").AttrOr("href", "")
		id := extractAnimeIDFromEpisode(link)
		image := s.Find(".limit img").AttrOr("src", "")

		titleDiv := s.Find(".ttt > .tt")
		title := ""
		titleDiv.Contents().Each(func(i int, sel *goquery.Selection) {
			if sel.Is("h2") {
				return
			}
			title += sel.Text()
		})
		title = cleanText(title)

		if id != "" && title != "" {
			animes = append(animes, scraper.Anime{
				ID:    id,
				Title: title,
				Image: fixImageURL(image),
			})
		}
	})

	hasNext := doc.Find(".hpage .r").Length() > 0

	result := &scraper.SearchResult{
		Animes:  animes,
		Page:    page,
		HasNext: hasNext,
	}

	c.setCache(ctx, cacheKey, result, SearchCacheTTL)
	return result, nil
}

func (c *Client) fetchDoc(ctx context.Context, fetchURL string) (*goquery.Document, error) {
	logger.Printf("[DEBUG] fetchDoc: Starting request to %s", fetchURL)
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", fetchURL, nil)
	if err != nil {
		logger.Printf("[DEBUG] fetchDoc: Failed to create request for %s: %v", fetchURL, err)
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	logger.Printf("[DEBUG] fetchDoc: Sending request to %s", fetchURL)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Printf("[DEBUG] fetchDoc: Request failed for %s: %v (took %v)", fetchURL, err, time.Since(start))
		return nil, err
	}
	defer resp.Body.Close()
	logger.Printf("[DEBUG] fetchDoc: Got response %d from %s (took %v)", resp.StatusCode, fetchURL, time.Since(start))

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, fetchURL)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		logger.Printf("[DEBUG] fetchDoc: Failed to parse HTML from %s: %v", fetchURL, err)
		return nil, err
	}

	logger.Printf("[DEBUG] fetchDoc: Successfully parsed document from %s (took %v)", fetchURL, time.Since(start))
	return doc, nil
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

func extractAnimeIDFromEpisode(link string) string {
	if link == "" {
		return ""
	}
	// Extract just the path
	re := regexp.MustCompile(`/([^/]+)-episode-\d+`)
	match := re.FindStringSubmatch(link)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

func extractSeriesID(link string) string {
	if link == "" {
		return ""
	}
	// Extract from /series/xxx/ URL
	re := regexp.MustCompile(`/series/([^/]+)`)
	match := re.FindStringSubmatch(link)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

func extractEpisodeID(link string) string {
	re := regexp.MustCompile(`/([^/]+)-episode-(\d+)`)
	match := re.FindStringSubmatch(link)
	if match == nil {
		return ""
	}
	return match[1] + "-episode-" + match[2]
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

	// Handle protocol-relative URLs
	if strings.HasPrefix(url, "//") {
		url = "https:" + url
	}

	// Handle root-relative URLs
	if strings.HasPrefix(url, "/") {
		return BaseURL + url
	}

	// Convert WordPress Photon CDN URLs (i0.wp.com, i1.wp.com, i2.wp.com, i3.wp.com)
	// back to the original gogoanime.by URL for better compatibility
	// Photon URL format: https://i0.wp.com/gogoanime.by/wp-content/uploads/...
	if strings.Contains(url, ".wp.com/gogoanime.by/") {
		// Extract the path after the domain
		parts := strings.SplitN(url, ".wp.com/gogoanime.by/", 2)
		if len(parts) == 2 {
			// Remove query parameters (like ?resize=246,350)
			path := parts[1]
			if idx := strings.Index(path, "?"); idx != -1 {
				path = path[:idx]
			}
			url = BaseURL + "/" + path
		}
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
