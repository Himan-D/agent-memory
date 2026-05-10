import { NextResponse } from "next/server";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "https://api.hystersis.ai";

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

function getBackendAuth(request: Request): Record<string, string> {
  // Get session token from Authorization header (forwarded from client)
  const authHeader = request.headers.get("Authorization") || "";
  const sessionToken = authHeader.replace(/^Bearer\s+/i, "");

  if (sessionToken) {
    return { Authorization: `Bearer ${sessionToken}` };
  }

  // No auth - let backend validate
  return {};
}

async function proxyRequest(request: Request, method: string): Promise<Response> {
  const { searchParams } = new URL(request.url);
  const endpoint = searchParams.get("endpoint") || "";
  const url = buildTargetUrl(endpoint);

  const isFormData = isFormDataRequest(request);

  // Get auth headers - forward session token
  const authHeaders = getBackendAuth(request);

  if (isFormData) {
    const formData = await request.formData();
    const headers = new Headers({ ...authHeaders });

    const response = await fetch(url, {
      method,
      headers,
      body: formData,
    });

    const { data, status } = await safeFetchResponse(response);
    return NextResponse.json(data, { status });
  }

  const body = await request.text();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...authHeaders,
  };

  const response = await fetch(url, {
    method,
    headers,
    body,
  });

  const { data, status } = await safeFetchResponse(response);
  return NextResponse.json(data, { status });
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