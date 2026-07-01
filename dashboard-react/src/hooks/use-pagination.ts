"use client";

import { useState, useCallback } from "react";

export interface PaginationOptions {
  initialPage?: number;
  initialPageSize?: number;
  totalItems?: number;
}

export function usePagination(initialPage: number = 1, initialPageSize: number = 20, totalItems: number = 0) {
  const [page, setPage] = useState(initialPage);
  const [pageSize, setPageSize] = useState(initialPageSize);

  const reset = useCallback(() => {
    setPage(1);
  }, []);

  const nextPage = useCallback(() => {
    setPage(prev => prev + 1);
  }, []);

  const prevPage = useCallback(() => {
    setPage(prev => Math.max(1, prev - 1));
  }, []);

  const goToPage = useCallback((pageNum: number) => {
    setPage(Math.max(1, pageNum));
  }, []);

  const setItemsPerPage = useCallback((size: number) => {
    setPageSize(size);
    setPage(1);
  }, []);

  return {
    page,
    pageSize,
    setPage,
    setPageSize,
    setItemsPerPage,
    reset,
    nextPage,
    prevPage,
    goToPage,
    totalPages: Math.ceil((totalItems || 0) / pageSize),
    startIndex: (page - 1) * pageSize,
    endIndex: Math.min(page * pageSize - 1, (totalItems || 0) - 1),
  };
}