const DEFAULT_BACKEND_ORIGIN = 'http://ec2-54-87-249-176.compute-1.amazonaws.com:8081'

const CORS_HEADERS = {
  'Access-Control-Allow-Origin': '*',
  'Access-Control-Allow-Headers': 'Authorization,Content-Type,X-API-Key,X-Request-ID',
  'Access-Control-Allow-Methods': 'GET,HEAD,POST,PUT,PATCH,DELETE,OPTIONS',
  'Access-Control-Max-Age': '86400',
}

function json(data, status = 200) {
  return new Response(JSON.stringify(data, null, 2), {
    status,
    headers: {
      'Content-Type': 'application/json; charset=utf-8',
      ...CORS_HEADERS,
    },
  })
}

function withCors(response) {
  const headers = new Headers(response.headers)
  for (const [key, value] of Object.entries(CORS_HEADERS)) {
    headers.set(key, value)
  }
  return new Response(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers,
  })
}

async function fetchBackend(request, env) {
  const backendOrigin = env.BACKEND_ORIGIN || DEFAULT_BACKEND_ORIGIN
  const incomingUrl = new URL(request.url)
  const backendUrl = new URL(incomingUrl.pathname + incomingUrl.search, backendOrigin)

  return fetch(new Request(backendUrl, request))
}

export default {
  async fetch(request, env) {
    if (request.method === 'OPTIONS') {
      return new Response(null, { status: 204, headers: CORS_HEADERS })
    }

    const url = new URL(request.url)

    if (url.pathname === '/status') {
      return json({
        name: 'Hystersis API',
        service: 'hystersis-api',
        edge: 'cloudflare-workers',
        backend_origin: env.BACKEND_ORIGIN || DEFAULT_BACKEND_ORIGIN,
        backend_configured: true,
        health: 'https://api.hystersis.com/health',
        docs: 'https://hystersis.com/docs/',
        dashboard: 'https://app.hystersis.com',
      })
    }

    try {
      const backendResponse = await fetchBackend(request, env)
      return withCors(backendResponse)
    } catch (error) {
      return json({
        status: 'unavailable',
        service: 'hystersis-api',
        backend_configured: true,
        error: error instanceof Error ? error.message : 'backend fetch failed',
      }, 503)
    }
  },
}
