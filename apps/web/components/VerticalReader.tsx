'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
import Image from 'next/image';
import { saveProgress } from '@/lib/api';

interface VerticalReaderProps {
  imageUrls: string[];
  userId: string;
  mangaId: string;
  chapterId: string;
  initialPage?: number;
}

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

function SkeletonImage({ index }: { index: number }) {
  return (
    <div
      className="w-full animate-pulse bg-zinc-200 dark:bg-zinc-800"
      style={{ aspectRatio: '3/4' }}
    >
      <div className="flex items-center justify-center h-full text-zinc-400">
        Loading page {index + 1}...
      </div>
    </div>
  );
}

export default function VerticalReader({
  imageUrls,
  userId,
  mangaId,
  chapterId,
  initialPage = 0,
}: VerticalReaderProps) {
  const [currentPage, setCurrentPage] = useState(initialPage);
  const [loadingStates, setLoadingStates] = useState<boolean[]>(
    new Array(imageUrls.length).fill(true)
  );
  const [savingProgress, setSavingProgress] = useState(false);
  const [imageDimensions, setImageDimensions] = useState<Map<number, { width: number; height: number }>>(new Map());
  const observerRef = useRef<IntersectionObserver | null>(null);
  const containerRefs = useRef<Map<number, HTMLDivElement>>(new Map());

  const debouncedPage = useDebounce(currentPage, 2000);

  const handleImageLoad = useCallback((index: number) => {
    setLoadingStates((prev) => {
      const newStates = [...prev];
      newStates[index] = false;
      return newStates;
    });
  }, []);

  const handleImageError = useCallback((index: number) => {
    setLoadingStates((prev) => {
      const newStates = [...prev];
      newStates[index] = false;
      return newStates;
    });
  }, []);

  // Handle image load complete to get dimensions for proper aspect ratio
  const handleImageComplete = useCallback((index: number, img: HTMLImageElement) => {
    setImageDimensions((prev) => {
      const newMap = new Map(prev);
      newMap.set(index, { width: img.naturalWidth, height: img.naturalHeight });
      return newMap;
    });
    handleImageLoad(index);
  }, [handleImageLoad]);

  useEffect(() => {
    if (debouncedPage === currentPage || !userId) {
      return;
    }

    setSavingProgress(true);
    saveProgress({
      user_id: userId,
      manga_id: mangaId,
      chapter_id: chapterId,
      page_number: currentPage,
    })
      .catch((err) => {
        console.error('Failed to save progress:', err);
      })
      .finally(() => {
        setSavingProgress(false);
      });
  }, [debouncedPage, currentPage, userId, mangaId, chapterId]);

  useEffect(() => {
    const options = {
      root: null,
      rootMargin: '0px',
      threshold: 0.5,
    };

    observerRef.current = new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          const index = Number(entry.target.getAttribute('data-page-index'));
          setCurrentPage(index);
        }
      });
    }, options);

    containerRefs.current.forEach((el) => {
      if (el) {
        observerRef.current?.observe(el);
      }
    });

    return () => {
      observerRef.current?.disconnect();
    };
  }, [imageUrls]);

  return (
    <div className="relative w-full max-w-2xl mx-auto">
      {savingProgress && (
        <div className="fixed top-4 right-4 z-50 px-3 py-1 bg-blue-600 text-white text-sm rounded-full">
          Saving...
        </div>
      )}

      <div className="flex flex-col gap-0">
        {imageUrls.map((url, index) => {
          const dims = imageDimensions.get(index);
          const aspectRatio = dims ? dims.width / dims.height : 3/4;
          
          return (
            <div
              key={`${chapterId}-${index}`}
              data-page-index={index}
              ref={(el) => {
                if (el) {
                  containerRefs.current.set(index, el);
                }
              }}
              className="w-full relative"
              style={{ aspectRatio: `${aspectRatio}` }}
            >
              {loadingStates[index] && <SkeletonImage index={index} />}
              <Image
                src={url}
                alt={`Page ${index + 1}`}
                fill
                sizes="(max-width: 768px) 100vw, 672px"
                priority={index < 3} // Priority loading for first 3 images
                loading={index < 3 ? 'eager' : 'lazy'}
                quality={90}
                onLoadingComplete={(img) => handleImageComplete(index, img)}
                onError={() => handleImageError(index)}
                className={`object-contain transition-opacity duration-300 ${
                  loadingStates[index] ? 'opacity-0' : 'opacity-100'
                }`}
              />
            </div>
          );
        })}
      </div>
    </div>
  );
}
