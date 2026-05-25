import { NextResponse } from "next/server";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "https://api.hystersis.ai";
const ADMIN_API_KEY = process.env.ADMIN_API_KEY || "";

const ADMIN_ENDPOINTS = [
  "/admin/users",
  "/admin/api-keys",
  "/admin/invites",
  "/compression/",
  "/tier/",
  "/search/enhanced",
  "/search/hybrid",
  "/search/advanced",
  "/graph/",
  "/compact/",
  "/projects/",
  "/skills/",
  "/chains/",
  "/webhooks/",
  "/alerts/",
  "/groups/",
  "/agents/",
  "/users/",
  "/sessions/",
  "/entities/",
  "/memories/",
  "/playground/",
  "/documents/",
  "/notifications/",
  "/feedback",
  "/api-keys",
];

const WRITE_ENDPOINTS = [
  "/memories",
  "/entities",
  "/skills",
  "/chains",
  "/projects",
  "/webhooks",
  "/sessions",
  "/feedback",
  "/alerts",
  "/relations",
  "/playground/compress",
  "/playground/search",
  "/documents",
  "/groups",
  "/agents",
  "/notifications",
  "/compact",
];

function isAdminEndpoint(path: string): boolean {
  const pathOnly = path.split('?')[0];
  const normalizedPath = pathOnly.endsWith('/') ? pathOnly.slice(0, -1) : pathOnly;
  return ADMIN_ENDPOINTS.some(ep => {
    const normalizedEp = ep.endsWith('/') ? ep.slice(0, -1) : ep;
    return normalizedPath.startsWith(normalizedEp);
  });
}

const USER_ENDPOINTS = [
  "/api-keys",
  "/notifications/create",
];

function isUserEndpoint(path: string): boolean {
  return USER_ENDPOINTS.some(ep => path.startsWith(ep));
}

function needsApiKey(method: string, endpoint: string): boolean {
  if (method === "GET") return false;
  if (isAdminEndpoint(endpoint) || isUserEndpoint(endpoint)) return true;
  return WRITE_ENDPOINTS.some(ep => endpoint.startsWith(ep));
}

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

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const endpoint = searchParams.get("endpoint") || "";
  const url = `${API_BASE}${endpoint}`;

  const apiKey = ADMIN_API_KEY || "am_AYQh3k5V47AVVoyY_1776234755";

  const response = await fetch(url, {
    headers: {
      "Content-Type": "application/json",
      "X-API-Key": apiKey,
    },
  });

  const { data, status } = await safeFetchResponse(response);
  return NextResponse.json(data, { status });
}

export async function POST(request: Request) {
  const { searchParams } = new URL(request.url);
  const endpoint = searchParams.get("endpoint") || "";
  const url = `${API_BASE}${endpoint}`;

  const apiKey = ADMIN_API_KEY || "am_AYQh3k5V47AVVoyY_1776234755";

  const isFormData = isFormDataRequest(request);

  let body: string | FormData;
  let headers: Record<string, string>;

  if (isFormData) {
    const formData = await request.formData();
    body = formData;
    headers = { "X-API-Key": apiKey };
  } else {
    body = await request.text();
    headers = { "Content-Type": "application/json", "X-API-Key": apiKey };
  }

  const response = await fetch(url, {
    method: "POST",
    headers,
    body: body as BodyInit,
  });

  const { data, status } = await safeFetchResponse(response);
  return NextResponse.json(data, { status });
}

export async function PUT(request: Request) {
  const { searchParams } = new URL(request.url);
  const endpoint = searchParams.get("endpoint") || "";
  const url = `${API_BASE}${endpoint}`;

  const apiKey = ADMIN_API_KEY || "am_AYQh3k5V47AVVoyY_1776234755";

  const body = await request.text();

  const response = await fetch(url, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
      "X-API-Key": apiKey,
    },
    body,
  });

  const { data, status } = await safeFetchResponse(response);
  return NextResponse.json(data, { status });
}

export async function DELETE(request: Request) {
  const { searchParams } = new URL(request.url);
  const endpoint = searchParams.get("endpoint") || "";
  const url = `${API_BASE}${endpoint}`;

  const apiKey = ADMIN_API_KEY || "am_AYQh3k5V47AVVoyY_1776234755";

  const response = await fetch(url, {
    method: "DELETE",
    headers: {
      "Content-Type": "application/json",
      "X-API-Key": apiKey,
    },
  });

  const { data, status } = await safeFetchResponse(response);
  return NextResponse.json(data, { status });
}