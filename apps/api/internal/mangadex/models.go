package mangadex

type SearchRequest struct {
	Title         string   `json:"title"`
	IncludedTags  []string `json:"includedTags"`
	ExcludedTags  []string `json:"excludedTags"`
	Order         Order    `json:"order"`
	ContentRating []string `json:"contentRating"`
	Limit         int      `json:"limit"`
	Offset        int      `json:"offset"`
}

type Order struct {
	Rating      string `json:"rating"`
	Follows     string `json:"follows"`
	LastChapter string `json:"lastChapter"`
	Title       string `json:"title"`
	Year        string `json:"year"`
}

type Manga struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	CoverArtID    string `json:"cover_art_id"`
	CoverFileName string `json:"cover_filename"`
	LastChapter   string `json:"last_chapter"`
}

type SearchResponse struct {
	Data       []Manga    `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

type MangaDexTag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TagCache map[string]string

type ChapterPages struct {
	BaseURL string   `json:"baseUrl"`
	Hash    string   `json:"hash"`
	Images  []string `json:"images"`
	MangaID string   `json:"mangaId"`
}

type ChapterCache struct {
	BaseURL string   `json:"baseUrl"`
	Hash    string   `json:"hash"`
	Images  []string `json:"images"`
	MangaID string   `json:"mangaId"`
}

type Chapter struct {
	ID        string `json:"id"`
	Chapter   string `json:"chapter"`
	Title     string `json:"title"`
	Volume    string `json:"volume"`
	Pages     int    `json:"pages"`
	Published string `json:"published"`
}

type MangaDetails struct {
	Manga    Manga     `json:"manga"`
	Chapters []Chapter `json:"chapters"`
}
