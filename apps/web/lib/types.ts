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
}

export interface MangaDetails {
  manga: Manga;
  chapters: Chapter[];
}
