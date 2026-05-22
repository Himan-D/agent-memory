import { NextResponse } from "next/server";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "https://api.hystersis.ai";
const ADMIN_API_KEY = process.env.ADMIN_API_KEY || "";

// List of endpoints that require admin API key
const ADMIN_ENDPOINTS = [
  "/admin",
  "/compression",
  "/tier",
  "/search/enhanced",
  "/projects",
  "/skills",
  "/chains",
  "/webhooks",
  "/alerts",
  "/groups",
  "/agents",
  "/users",
  "/sessions",
  "/entities",
  "/memories",
  "/analytics",
  "/notifications",
  "/backup",
  "/compact",
  "/sync",
  "/playground",
  "/documents",
  "/graph",
  "/feedback",
  "/wiki",
  "/reviews",
  "/relations",
  "/api-keys",
  "/metrics",
  "/health",
  "/ready",
  "/status",
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

function buildTargetUrl(endpoint: string): string {
  let decoded: string;
  try {
    decoded = decodeURIComponent(endpoint);
  } catch {
    decoded = endpoint;
  }
  const clean = decoded.startsWith("/") ? decoded : `/${decoded}`;
  return `${API_BASE}${clean}`;
}

function getBackendAuth(request: Request, endpoint: string): Record<string, string> | null {
  const authHeader = request.headers.get("Authorization") || "";
  const sessionToken = authHeader.replace(/^Bearer\s+/i, "");

  // Check if this endpoint requires admin API key
  const requiresAdminKey = ADMIN_ENDPOINTS.some(prefix => endpoint.startsWith(prefix));

  // Admin endpoints ALWAYS get the admin API key (even if user has a session)
  if (requiresAdminKey) {
    if (!ADMIN_API_KEY) {
      return null;
    }
    console.log(`[PROXY] Admin endpoint "${endpoint}" → using X-API-Key`);
    return { "X-API-Key": ADMIN_API_KEY };
  }

  // Non-admin endpoints: use session token if available
  if (sessionToken) {
    return { Authorization: `Bearer ${sessionToken}` };
  }

  console.log(`[PROXY] No auth for endpoint "${endpoint}"`);
  return {};
}

async function proxyRequest(request: Request, method: string): Promise<Response> {
  try {
    const { searchParams } = new URL(request.url);
    const endpoint = searchParams.get("endpoint") || "";
    
    if (!endpoint) {
      return NextResponse.json({ error: "Missing endpoint parameter" }, { status: 400 });
    }
    
    const url = buildTargetUrl(endpoint);

    const isFormData = isFormDataRequest(request);
    const authHeaders = getBackendAuth(request, endpoint);
    if (authHeaders === null) {
      return NextResponse.json({ error: "ADMIN_API_KEY not configured" }, { status: 401 });
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
    const stack = error instanceof Error ? error.stack : undefined;
    return NextResponse.json({ error: "Proxy error: " + errorMessage, stack }, { status: 500 });
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