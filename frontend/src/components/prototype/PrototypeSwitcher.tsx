'use client';

import React, { useEffect } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { ChevronLeft, ChevronRight } from 'lucide-react';

export interface PrototypeVariantMeta {
  key: string;
  label: string;
}

interface PrototypeSwitcherProps {
  variants: PrototypeVariantMeta[];
  current: string;
}

export function PrototypeSwitcher({ variants, current }: PrototypeSwitcherProps) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const currentIdx = Math.max(
    0,
    variants.findIndex((v) => v.key === current)
  );
  const currentMeta = variants[currentIdx] ?? variants[0];

  const goTo = (nextIdx: number) => {
    const wrapped = (nextIdx + variants.length) % variants.length;
    const nextKey = variants[wrapped].key;
    const params = new URLSearchParams(searchParams.toString());
    params.set('variant', nextKey);
    router.replace(`${pathname}?${params.toString()}`, { scroll: false });
  };

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement | null)?.tagName?.toLowerCase();
      const isEditable =
        tag === 'input' ||
        tag === 'textarea' ||
        tag === 'select' ||
        (e.target as HTMLElement | null)?.isContentEditable;
      if (isEditable) return;

      if (e.key === 'ArrowLeft') {
        e.preventDefault();
        goTo(currentIdx - 1);
      } else if (e.key === 'ArrowRight') {
        e.preventDefault();
        goTo(currentIdx + 1);
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  });

  if (process.env.NODE_ENV === 'production') {
    return null;
  }

  return (
    <div className="fixed bottom-4 left-1/2 z-50 -translate-x-1/2 flex items-center gap-2 rounded-full border border-amber-400/50 bg-stone-950/95 px-3 py-1.5 text-xs text-amber-100 shadow-2xl backdrop-blur">
      <button
        type="button"
        onClick={() => goTo(currentIdx - 1)}
        className="rounded-full p-1 hover:bg-amber-500/20 text-amber-300 transition"
        title="Biến thể trước (←)"
      >
        <ChevronLeft className="h-4 w-4" />
      </button>
      <span className="px-2 font-mono font-semibold tracking-wide">
        PROTOTYPE · {currentMeta.key} ({currentMeta.label})
      </span>
      <button
        type="button"
        onClick={() => goTo(currentIdx + 1)}
        className="rounded-full p-1 hover:bg-amber-500/20 text-amber-300 transition"
        title="Biến thể tiếp (→)"
      >
        <ChevronRight className="h-4 w-4" />
      </button>
    </div>
  );
}
