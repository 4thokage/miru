'use client';

import { useQuery } from '@tanstack/react-query';
import { useParams, useSearchParams } from 'next/navigation';
import { getChapterPages } from '@/lib/api';
import { useDeviceId } from '@/lib/useDeviceId';
import VerticalReader from '@/components/VerticalReader';

export default function ReadPage() {
  const params = useParams();
  const searchParams = useSearchParams();
  const chapterId = params?.chapterId as string;
  const mangaId = searchParams?.get('mangaId') || '';
  const deviceId = useDeviceId();

  const { data, isLoading, error, isError } = useQuery({
    queryKey: ['chapter', chapterId],
    queryFn: () => getChapterPages(chapterId),
    enabled: !!chapterId,
    staleTime: 10 * 60 * 1000,
  });

  if (!chapterId) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-black">
        <p className="text-white">Invalid chapter ID</p>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="min-h-screen bg-black flex items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <div className="w-8 h-8 border-2 border-white border-t-transparent rounded-full animate-spin" />
          <p className="text-white text-sm">Loading chapter...</p>
        </div>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="min-h-screen bg-black flex items-center justify-center">
        <div className="flex flex-col items-center gap-4 text-center px-4">
          <p className="text-red-500 text-lg">Failed to load chapter</p>
          <p className="text-zinc-400 text-sm">
            {error instanceof Error ? error.message : 'Unknown error'}
          </p>
        </div>
      </div>
    );
  }

  if (!data || data.images.length === 0) {
    return (
      <div className="min-h-screen bg-black flex items-center justify-center">
        <div className="flex flex-col items-center gap-4 text-center px-4">
          <p className="text-white text-lg">No pages found</p>
          <p className="text-zinc-400 text-sm">This chapter may not be available</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-black">
      <VerticalReader
        imageUrls={data.images}
        userId={deviceId}
        mangaId={mangaId}
        chapterId={chapterId}
      />
    </div>
  );
}
