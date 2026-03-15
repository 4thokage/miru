'use client';

import { useQuery } from '@tanstack/react-query';
import { useSearchParams, useRouter, useParams } from 'next/navigation';
import { useState } from 'react';
import VideoPlayer from '@/components/VideoPlayer';
import { getAnimeSources, getAnimeDetails, getAnimeEpisodes } from '@/lib/api';

export default function WatchPage() {
  const params = useParams();
  const searchParams = useSearchParams();
  const router = useRouter();
  const episodeId = params.episodeId as string;
  const animeId = searchParams.get('animeId') as string;

  const [currentServer, setCurrentServer] = useState(0);

  const { data: sources, isLoading, error } = useQuery({
    queryKey: ['anime', 'sources', episodeId],
    queryFn: () => getAnimeSources(episodeId),
    staleTime: 2 * 60 * 1000,
  });

  const { data: animeDetails } = useQuery({
    queryKey: ['anime', 'details', animeId],
    queryFn: () => getAnimeDetails(animeId),
    enabled: !!animeId,
    staleTime: 10 * 60 * 1000,
  });

  const { data: episodes } = useQuery({
    queryKey: ['anime', 'episodes', animeId],
    queryFn: () => getAnimeEpisodes(animeId),
    enabled: !!animeId,
    staleTime: 10 * 60 * 1000,
  });

  const currentEpisode = episodes?.find(ep => ep.id === episodeId);
  const currentSource = sources?.[currentServer];

  const goToEpisode = (epId: string) => {
    router.push(`/watch/${epId}?animeId=${animeId}`);
  };

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gray-950 flex items-center justify-center">
        <div className="animate-spin w-8 h-8 border-2 border-blue-500 border-t-transparent rounded-full"></div>
      </div>
    );
  }

  if (error || !sources || sources.length === 0) {
    return (
      <div className="min-h-screen bg-gray-950 text-white p-4">
        <div className="max-w-4xl mx-auto text-center">
          <p className="text-red-500 mb-4">Failed to load video sources</p>
          <button
            onClick={() => router.back()}
            className="px-4 py-2 bg-gray-800 rounded hover:bg-gray-700"
          >
            Go Back
          </button>
        </div>
      </div>
    );
  }

  const isM3U8 = currentSource?.is_m3u8 || 
    (currentSource?.url && currentSource.url.includes('.m3u8'));

  return (
    <div className="min-h-screen bg-gray-950 text-white">
      <div className="relative aspect-video bg-black">
        {currentSource?.url && isM3U8 ? (
          <VideoPlayer src={currentSource.url} />
        ) : currentSource?.url ? (
          <iframe
            src={currentSource.url}
            className="w-full h-full"
            allowFullScreen
            allow="autoplay; fullscreen"
          />
        ) : (
          <div className="flex items-center justify-center h-full">
            <p className="text-gray-500">No video source available</p>
          </div>
        )}
      </div>

      <div className="p-4">
        <div className="max-w-4xl mx-auto">
          <h1 className="text-xl font-bold mb-2">
            {animeDetails?.title} - Episode {currentEpisode?.number}
          </h1>

          {sources.length > 1 && (
            <div className="mb-4">
              <label className="text-sm text-gray-400 mr-2">Server:</label>
              <select
                value={currentServer}
                onChange={(e) => setCurrentServer(parseInt(e.target.value))}
                className="bg-gray-800 px-3 py-1 rounded"
              >
                {sources.map((source, i) => (
                  <option key={i} value={i}>
                    {source.server} ({source.quality})
                  </option>
                ))}
              </select>
            </div>
          )}

          {episodes && episodes.length > 1 && (
            <div>
              <h2 className="text-lg font-semibold mb-2">Episodes</h2>
              <div className="flex flex-wrap gap-2">
                {episodes.map((ep) => (
                  <button
                    key={ep.id}
                    onClick={() => goToEpisode(ep.id)}
                    className={`px-3 py-1 rounded text-sm ${
                      ep.id === episodeId
                        ? 'bg-blue-600'
                        : 'bg-gray-800 hover:bg-gray-700'
                    }`}
                  >
                    {ep.number}
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
