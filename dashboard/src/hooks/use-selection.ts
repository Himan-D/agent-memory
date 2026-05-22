"use client";

import { useState, useCallback } from "react";

export interface SelectionOptions {
  multiple?: boolean;
  maxSelections?: number;
}

export function useSelection<T = string>(options: SelectionOptions = {}) {
  const { multiple = false, maxSelections } = options;
  const [selected, setSelected] = useState<Set<T>>(new Set());

  const toggle = useCallback((item: T) => {
    setSelected(prev => {
      const next = new Set(prev);
      
      if (next.has(item)) {
        next.delete(item);
      } else {
        if (maxSelections && next.size >= maxSelections) {
          return prev;
        }
        next.add(item);
      }
      
      return next;
    });
  }, [maxSelections]);

  const select = useCallback((item: T) => {
    setSelected(prev => {
      if (!multiple) {
        return new Set([item]);
      }
      if (maxSelections && prev.size >= maxSelections) {
        return prev;
      }
      const next = new Set(prev);
      next.add(item);
      return next;
    });
  }, [multiple, maxSelections]);

  const deselect = useCallback((item: T) => {
    setSelected(prev => {
      const next = new Set(prev);
      next.delete(item);
      return next;
    });
  }, []);

  const selectAll = useCallback((items: T[]) => {
    if (multiple) {
      if (maxSelections && items.length > maxSelections) {
        const next = new Set(items.slice(0, maxSelections));
        setSelected(next);
      } else {
        setSelected(new Set(items));
      }
    } else {
      setSelected(new Set([items[0]]));
    }
  }, [multiple, maxSelections]);

  const clear = useCallback(() => {
    setSelected(new Set());
  }, []);

  const isSelected = useCallback((item: T) => selected.has(item), [selected]);

  const isAllSelected = useCallback((totalItems: number) => {
    if (!multiple) return false;
    return selected.size === totalItems;
  }, [multiple, selected.size]);

  return {
    selected,
    selectedIds: selected,
    toggle,
    select,
    deselect,
    selectAll,
    clear,
    isSelected,
    isAllSelected,
    count: selected.size,
  };
}