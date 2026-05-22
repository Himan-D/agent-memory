"use client";

import { useRef, useEffect, type ReactNode } from 'react';

type WebComponentProps = Record<string, unknown>;

interface WebComponentWrapperProps {
  tag: string;
  props?: WebComponentProps;
  children?: ReactNode;
  className?: string;
  style?: React.CSSProperties;
  onEvent?: Record<string, (e: CustomEvent) => void>;
}

export function WebComponent({
  tag,
  props = {},
  children,
  className,
  style,
  onEvent,
}: WebComponentWrapperProps) {
  const ref = useRef<HTMLElement>(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    for (const [key, value] of Object.entries(props)) {
      if (typeof value === 'object' || typeof value === 'boolean') {
        (el as any)[key] = value;
      } else {
        el.setAttribute(key, String(value));
      }
    }

    if (onEvent) {
      for (const [eventName, handler] of Object.entries(onEvent)) {
        el.addEventListener(eventName, handler as EventListener);
      }
      return () => {
        for (const [eventName, handler] of Object.entries(onEvent)) {
          el.removeEventListener(eventName, handler as EventListener);
        }
      };
    }
  }, [props, onEvent]);

  return React.createElement(tag, { ref, className, style }, children);
}

export function registerWebComponents() {
  if (typeof window === 'undefined') return;
  import('@/web-components/hyst-stats-card');
  import('@/web-components/hyst-skeleton');
  import('@/web-components/hyst-badge');
  import('@/web-components/hyst-progress-bar');
  import('@/web-components/hyst-theme-toggle');
}
