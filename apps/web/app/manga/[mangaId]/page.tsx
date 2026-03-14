'use client';

import { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import Image from 'next/image';
import { getMangaDetails, getCoverUrl } from '@/lib/api';
import { MangaDetails as MangaDetailsType, Chapter } from '@/lib/types';

const CHAPTERS_PER_PAGE = 20;

export default function MangaPage() {
  const params = useParams();
  const mangaId = params?.mangaId as string;
  const [selectedLanguage, setSelectedLanguage] = useState<string>('en');
  const [currentPage, setCurrentPage] = useState(1);

  const { data, isLoading, error, isError } = useQuery({
    queryKey: ['manga', mangaId],
    queryFn: () => getMangaDetails(mangaId),
    enabled: !!mangaId,
    staleTime: 10 * 60 * 1000,
  });

  // Get unique languages from chapters
  const availableLanguages = useMemo(() => {
    if (!data?.chapters) return [];
    const languages = new Set(data.chapters.map((ch: Chapter) => ch.language || 'en'));
    return Array.from(languages).sort();
  }, [data?.chapters]);

  // Filter chapters by selected language
  const filteredChapters = useMemo(() => {
    if (!data?.chapters) return [];
    return data.chapters.filter((ch: Chapter) => (ch.language || 'en') === selectedLanguage);
  }, [data?.chapters, selectedLanguage]);

  // Paginate chapters
  const totalPages = Math.ceil(filteredChapters.length / CHAPTERS_PER_PAGE);
  const paginatedChapters = useMemo(() => {
    const start = (currentPage - 1) * CHAPTERS_PER_PAGE;
    return filteredChapters.slice(start, start + CHAPTERS_PER_PAGE);
  }, [filteredChapters, currentPage]);

  // Reset to page 1 when language changes
  const handleLanguageChange = (lang: string) => {
    setSelectedLanguage(lang);
    setCurrentPage(1);
  };

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

  const { manga } = data;
  const hasMultipleLanguages = availableLanguages.length > 1;

  return (
    <div className="min-h-screen bg-zinc-50 dark:bg-black">
      <div className="max-w-4xl mx-auto p-4">
        <Link href="/" className="inline-flex items-center gap-2 text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100 mb-6">
          ← Back to search
        </Link>

        <div className="flex flex-col md:flex-row gap-6 mb-8">
          <div className="w-48 flex-shrink-0 relative aspect-[3/4]">
            <Image
              src={getCoverUrl(manga.id, manga.cover_filename, 512)}
              alt={manga.title}
              fill
              sizes="12rem"
              className="object-cover rounded-lg shadow-lg"
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
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-4">
            <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
              Chapters ({filteredChapters.length})
            </h2>
            
            {hasMultipleLanguages && (
              <div className="flex items-center gap-2">
                <label htmlFor="language-filter" className="text-sm text-zinc-600 dark:text-zinc-400">
                  Language:
                </label>
                <select
                  id="language-filter"
                  value={selectedLanguage}
                  onChange={(e) => handleLanguageChange(e.target.value)}
                  className="px-3 py-1.5 text-sm bg-white dark:bg-zinc-800 border border-zinc-300 dark:border-zinc-700 rounded-md text-zinc-900 dark:text-zinc-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  {availableLanguages.map((lang) => (
                    <option key={lang} value={lang}>
                      {lang === 'en' ? 'English' : lang}
                    </option>
                  ))}
                </select>
              </div>
            )}
          </div>

          {filteredChapters.length === 0 ? (
            <p className="text-zinc-500">No chapters available in {selectedLanguage === 'en' ? 'English' : selectedLanguage}</p>
          ) : (
            <>
              <div className="space-y-2">
                {paginatedChapters.map((chapter) => (
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

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="flex items-center justify-center gap-2 mt-6">
                  <button
                    onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                    disabled={currentPage === 1}
                    className="px-3 py-1.5 text-sm bg-white dark:bg-zinc-800 border border-zinc-300 dark:border-zinc-700 rounded-md text-zinc-900 dark:text-zinc-100 disabled:opacity-50 disabled:cursor-not-allowed hover:bg-zinc-100 dark:hover:bg-zinc-700 transition-colors"
                  >
                    Previous
                  </button>
                  
                  <div className="flex items-center gap-1">
                    {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                      // Show pages around current page
                      let pageNum;
                      if (totalPages <= 5) {
                        pageNum = i + 1;
                      } else if (currentPage <= 3) {
                        pageNum = i + 1;
                      } else if (currentPage >= totalPages - 2) {
                        pageNum = totalPages - 4 + i;
                      } else {
                        pageNum = currentPage - 2 + i;
                      }
                      
                      return (
                        <button
                          key={pageNum}
                          onClick={() => setCurrentPage(pageNum)}
                          className={`w-8 h-8 text-sm rounded-md transition-colors ${
                            currentPage === pageNum
                              ? 'bg-blue-600 text-white'
                              : 'bg-white dark:bg-zinc-800 border border-zinc-300 dark:border-zinc-700 text-zinc-900 dark:text-zinc-100 hover:bg-zinc-100 dark:hover:bg-zinc-700'
                          }`}
                        >
                          {pageNum}
                        </button>
                      );
                    })}
                  </div>
                  
                  <button
                    onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                    disabled={currentPage === totalPages}
                    className="px-3 py-1.5 text-sm bg-white dark:bg-zinc-800 border border-zinc-300 dark:border-zinc-700 rounded-md text-zinc-900 dark:text-zinc-100 disabled:opacity-50 disabled:cursor-not-allowed hover:bg-zinc-100 dark:hover:bg-zinc-700 transition-colors"
                  >
                    Next
                  </button>
                </div>
              )}

              <p className="text-center text-sm text-zinc-500 mt-2">
                Page {currentPage} of {totalPages}
              </p>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
