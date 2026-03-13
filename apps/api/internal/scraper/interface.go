package scraper

import "context"

type AnimeProvider interface {
	Name() string
	Search(ctx context.Context, query string, page int) (*SearchResult, error)
	GetAnimeDetails(ctx context.Context, id string) (*AnimeDetails, error)
	GetEpisodes(ctx context.Context, animeID string) ([]Episode, error)
	GetStreamingLinks(ctx context.Context, episodeID string) ([]StreamSource, error)
	GetRecent(ctx context.Context, page int) (*RecentResult, error)
	GetPopular(ctx context.Context, page int) (*SearchResult, error)
}

type Anime struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Image string `json:"image"`
	Year  string `json:"year,omitempty"`
}

type SearchResult struct {
	Animes  []Anime `json:"animes"`
	Page    int     `json:"page"`
	HasNext bool    `json:"has_next"`
}

type AnimeDetails struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Image         string    `json:"image"`
	Description   string    `json:"description"`
	Status        string    `json:"status"`
	Genres        []string  `json:"genres"`
	Episodes      []Episode `json:"episodes"`
	TotalEpisodes int       `json:"total_episodes"`
	ReleasedYear  string    `json:"released_year"`
}

type Episode struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title,omitempty"`
}

type StreamSource struct {
	Server  string `json:"server"`
	URL     string `json:"url"`
	Quality string `json:"quality"`
	IsM3U8  bool   `json:"is_m3u8"`
}

type RecentResult struct {
	Episodes []RecentEpisode `json:"episodes"`
	Page     int             `json:"page"`
	HasNext  bool            `json:"has_next"`
}

type RecentEpisode struct {
	ID         string `json:"id"`
	AnimeID    string `json:"anime_id"`
	AnimeTitle string `json:"anime_title"`
	Image      string `json:"image"`
	Episode    int    `json:"episode"`
	SubOrDub   string `json:"sub_or_dub"`
}
