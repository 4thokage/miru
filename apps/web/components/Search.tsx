'use client';

import { useState, useEffect } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';
import Link from 'next/link';
import Image from 'next/image';
import { searchManga, getCoverUrl } from '@/lib/api';
import { SearchRequest, SearchResponse, Manga } from '@/lib/types';

function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);

  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => {
      clearTimeout(handler);
    };
  }, [value, delay]);

  return debouncedValue;
}

const PAGE_SIZE = 20;

export default function Search() {
  const [searchTerm, setSearchTerm] = useState('');
  const [order, setOrder] = useState('');
  const debouncedSearch = useDebounce(searchTerm, 300);

  const query: SearchRequest = {};
  
  if (debouncedSearch) {
    query.title = debouncedSearch;
  }
  
  if (order) {
    const orderObj: Record<string, string> = {};
    if (order === 'rating-desc') orderObj.rating = 'desc';
    else if (order === 'follows-desc') orderObj.follows = 'desc';
    else if (order === 'latest') orderObj.lastChapter = 'desc';
    else if (order === 'title-asc') orderObj.title = 'asc';
    query.order = orderObj;
  }

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading,
    error,
    refetch,
  } = useInfiniteQuery({
    queryKey: ['search', query],
    queryFn: ({ pageParam = 0 }) => searchManga({ ...query, limit: PAGE_SIZE, offset: pageParam }),
    getNextPageParam: (lastPage: SearchResponse) => {
      const { pagination } = lastPage;
      const nextOffset = pagination.offset + pagination.limit;
      if (nextOffset >= pagination.total) {
        return undefined;
      }
      return nextOffset;
    },
    initialPageParam: 0,
    enabled: debouncedSearch.length > 0,
    staleTime: 5 * 60 * 1000,
  });

  useEffect(() => {
    refetch();
  }, [debouncedSearch, order]);

  const allManga = data?.pages.flatMap(page => page.data) ?? [];
  const totalResults = data?.pages[0]?.pagination.total ?? 0;

  return (
    <div className="w-full max-w-4xl mx-auto p-4">
      <div className="flex flex-col sm:flex-row gap-3 mb-6">
        <input
          type="text"
          placeholder="Search manga..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="flex-1 px-4 py-3 rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-900 text-zinc-900 dark:text-zinc-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        <select
          value={order}
          onChange={(e) => setOrder(e.target.value)}
          className="px-4 py-3 rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-900 text-zinc-900 dark:text-zinc-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="">Sort by</option>
          <option value="rating-desc">Top Rated</option>
          <option value="follows-desc">Most Followed</option>
          <option value="latest">Latest</option>
          <option value="title-asc">Title A-Z</option>
        </select>
      </div>

      {isLoading && (
        <div className="text-center py-12 text-zinc-500">Loading...</div>
      )}

      {error && (
        <div className="text-center py-12 text-red-500">
          Error searching manga
        </div>
      )}

      {data && !isLoading && (
        <>
          {totalResults > 0 && (
            <p className="text-sm text-zinc-500 mb-4">
              Found {totalResults} results
            </p>
          )}

          {allManga.length === 0 ? (
            <div className="text-center py-12 text-zinc-500">
              No manga found
            </div>
          ) : (
            <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
              {allManga.map((manga: Manga) => (
                <MangaCard key={manga.id} manga={manga} />
              ))}
            </div>
          )}

          {hasNextPage && (
            <div className="flex justify-center mt-6">
              <button
                onClick={() => fetchNextPage()}
                disabled={isFetchingNextPage}
                className="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isFetchingNextPage ? 'Loading...' : 'Load More'}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}

function MangaCard({ manga }: { manga: Manga }) {
  return (
    <Link href={`/manga/${manga.id}`} className="block">
      <div className="group relative bg-zinc-100 dark:bg-zinc-900 rounded-lg overflow-hidden transition-transform hover:scale-105">
        <div className="aspect-[3/4] relative">
          <Image
            src={getCoverUrl(manga.id, manga.cover_filename)}
            alt={manga.title}
            fill
            sizes="(max-width: 768px) 50vw, 200px"
            className="object-cover"
          />
        </div>
        <div className="p-2">
          <h3 className="text-sm font-medium text-zinc-900 dark:text-zinc-100 line-clamp-2">
            {manga.title}
          </h3>
          {manga.last_chapter && (
            <p className="text-xs text-zinc-500 mt-1">
              Ch. {manga.last_chapter}
            </p>
          )}
        </div>
      </div>
    </Link>
  );
}
