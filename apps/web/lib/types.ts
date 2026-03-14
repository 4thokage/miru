export interface Order {
  rating?: string;
  follows?: string;
  lastChapter?: string;
  title?: string;
  year?: string;
}

export interface SearchRequest {
  title?: string;
  includedTags?: string[];
  excludedTags?: string[];
  order?: Order;
  contentRating?: string[];
  limit?: number;
  offset?: number;
}

export interface Manga {
  id: string;
  title: string;
  description: string;
  cover_art_id: string;
  cover_filename: string;
  last_chapter: string;
}

export interface Pagination {
  limit: number;
  offset: number;
  total: number;
}

export interface SearchResponse {
  data: Manga[];
  pagination: Pagination;
}

export interface APIResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}

export interface ChapterPages {
  baseUrl: string;
  hash: string;
  images: string[];
  mangaId: string;
}

export interface ProgressRequest {
  user_id: string;
  manga_id: string;
  chapter_id: string;
  page_number: number;
}

export interface Chapter {
  id: string;
  chapter: string;
  title: string;
  volume: string;
  pages: number;
  published: string;
  language: string;
}

export interface MangaDetails {
  manga: Manga;
  chapters: Chapter[];
}

export interface Anime {
  id: string;
  title: string;
  image: string;
  year?: string;
}

export interface AnimeDetails {
  id: string;
  title: string;
  image: string;
  description: string;
  status: string;
  genres: string[];
  episodes: AnimeEpisode[];
  total_episodes: number;
  released_year: string;
}

export interface AnimeEpisode {
  id: string;
  number: number;
  title?: string;
}

export interface StreamSource {
  server: string;
  url: string;
  quality: string;
  is_m3u8: boolean;
}

export interface AnimeSearchResult {
  animes: Anime[];
  page: number;
  has_next: boolean;
}

export interface RecentEpisode {
  id: string;
  anime_id: string;
  anime_title: string;
  image: string;
  episode: number;
  sub_or_dub: string;
}

export interface RecentResult {
  episodes: RecentEpisode[];
  page: number;
  has_next: boolean;
}
