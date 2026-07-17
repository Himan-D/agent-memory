"use client";

import { useMemo, useRef, useState, useEffect, type ReactNode } from "react";

interface VirtualListProps<T> {
  items: T[];
  estimateSize?: number;
  height?: number;
  overscan?: number;
  className?: string;
  getKey: (item: T, index: number) => string | number;
  renderItem: (item: T, index: number) => ReactNode;
}

/**
 * Lightweight fixed-row virtual list (no extra deps).
 * Suitable for large tables (memories, entities) when each row is ~constant height.
 */
export function VirtualList<T>({
  items,
  estimateSize = 72,
  height = 480,
  overscan = 6,
  className,
  getKey,
  renderItem,
}: VirtualListProps<T>) {
  const scrollerRef = useRef<HTMLDivElement>(null);
  const [scrollTop, setScrollTop] = useState(0);

  useEffect(() => {
    const el = scrollerRef.current;
    if (!el) return;
    const onScroll = () => setScrollTop(el.scrollTop);
    el.addEventListener("scroll", onScroll, { passive: true });
    return () => el.removeEventListener("scroll", onScroll);
  }, []);

  const totalHeight = items.length * estimateSize;
  const start = Math.max(0, Math.floor(scrollTop / estimateSize) - overscan);
  const visibleCount = Math.ceil(height / estimateSize) + overscan * 2;
  const end = Math.min(items.length, start + visibleCount);

  const slice = useMemo(() => items.slice(start, end), [items, start, end]);

  return (
    <div
      ref={scrollerRef}
      className={className}
      style={{ height, overflow: "auto", position: "relative" }}
    >
      <div style={{ height: totalHeight, position: "relative" }}>
        <div
          style={{
            position: "absolute",
            top: 0,
            left: 0,
            right: 0,
            transform: `translateY(${start * estimateSize}px)`,
          }}
        >
          {slice.map((item, i) => {
            const index = start + i;
            return (
              <div
                key={getKey(item, index)}
                style={{ minHeight: estimateSize }}
              >
                {renderItem(item, index)}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
