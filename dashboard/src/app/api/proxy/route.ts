import { NextResponse } from "next/server";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "https://api.hystersis.ai";
const ADMIN_API_KEY = process.env.ADMIN_API_KEY || "";

const ADMIN_ENDPOINTS = [
  "/admin/users",
  "/admin/api-keys",
  "/admin/invites",
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
];

function isAdminEndpoint(path: string): boolean {
  return ADMIN_ENDPOINTS.some(ep => path.startsWith(ep));
}

function needsApiKey(method: string, endpoint: string): boolean {
  if (method === "GET") return false;
  if (isAdminEndpoint(endpoint)) return true;
  return WRITE_ENDPOINTS.some(ep => endpoint.startsWith(ep));
}

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const endpoint = searchParams.get("endpoint") || "";
  const url = `${API_BASE}${endpoint}`;
  
  let apiKey = request.headers.get("x-api-key") || request.headers.get("Authorization")?.replace("Bearer ", "") || "";
  
  if (isAdminEndpoint(endpoint) && !apiKey) {
    apiKey = ADMIN_API_KEY;
  }
  
  const response = await fetch(url, {
    headers: {
      "Content-Type": "application/json",
      "X-API-Key": apiKey,
    },
  });
  
  const data = await response.json();
  return NextResponse.json(data);
}

export async function POST(request: Request) {
  const { searchParams } = new URL(request.url);
  const endpoint = searchParams.get("endpoint") || "";
  const url = `${API_BASE}${endpoint}`;
  
  let apiKey = request.headers.get("x-api-key") || request.headers.get("Authorization")?.replace("Bearer ", "") || "";
  
  if (needsApiKey("POST", endpoint) && !apiKey) {
    apiKey = ADMIN_API_KEY;
  }
  
  const body = await request.json();
  
  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-API-Key": apiKey,
    },
    body: JSON.stringify(body),
  });
  
  const data = await response.json();
  return NextResponse.json(data);
}

export async function PUT(request: Request) {
  const { searchParams } = new URL(request.url);
  const endpoint = searchParams.get("endpoint") || "";
  const url = `${API_BASE}${endpoint}`;
  
  let apiKey = request.headers.get("x-api-key") || request.headers.get("Authorization")?.replace("Bearer ", "") || "";
  
  if (needsApiKey("PUT", endpoint) && !apiKey) {
    apiKey = ADMIN_API_KEY;
  }
  
  const body = await request.json();
  
  const response = await fetch(url, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
      "X-API-Key": apiKey,
    },
    body: JSON.stringify(body),
  });
  
  const data = await response.json();
  return NextResponse.json(data);
}

export async function DELETE(request: Request) {
  const { searchParams } = new URL(request.url);
  const endpoint = searchParams.get("endpoint") || "";
  const url = `${API_BASE}${endpoint}`;
  
  let apiKey = request.headers.get("x-api-key") || request.headers.get("Authorization")?.replace("Bearer ", "") || "";
  
  if (needsApiKey("DELETE", endpoint) && !apiKey) {
    apiKey = ADMIN_API_KEY;
  }
  
  const response = await fetch(url, {
    method: "DELETE",
    headers: {
      "Content-Type": "application/json",
      "X-API-Key": apiKey,
    },
  });
  
  if (response.status === 204) {
    return NextResponse.json({ success: true });
  }
  
  const data = await response.json();
  return NextResponse.json(data);
}