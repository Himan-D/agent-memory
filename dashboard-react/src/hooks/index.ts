import { useState, useCallback, useRef, useEffect } from "react";

export function useDebounce<T extends (...args: any[]) => any>(
  fn: T,
  delay: number = 300
): T {
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const debouncedFn = useCallback(
    (...args: Parameters<T>) => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
      timeoutRef.current = setTimeout(() => {
        fn(...args);
      }, delay);
    },
    [fn, delay]
  ) as T;

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  return debouncedFn;
}

export function useLocalStorage<T>(key: string, initialValue: T): [T, (value: T) => void] {
  const [storedValue, setStoredValue] = useState<T>(() => {
    try {
      const item = window.localStorage.getItem(key);
      return item ? JSON.parse(item) : initialValue;
    } catch {
      return initialValue;
    }
  });

  const setValue = useCallback(
    (value: T | ((val: T) => T)) => {
      try {
        const valueToStore = value instanceof Function ? value(storedValue) : value;
        setStoredValue(valueToStore);
        if (typeof window !== "undefined") {
          window.localStorage.setItem(key, JSON.stringify(valueToStore));
        }
      } catch (error) {
        console.warn(`Error setting localStorage key "${key}":`, error);
      }
    },
    [key, storedValue]
  );

  return [storedValue, setValue];
}

export function usePagination(initialPage: number = 1, initialPageSize: number = 20) {
  const [page, setPage] = useState(initialPage);
  const [pageSize, setPageSize] = useState(initialPageSize);

  const reset = useCallback(() => {
    setPage(1);
  }, []);

  const nextPage = useCallback(() => {
    setPage((prev) => prev + 1);
  }, []);

  const prevPage = useCallback(() => {
    setPage((prev) => Math.max(1, prev - 1));
  }, []);

  const goToPage = useCallback((pageNum: number) => {
    setPage(Math.max(1, pageNum));
  }, []);

  return { page, pageSize, setPage, setPageSize, reset, nextPage, prevPage, goToPage };
}

export function useSearch(initialQuery: string = "") {
  const [query, setQuery] = useState(initialQuery);
  const [debouncedQuery, setDebouncedQuery] = useState(initialQuery);

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(query);
    }, 300);
    return () => clearTimeout(timer);
  }, [query]);

  const clear = useCallback(() => {
    setQuery("");
  }, []);

  return { query, setQuery, debouncedQuery, clear };
}

export function useFilter<T extends Record<string, any>>(
  items: T[],
  filters: Record<string, string | string[] | undefined>
) {
  const [filteredItems, setFilteredItems] = useState<T[]>(items);

  useEffect(() => {
    let result = [...items];

    Object.entries(filters).forEach(([key, value]) => {
      if (!value || (Array.isArray(value) && value.length === 0)) return;

      result = result.filter((item) => {
        const itemValue = item[key];
        if (Array.isArray(value)) {
          return value.includes(String(itemValue));
        }
        return String(itemValue)
          .toLowerCase()
          .includes(String(value).toLowerCase());
      });
    });

    setFilteredItems(result);
  }, [items, JSON.stringify(filters)]);

  return filteredItems;
}

export function useSelection<T = string>() {
  const [selected, setSelected] = useState<Set<T>>(new Set());

  const toggle = useCallback((id: T) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  const selectAll = useCallback((ids: T[]) => {
    setSelected(new Set(ids));
  }, []);

  const clear = useCallback(() => {
    setSelected(new Set());
  }, []);

  const isSelected = useCallback(
    (id: T) => selected.has(id),
    [selected]
  );

  return { selected, toggle, selectAll, clear, isSelected, count: selected.size };
}

export function useConfirmation() {
  const [open, setOpen] = useState(false);
  const [onConfirm, setOnConfirm] = useState<(() => void) | null>(null);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");

  const confirm = useCallback(
    (opts: { title: string; description?: string; onConfirm: () => void }) => {
      setTitle(opts.title);
      setDescription(opts.description || "");
      setOnConfirm(() => opts.onConfirm);
      setOpen(true);
    },
    []
  );

  const handleConfirm = useCallback(() => {
    onConfirm?.();
    setOpen(false);
    setOnConfirm(null);
  }, [onConfirm]);

  const cancel = useCallback(() => {
    setOpen(false);
    setOnConfirm(null);
  }, []);

  return { open, title, description, confirm, handleConfirm, cancel };
}

export function useErrorMessage() {
  const getError = useCallback((error: unknown): string => {
    if (error instanceof Error) {
      return error.message;
    }
    if (typeof error === "string") {
      return error;
    }
    return "An unexpected error occurred";
  }, []);

  return { getError };
}