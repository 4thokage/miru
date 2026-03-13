import { APIResponse, SearchRequest, SearchResponse, ChapterPages, ProgressRequest, MangaDetails } from './types';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

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
