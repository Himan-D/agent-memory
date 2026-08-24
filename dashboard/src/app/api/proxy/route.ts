import { NextResponse } from "next/server";

const API_BASE = (process.env.NEXT_PUBLIC_API_URL || "https://api.hystersis.com").replace(
  /\/$/,
  "",
);
const ADMIN_API_KEY = process.env.ADMIN_API_KEY || "";

// Allowlist of valid endpoint prefixes for SSRF protection.
// Only path prefixes on API_BASE are reachable — never absolute URLs.
const ALLOWED_PREFIXES = [
  "/memories",
  "/entities",
  "/relations",
  "/sessions",
  "/search",
  "/skills",
  "/chains",
  "/agents",
  "/groups",
  "/projects",
  "/webhooks",
  "/alerts",
  "/notifications",
  "/analytics",
  "/compression",
  "/tier",
  "/playground",
  "/admin/users",
  "/admin/api-keys",
  "/admin/invites",
  "/admin/sync",
  "/admin/tokens",
  "/auth/",
  "/billing/",
  "/health",
  "/ready",
  "/status",
  "/graph",
  "/feedback",
  "/compact",
  "/backup",
  "/concepts",
  "/reminders",
  "/safety",
  "/demo",
  "/stripe",
  "/documents",
  "/api-keys",
  "/metrics",
  "/wiki",
  "/reviews",
  "/audit",
  "/sources",
  "/events",
  "/v3",
];

// User-scoped endpoints that prefer session token (not admin API key)
const SESSION_AUTH_ENDPOINTS = [
  "/admin/users/me",
  "/admin/users",
  "/admin/invites",
  "/admin/api-keys",
  "/admin/tokens",
  "/auth/change-password",
  "/memories",
  "/entities",
  "/relations",
  "/sessions",
  "/search",
  "/skills",
  "/chains",
  "/agents",
  "/groups",
  "/projects",
  "/webhooks",
  "/alerts",
  "/notifications",
  "/analytics",
  "/compression",
  "/tier",
  "/playground",
  "/graph",
  "/feedback",
  "/compact",
  "/backup",
  "/documents",
  "/api-keys",
  "/wiki",
  "/reviews",
  "/audit",
  "/sources",
  "/billing/",
  "/stripe/checkout",
  "/events",
  "/v3",
];

const ADMIN_ENDPOINTS = ["/admin"];

function jsonError(
  error: string,
  status: number,
  extras?: Record<string, unknown>,
) {
  return NextResponse.json(
    {
      error,
      status,
      ...extras,
    },
    {
      status,
      headers: {
        "Cache-Control": "no-store",
        "X-Content-Type-Options": "nosniff",
      },
    },
  );
}

function isFormDataRequest(request: Request): boolean {
  const contentType = request.headers.get("content-type") || "";
  return contentType.includes("multipart/form-data");
}

/** Normalize + validate endpoint path (SSRF harden). */
function sanitizeEndpoint(raw: string): { ok: true; path: string } | { ok: false; reason: string } {
  if (!raw || typeof raw !== "string") {
    return { ok: false, reason: "Missing endpoint" };
  }
  let decoded: string;
  try {
    decoded = decodeURIComponent(raw);
  } catch {
    return { ok: false, reason: "Invalid endpoint encoding" };
  }

  // Reject absolute URLs / schemes / hosts (SSRF)
  if (/^[a-zA-Z][a-zA-Z0-9+.-]*:/.test(decoded) || decoded.includes("://")) {
    return { ok: false, reason: "Absolute URLs are not allowed" };
  }
  if (decoded.includes("\\") || decoded.includes("\0")) {
    return { ok: false, reason: "Invalid endpoint characters" };
  }
  if (decoded.includes("..")) {
    return { ok: false, reason: "Path traversal is not allowed" };
  }
  // Collapse accidental double slashes in path only
  let path = decoded.startsWith("/") ? decoded : `/${decoded}`;
  path = path.replace(/\/{2,}/g, "/");

  // No userinfo / host fragments
  if (path.includes("@") || path.includes("?")) {
    // Query should come from request searchParams, not embedded in endpoint
    const q = path.indexOf("?");
    if (q >= 0) path = path.slice(0, q);
  }

  const isAllowed = ALLOWED_PREFIXES.some(
    (prefix) => path === prefix || path.startsWith(prefix + "/") || path.startsWith(prefix),
  );
  if (!isAllowed) {
    return { ok: false, reason: "Endpoint not allowed" };
  }

  return { ok: true, path };
}

async function safeFetchResponse(
  response: Response,
): Promise<{ data: unknown; status: number }> {
  const status = response.status;

  if (status === 204) {
    return { data: { success: true }, status: 204 };
  }

  const contentType = response.headers.get("content-type") || "";

  if (contentType.includes("application/json")) {
    try {
      const data = await response.json();
      return { data, status };
    } catch {
      const text = await response.text();
      return {
        data: { error: "Invalid JSON from upstream", message: text || response.statusText },
        status,
      };
    }
  }

  const text = await response.text();
  try {
    const parsed = JSON.parse(text);
    return { data: parsed, status };
  } catch {
    return {
      data: {
        error: status >= 400 ? "Upstream error" : "Non-JSON response",
        message: text || response.statusText,
      },
      status,
    };
  }
}

