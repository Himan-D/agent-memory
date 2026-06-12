import { NextResponse } from "next/server";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "https://api.hystersis.com";
const ADMIN_API_KEY = process.env.ADMIN_API_KEY || "";

// Allowlist of valid endpoint prefixes for SSRF protection
const ALLOWED_PREFIXES = [
  "/memories", "/entities", "/relations", "/sessions",
  "/search", "/skills", "/chains", "/agents", "/groups",
  "/projects", "/webhooks", "/alerts", "/notifications",
  "/analytics", "/compression", "/tier", "/playground",
  "/admin/users", "/admin/api-keys", "/admin/invites", "/admin/sync", "/admin/tokens",
  "/auth/", "/billing/", "/health", "/ready", "/status",
  "/graph", "/feedback", "/compact", "/backup",
  "/concepts", "/reminders", "/safety",
  "/demo", "/stripe",
  "/documents", "/api-keys", "/metrics", "/wiki", "/reviews",
  "/audit", "/sources",
];

// User-scoped endpoints that require session token (not admin API key)
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
  "/notifications/preferences",
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
];

// List of endpoints that require admin API key
const ADMIN_ENDPOINTS = [
  "/admin",
];

function isFormDataRequest(request: Request): boolean {
  const contentType = request.headers.get("content-type") || "";
  return contentType.includes("multipart/form-data");
}

async function safeFetchResponse(response: Response): Promise<{ data: unknown; status: number }> {
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
      return { data: { message: text || response.statusText }, status };
    }
  }

  const text = await response.text();

  try {
    const parsed = JSON.parse(text);
    return { data: parsed, status };
  } catch {
    return { data: { message: text || response.statusText, error: text || response.statusText }, status };
  }
}

function buildTargetUrl(endpoint: string, requestSearchParams: URLSearchParams): string {
  let decoded: string;
  try {
    decoded = decodeURIComponent(endpoint);
  } catch {
    decoded = endpoint;
  }
  const clean = decoded.startsWith("/") ? decoded : `/${decoded}`;
  const url = new URL(`${API_BASE}${clean}`);
  requestSearchParams.forEach((value, key) => {
    if (key !== "endpoint") {
      url.searchParams.append(key, value);
    }
  });
  return url.toString();
}

function usesSessionAuth(endpoint: string): boolean {
  return SESSION_AUTH_ENDPOINTS.some(prefix => endpoint.startsWith(prefix));
}

function getBackendAuth(request: Request, endpoint: string): Record<string, string> | null {
  const authHeader = request.headers.get("Authorization") || "";
  const sessionToken = authHeader.replace(/^Bearer\s+/i, "");

  // User-scoped endpoints use the signed-in user's session token
  if (usesSessionAuth(endpoint)) {
    if (sessionToken) {
      return { Authorization: `Bearer ${sessionToken}` };
    }
    return {};
  }

  // Check if this endpoint requires admin API key
  const requiresAdminKey = ADMIN_ENDPOINTS.some(prefix => endpoint.startsWith(prefix));

  // Prefer the signed-in user's session for admin endpoints too; the backend
  // decides whether the user's role or key scopes are sufficient.
  if (requiresAdminKey) {
    if (sessionToken) {
      return { Authorization: `Bearer ${sessionToken}` };
    }
    if (!ADMIN_API_KEY) {
      return null;
    }
    return { "X-API-Key": ADMIN_API_KEY };
  }

  // Non-admin endpoints: use session token if available
  if (sessionToken) {
    return { Authorization: `Bearer ${sessionToken}` };
  }

  return {};
}

async function proxyRequest(request: Request, method: string): Promise<Response> {
  try {
    const { searchParams } = new URL(request.url);
    const endpoint = searchParams.get("endpoint") || "";

    if (!endpoint) {
      return NextResponse.json({ error: "Missing endpoint" }, { status: 400 });
    }

    // Block path traversal
    if (endpoint.includes("..") || endpoint.includes("//")) {
      return NextResponse.json({ error: "Invalid endpoint" }, { status: 400 });
    }

    // SSRF protection: only allow known endpoint prefixes
    const isAllowed = ALLOWED_PREFIXES.some(prefix => endpoint.startsWith(prefix));
    if (!isAllowed) {
      return NextResponse.json({ error: "Endpoint not allowed" }, { status: 403 });
    }

    const url = buildTargetUrl(endpoint, searchParams);

    const isFormData = isFormDataRequest(request);
    const authHeaders = getBackendAuth(request, endpoint);
    if (authHeaders === null) {
      return NextResponse.json({ error: "Admin authorization is not configured. Sign in with an admin session." }, { status: 401 });
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
        const fetchErrorMessage = fetchError instanceof Error ? fetchError.message : "Fetch failed";
        return NextResponse.json({ error: "Fetch error: " + fetchErrorMessage, details: url }, { status: 502 });
      }

      const { data, status } = await safeFetchResponse(response);
      return NextResponse.json(data, { status });
    }

    const body = method !== "GET" && method !== "HEAD" ? await request.text() : undefined;
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
      const fetchErrorMessage = fetchError instanceof Error ? fetchError.message : "Fetch failed";
      return NextResponse.json({ error: "Fetch error: " + fetchErrorMessage, details: url }, { status: 502 });
    }

    const { data, status } = await safeFetchResponse(response);
    return NextResponse.json(data, { status });
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : "Unknown error";
    return NextResponse.json({ error: "Proxy error", details: errorMessage }, { status: 500 });
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

export async function DELETE(request: Request) {
  return proxyRequest(request, "DELETE");
}
