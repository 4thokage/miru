'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
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
  const observerRef = useRef<IntersectionObserver | null>(null);
  const imageRefs = useRef<Map<number, HTMLImageElement>>(new Map());

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

    imageRefs.current.forEach((img) => {
      if (img) {
        observerRef.current?.observe(img);
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
        {imageUrls.map((url, index) => (
          <div
            key={`${chapterId}-${index}`}
            className="w-full relative"
            ref={(el) => {
              if (el) {
                const img = el.querySelector('img');
                if (img) {
                  imageRefs.current.set(index, img);
                }
              }
            }}
          >
            {loadingStates[index] && <SkeletonImage index={index} />}
            <img
              data-page-index={index}
              src={url}
              alt={`Page ${index + 1}`}
              loading="lazy"
              onLoad={() => handleImageLoad(index)}
              onError={() => handleImageError(index)}
              className={`w-full h-auto transition-opacity duration-300 ${
                loadingStates[index] ? 'opacity-0' : 'opacity-100'
              }`}
            />
          </div>
        ))}
      </div>
    </div>
  );
}