function buildTargetUrl(endpoint: string, requestSearchParams: URLSearchParams): string {
  const url = new URL(`${API_BASE}${endpoint}`);
  // Ensure we never leave API_BASE host
  if (url.origin !== new URL(API_BASE).origin) {
    throw new Error("Refusing to proxy outside configured API origin");
  }
  requestSearchParams.forEach((value, key) => {
    if (key !== "endpoint") {
      url.searchParams.append(key, value);
    }
  });
  return url.toString();
}

function usesSessionAuth(endpoint: string): boolean {
  return SESSION_AUTH_ENDPOINTS.some(
    (prefix) => endpoint === prefix || endpoint.startsWith(prefix),
  );
}

function getBackendAuth(
  request: Request,
  endpoint: string,
): Record<string, string> | null {
  const authHeader = request.headers.get("Authorization") || "";
  const sessionToken = authHeader.replace(/^Bearer\s+/i, "");
  const clientApiKey = request.headers.get("X-API-Key") || "";

  if (usesSessionAuth(endpoint)) {
    if (sessionToken) {
      return { Authorization: `Bearer ${sessionToken}` };
    }
    if (clientApiKey) {
      return { "X-API-Key": clientApiKey };
    }
    return {};
  }

  const requiresAdminKey = ADMIN_ENDPOINTS.some((prefix) =>
    endpoint.startsWith(prefix),
  );

  if (requiresAdminKey) {
    if (sessionToken) {
      return { Authorization: `Bearer ${sessionToken}` };
    }
    if (!ADMIN_API_KEY) {
      return null;
    }
    return { "X-API-Key": ADMIN_API_KEY };
  }

  if (sessionToken) {
    return { Authorization: `Bearer ${sessionToken}` };
  }
  if (clientApiKey) {
    return { "X-API-Key": clientApiKey };
  }

  return {};
}

function forwardRateLimitHeaders(upstream: Response, res: NextResponse) {
  const keys = [
    "x-ratelimit-limit",
    "x-ratelimit-remaining",
    "x-ratelimit-reset",
    "retry-after",
  ];
  for (const key of keys) {
    const val = upstream.headers.get(key);
    if (val) res.headers.set(key, val);
  }
  res.headers.set("Cache-Control", "no-store");
  res.headers.set("X-Content-Type-Options", "nosniff");
}

async function proxyRequest(request: Request, method: string): Promise<Response> {
  try {
    const { searchParams } = new URL(request.url);
    const endpointRaw = searchParams.get("endpoint") || "";

    const sanitized = sanitizeEndpoint(endpointRaw);
    if (!sanitized.ok) {
      const status =
        sanitized.reason === "Missing endpoint"
          ? 400
          : sanitized.reason === "Endpoint not allowed"
            ? 403
            : 400;
      return jsonError(sanitized.reason, status);
    }

    let url: string;
    try {
      url = buildTargetUrl(sanitized.path, searchParams);
    } catch (e) {
      return jsonError(
        e instanceof Error ? e.message : "Invalid proxy target",
        400,
      );
    }

    const isFormData = isFormDataRequest(request);
    const authHeaders = getBackendAuth(request, sanitized.path);
    if (authHeaders === null) {
      return jsonError(
        "Admin authorization is not configured. Sign in with an admin session.",
        401,
      );
    }

    if (isFormData) {
      const formData = await request.formData();
      const headers = new Headers({ ...authHeaders });

      let response: Response;
      try {
        response = await fetch(url, {
          method,
          headers,
          body: formData,
        });
      } catch (fetchError: unknown) {
        const fetchErrorMessage =
          fetchError instanceof Error ? fetchError.message : "Fetch failed";
        return jsonError("Upstream fetch failed", 502, {
          message: fetchErrorMessage,
        });
      }

      const { data, status } = await safeFetchResponse(response);
      const res = NextResponse.json(data, { status });
      forwardRateLimitHeaders(response, res);
      return res;
    }

    const body =
      method !== "GET" && method !== "HEAD" ? await request.text() : undefined;
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      ...authHeaders,
    };

    let response: Response;
    try {
      response = await fetch(url, {
        method,
        headers,
        ...(body && { body }),
      });
    } catch (fetchError: unknown) {
      const fetchErrorMessage =
        fetchError instanceof Error ? fetchError.message : "Fetch failed";
      return jsonError("Upstream fetch failed", 502, {
        message: fetchErrorMessage,
      });
    }

    const { data, status } = await safeFetchResponse(response);
    const res = NextResponse.json(data, { status });
    forwardRateLimitHeaders(response, res);
    return res;
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : "Unknown error";
    return jsonError("Proxy error", 500, { message: errorMessage });
  }
}

export async function GET(request: Request) {
  return proxyRequest(request, "GET");
}

export async function POST(request: Request) {
  return proxyRequest(request, "POST");
}

export async function PUT(request: Request) {
  return proxyRequest(request, "PUT");
}

export async function PATCH(request: Request) {
  return proxyRequest(request, "PATCH");
}

export async function DELETE(request: Request) {
  return proxyRequest(request, "DELETE");
}
