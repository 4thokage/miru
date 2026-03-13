'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import Link from 'next/link';
import Image from 'next/image';
import { searchAnime, getRecentAnime, getPopularAnime } from '@/lib/api';
import { Anime } from '@/lib/types';

type Tab = 'recent' | 'popular' | 'search';

export default function AnimePage() {
  const [tab, setTab] = useState<Tab>('recent');
  const [searchQuery, setSearchQuery] = useState('');
  const [page, setPage] = useState(1);

  const { data: recentData, isLoading: recentLoading } = useQuery({
    queryKey: ['anime', 'recent', page],
    queryFn: () => getRecentAnime(page),
    enabled: tab === 'recent',
    staleTime: 5 * 60 * 1000,
  });

  const { data: popularData, isLoading: popularLoading } = useQuery({
    queryKey: ['anime', 'popular', page],
    queryFn: () => getPopularAnime(page),
    enabled: tab === 'popular',
    staleTime: 5 * 60 * 1000,
  });

  const { data: searchData, isLoading: searchLoading, refetch: doSearch } = useQuery({
    queryKey: ['anime', 'search', searchQuery, page],
    queryFn: () => searchAnime(searchQuery, page),
    enabled: false,
    staleTime: 5 * 60 * 1000,
  });

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (searchQuery.trim()) {
      setTab('search');
      setPage(1);
      doSearch();
    }
  };

  const animes: Anime[] = tab === 'recent' 
    ? recentData?.episodes?.map(ep => ({ id: ep.anime_id, title: ep.anime_title, image: ep.image })) || []
    : tab === 'popular'
    ? popularData?.animes || []
    : searchData?.animes || [];

  const isLoading = tab === 'recent' ? recentLoading : tab === 'popular' ? popularLoading : searchLoading;
  const hasNext = tab === 'recent' ? recentData?.has_next : tab === 'popular' ? popularData?.has_next : searchData?.has_next;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-4">
      <div className="max-w-7xl mx-auto">
        <h1 className="text-3xl font-bold mb-6">Anime</h1>

        <div className="flex gap-4 mb-6">
          <button
            onClick={() => { setTab('recent'); setPage(1); }}
            className={`px-4 py-2 rounded ${tab === 'recent' ? 'bg-blue-600' : 'bg-gray-800'}`}
          >
            Recent
          </button>
          <button
            onClick={() => { setTab('popular'); setPage(1); }}
            className={`px-4 py-2 rounded ${tab === 'popular' ? 'bg-blue-600' : 'bg-gray-800'}`}
          >
            Popular
          </button>
        </div>

        <form onSubmit={handleSearch} className="mb-6 flex gap-2">
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search anime..."
            className="flex-1 px-4 py-2 bg-gray-900 border border-gray-800 rounded text-white"
          />
          <button type="submit" className="px-6 py-2 bg-blue-600 rounded hover:bg-blue-700">
            Search
          </button>
        </form>

        {isLoading ? (
          <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
            {[...Array(12)].map((_, i) => (
              <div key={i} className="animate-pulse">
                <div className="aspect-[3/4] bg-gray-800 rounded mb-2"></div>
                <div className="h-4 bg-gray-800 rounded w-3/4"></div>
              </div>
            ))}
          </div>
        ) : (
          <>
            <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
              {animes.map((anime) => (
                <Link key={anime.id} href={`/anime/${anime.id}`} className="group">
                  <div className="relative aspect-[3/4] rounded-lg overflow-hidden mb-2">
                    <Image
                      src={anime.image || '/placeholder.svg'}
                      alt={anime.title}
                      fill
                      className="object-cover group-hover:scale-105 transition-transform"
                    />
                  </div>
                  <h3 className="text-sm font-medium truncate">{anime.title}</h3>
                </Link>
              ))}
            </div>

            {animes.length === 0 && (
              <p className="text-center text-gray-500 py-8">No anime found</p>
            )}

            {hasNext && (
              <div className="flex justify-center mt-6">
                <button
                  onClick={() => setPage(p => p + 1)}
                  className="px-6 py-2 bg-gray-800 rounded hover:bg-gray-700"
                >
                  Load More
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
