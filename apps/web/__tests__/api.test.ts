import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

describe('API Integration', () => {
  const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

  beforeEach(() => {
    vi.stubEnv('NEXT_PUBLIC_API_URL', 'http://localhost:8080');
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  describe('searchManga', () => {
    it('should call the search endpoint with correct parameters', async () => {
      const mockResponse = {
        success: true,
        data: {
          results: [
            { id: 'manga-1', title: { en: 'One Piece' } },
            { id: 'manga-2', title: { en: 'Naruto' } },
          ],
          total: 2,
          offset: 0,
          limit: 20,
        },
      };

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      const { searchManga } = await import('../lib/api');
      const result = await searchManga({ title: 'One Piece' });

      expect(fetch).toHaveBeenCalledWith(
        `${API_URL}/api/v1/search`,
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ title: 'One Piece' }),
        })
      );

      expect(result).toEqual(mockResponse.data);
    });

    it('should throw error when search fails', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
      });

      const { searchManga } = await import('../lib/api');

      await expect(searchManga({ title: 'test' })).rejects.toThrow('Failed to search manga');
    });

    it('should throw error when API returns success: false', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: false, error: 'Invalid query' }),
      });

      const { searchManga } = await import('../lib/api');

      await expect(searchManga({ title: 'test' })).rejects.toThrow('Invalid query');
    });
  });

  describe('getMangaDetails', () => {
    it('should fetch manga details by ID', async () => {
      const mockDetails = {
        id: 'manga-123',
        title: { en: 'Test Manga' },
        description: { en: 'A test manga' },
        status: 'ongoing',
        chapters: [],
      };

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true, data: mockDetails }),
      });

      const { getMangaDetails } = await import('../lib/api');
      const result = await getMangaDetails('manga-123');

      expect(fetch).toHaveBeenCalledWith(
        `${API_URL}/api/v1/manga/manga-123`,
        expect.objectContaining({ method: 'GET' })
      );

      expect(result).toEqual(mockDetails);
    });

    it('should throw error for empty manga ID', async () => {
      const { getMangaDetails } = await import('../lib/api');

      await expect(getMangaDetails('')).rejects.toThrow();
    });
  });

  describe('saveProgress', () => {
    it('should save reading progress successfully', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true }),
      });

      const { saveProgress } = await import('../lib/api');
      const progress = {
        user_id: 'user-123',
        manga_id: 'manga-456',
        chapter_id: 'chapter-789',
        page_number: 15,
      };

      await expect(saveProgress(progress)).resolves.not.toThrow();

      expect(fetch).toHaveBeenCalledWith(
        `${API_URL}/api/v1/progress`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify(progress),
        })
      );
    });

    it('should handle progress save failure', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
      });

      const { saveProgress } = await import('../lib/api');

      await expect(
        saveProgress({
          user_id: 'user-123',
          manga_id: 'manga-456',
          chapter_id: 'chapter-789',
          page_number: 15,
        })
      ).rejects.toThrow('Failed to save progress');
    });
  });

  describe('getCoverUrl', () => {
    it('should generate correct cover URL', () => {
      const { getCoverUrl } = require('../lib/api');

      expect(getCoverUrl('manga-id', 'cover.jpg')).toBe(
        'https://uploads.mangadex.org/covers/manga-id/cover.jpg.256.jpg'
      );

      expect(getCoverUrl('manga-id', 'cover.jpg', 512)).toBe(
        'https://uploads.mangadex.org/covers/manga-id/cover.jpg.512.jpg'
      );

      expect(getCoverUrl('manga-id', '', 256)).toBe('/placeholder-cover.svg');
    });
  });
});
