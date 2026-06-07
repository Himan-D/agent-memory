/** Stable fallback for missing created_at timestamps (avoids Date.now() during render). */
export function parseCreatedAt(createdAt?: string | null): Date {
  return new Date(createdAt || 0);
}
