'use client';

import { useQuery } from '@tanstack/react-query';
import { useParams } from 'next/navigation';
import Image from 'next/image';
import Link from 'next/link';
import { getAnimeDetails, getAnimeEpisodes } from '@/lib/api';

export default function AnimeDetailsPage() {
  const params = useParams();
  const animeId = params.animeId as string;

  const { data: details, isLoading: detailsLoading } = useQuery({
    queryKey: ['anime', 'details', animeId],
    queryFn: () => getAnimeDetails(animeId),
    staleTime: 10 * 60 * 1000,
  });

  const { data: episodes, isLoading: episodesLoading } = useQuery({
    queryKey: ['anime', 'episodes', animeId],
    queryFn: () => getAnimeEpisodes(animeId),
    staleTime: 10 * 60 * 1000,
  });

  if (detailsLoading) {
    return (
      <div className="min-h-screen bg-gray-950 text-white p-4">
        <div className="max-w-7xl mx-auto animate-pulse">
          <div className="flex gap-6">
            <div className="w-64 aspect-[3/4] bg-gray-800 rounded"></div>
            <div className="flex-1">
              <div className="h-8 bg-gray-800 rounded w-48 mb-4"></div>
              <div className="h-4 bg-gray-800 rounded w-32 mb-2"></div>
              <div className="h-4 bg-gray-800 rounded w-full mb-2"></div>
              <div className="h-4 bg-gray-800 rounded w-3/4"></div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (!details) {
    return (
      <div className="min-h-screen bg-gray-950 text-white p-4">
        <div className="max-w-7xl mx-auto text-center">
          <p>Anime not found</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-950 text-white p-4">
      <div className="max-w-7xl mx-auto">
        <div className="flex gap-6 mb-8">
          <div className="w-48 md:w-64 shrink-0">
            <div className="relative aspect-[3/4] rounded-lg overflow-hidden">
              <Image
                src={details.image || '/placeholder.svg'}
                alt={details.title}
                fill
                className="object-cover"
              />
            </div>
          </div>
          
          <div className="flex-1">
            <h1 className="text-2xl md:text-3xl font-bold mb-2">{details.title}</h1>
            
            <div className="flex flex-wrap gap-2 mb-4">
              {details.genres.map((genre) => (
                <span key={genre} className="px-2 py-1 bg-gray-800 rounded text-sm">
                  {genre}
                </span>
              ))}
            </div>

            <div className="text-gray-400 text-sm mb-4">
              <p>Status: {details.status}</p>
              {details.released_year && <p>Year: {details.released_year}</p>}
              <p>Episodes: {details.total_episodes}</p>
            </div>

            <p className="text-gray-300 text-sm leading-relaxed line-clamp-6">
              {details.description}
            </p>
          </div>
        </div>

        <h2 className="text-xl font-bold mb-4">Episodes</h2>

        {episodesLoading ? (
          <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-2">
            {[...Array(12)].map((_, i) => (
              <div key={i} className="h-10 bg-gray-800 rounded animate-pulse"></div>
            ))}
          </div>
        ) : (
          <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 xl:grid-cols-8 gap-2">
            {episodes?.map((ep) => (
              <Link
                key={ep.id}
                href={`/watch/${ep.id}?animeId=${animeId}`}
                className="px-3 py-2 bg-gray-900 hover:bg-gray-800 rounded text-center transition-colors"
              >
                {ep.number}
              </Link>
            ))}
          </div>
        )}

        {(!episodes || episodes.length === 0) && (
          <p className="text-gray-500">No episodes available</p>
        )}
      </div>
    </div>
  );
}
