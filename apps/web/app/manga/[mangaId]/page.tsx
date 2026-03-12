'use client';

import { useQuery } from '@tanstack/react-query';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import { getMangaDetails, getCoverUrl } from '@/lib/api';
import { MangaDetails as MangaDetailsType } from '@/lib/types';

export default function MangaPage() {
  const params = useParams();
  const mangaId = params?.mangaId as string;

  const { data, isLoading, error, isError } = useQuery({
    queryKey: ['manga', mangaId],
    queryFn: () => getMangaDetails(mangaId),
    enabled: !!mangaId,
    staleTime: 10 * 60 * 1000,
  });

  if (!mangaId) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-zinc-50 dark:bg-black">
        <p className="text-zinc-900 dark:text-zinc-100">Invalid manga ID</p>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="min-h-screen bg-zinc-50 dark:bg-black flex items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <div className="w-8 h-8 border-2 border-zinc-600 border-t-transparent rounded-full animate-spin" />
          <p className="text-zinc-600 dark:text-zinc-400 text-sm">Loading manga...</p>
        </div>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="min-h-screen bg-zinc-50 dark:bg-black flex items-center justify-center">
        <div className="flex flex-col items-center gap-4 text-center px-4">
          <p className="text-red-500 text-lg">Failed to load manga</p>
          <p className="text-zinc-500 text-sm">
            {error instanceof Error ? error.message : 'Unknown error'}
          </p>
          <Link href="/" className="text-blue-500 hover:underline">
            Back to search
          </Link>
        </div>
      </div>
    );
  }

  if (!data) {
    return (
      <div className="min-h-screen bg-zinc-50 dark:bg-black flex items-center justify-center">
        <div className="flex flex-col items-center gap-4 text-center px-4">
          <p className="text-zinc-900 dark:text-zinc-100 text-lg">Manga not found</p>
          <Link href="/" className="text-blue-500 hover:underline">
            Back to search
          </Link>
        </div>
      </div>
    );
  }

  const { manga, chapters } = data;

  return (
    <div className="min-h-screen bg-zinc-50 dark:bg-black">
      <div className="max-w-4xl mx-auto p-4">
        <Link href="/" className="inline-flex items-center gap-2 text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100 mb-6">
          ← Back to search
        </Link>

        <div className="flex flex-col md:flex-row gap-6 mb-8">
          <div className="w-48 flex-shrink-0">
            <img
              src={getCoverUrl(manga.id, manga.cover_filename, 512)}
              alt={manga.title}
              className="w-full rounded-lg shadow-lg"
            />
          </div>
          <div className="flex-1">
            <h1 className="text-2xl font-bold text-zinc-900 dark:text-zinc-100 mb-2">
              {manga.title}
            </h1>
            {manga.description && (
              <p className="text-zinc-600 dark:text-zinc-400 text-sm leading-relaxed">
                {manga.description}
              </p>
            )}
          </div>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100 mb-4">
            Chapters ({chapters.length})
          </h2>

          {chapters.length === 0 ? (
            <p className="text-zinc-500">No chapters available</p>
          ) : (
            <div className="space-y-2">
              {chapters.map((chapter) => (
                <Link
                  key={chapter.id}
                  href={`/read/${chapter.id}?mangaId=${manga.id}`}
                  className="flex items-center justify-between p-3 bg-white dark:bg-zinc-900 rounded-lg hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
                >
                  <div className="flex items-center gap-3">
                    <span className="text-zinc-900 dark:text-zinc-100 font-medium">
                      {chapter.chapter ? `Ch. ${chapter.chapter}` : 'One-shot'}
                    </span>
                    {chapter.title && (
                      <span className="text-zinc-500 dark:text-zinc-400 text-sm">
                        {chapter.title}
                      </span>
                    )}
                  </div>
                  <span className="text-zinc-400 text-sm">
                    {new Date(chapter.published).toLocaleDateString()}
                  </span>
                </Link>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
