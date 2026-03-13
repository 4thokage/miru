import { APIResponse, SearchRequest, SearchResponse, ChapterPages, ProgressRequest, MangaDetails, AnimeDetails, AnimeSearchResult, AnimeEpisode, StreamSource, RecentResult } from './types';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export async function searchAnime(query: string, page = 1): Promise<AnimeSearchResult> {
  const response = await fetch(`${API_URL}/api/v1/anime/search`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ query, page }),
  });

  if (!response.ok) {
    throw new Error('Failed to search anime');
  }

  const data: APIResponse<AnimeSearchResult> = await response.json();
  
  if (!data.success || !data.data) {
    throw new Error(data.error || 'Unknown error');
  }

  return data.data;
}

export async function getRecentAnime(page = 1): Promise<RecentResult> {
  const response = await fetch(`${API_URL}/api/v1/anime/recent?page=${page}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok) {
    throw new Error('Failed to fetch recent anime');
  }

  const data: APIResponse<RecentResult> = await response.json();
  
  if (!data.success || !data.data) {
    throw new Error(data.error || 'Unknown error');
  }

  return data.data;
}

export async function getPopularAnime(page = 1): Promise<AnimeSearchResult> {
  const response = await fetch(`${API_URL}/api/v1/anime/popular?page=${page}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok) {
    throw new Error('Failed to fetch popular anime');
  }

  const data: APIResponse<AnimeSearchResult> = await response.json();
  
  if (!data.success || !data.data) {
    throw new Error(data.error || 'Unknown error');
  }

  return data.data;
}

export async function getAnimeDetails(animeId: string): Promise<AnimeDetails> {
  const response = await fetch(`${API_URL}/api/v1/anime/${animeId}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok) {
    throw new Error('Failed to fetch anime details');
  }

  const data: APIResponse<AnimeDetails> = await response.json();
  
  if (!data.success || !data.data) {
    throw new Error(data.error || 'Unknown error');
  }

  return data.data;
}

export async function getAnimeEpisodes(animeId: string): Promise<AnimeEpisode[]> {
  const response = await fetch(`${API_URL}/api/v1/anime/${animeId}/episodes`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok) {
    throw new Error('Failed to fetch anime episodes');
  }

  const data: APIResponse<AnimeEpisode[]> = await response.json();
  
  if (!data.success || !data.data) {
    throw new Error(data.error || 'Unknown error');
  }

  return data.data;
}

export async function getAnimeSources(episodeId: string): Promise<StreamSource[]> {
  const response = await fetch(`${API_URL}/api/v1/anime/episode/${episodeId}/sources`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok) {
    throw new Error('Failed to fetch anime sources');
  }

  const data: APIResponse<StreamSource[]> = await response.json();
  
  if (!data.success || !data.data) {
    throw new Error(data.error || 'Unknown error');
  }

  return data.data;
}

export async function searchManga(params: SearchRequest): Promise<SearchResponse> {
  const response = await fetch(`${API_URL}/api/v1/search`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(params),
  });

  if (!response.ok) {
    throw new Error('Failed to search manga');
  }

  const data: APIResponse<SearchResponse> = await response.json();
  
  if (!data.success || !data.data) {
    throw new Error(data.error || 'Unknown error');
  }

  return data.data;
}

export function getCoverUrl(mangaId: string, fileName: string, size: 256 | 512 | 1024 = 256): string {
  if (!fileName) return '/placeholder-cover.svg';
  return `https://uploads.mangadex.org/covers/${mangaId}/${fileName}.${size}.jpg`;
}

export async function getChapterPages(chapterId: string): Promise<ChapterPages> {
  const response = await fetch(`${API_URL}/api/v1/chapter/${chapterId}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok) {
    throw new Error('Failed to fetch chapter pages');
  }

  const data: APIResponse<ChapterPages> = await response.json();
  
  if (!data.success || !data.data) {
    throw new Error(data.error || 'Unknown error');
  }

  return data.data;
}

export async function saveProgress(progress: ProgressRequest): Promise<void> {
  const response = await fetch(`${API_URL}/api/v1/progress`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(progress),
  });

  if (!response.ok) {
    throw new Error('Failed to save progress');
  }

  const data: APIResponse<void> = await response.json();
  
  if (!data.success) {
    throw new Error(data.error || 'Failed to save progress');
  }
}

export async function getMangaDetails(mangaId: string): Promise<MangaDetails> {
  const response = await fetch(`${API_URL}/api/v1/manga/${mangaId}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok) {
    throw new Error('Failed to fetch manga details');
  }

  const data: APIResponse<MangaDetails> = await response.json();
  
  if (!data.success || !data.data) {
    throw new Error(data.error || 'Unknown error');
  }

  return data.data;
}
