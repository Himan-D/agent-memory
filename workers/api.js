const SERVICE = 'hystersis-api'
const VERSION = 'edge-2026-06-08'

const CORS_HEADERS = {
  'Access-Control-Allow-Origin': '*',
  'Access-Control-Allow-Methods': 'GET,HEAD,POST,PUT,PATCH,DELETE,OPTIONS',
  'Access-Control-Allow-Headers': 'Authorization,Content-Type,X-API-Key,X-Request-ID',
  'Access-Control-Max-Age': '86400',
}

function json(data, init = {}) {
  return new Response(JSON.stringify(data, null, 2), {
    ...init,
    headers: {
      'Content-Type': 'application/json; charset=utf-8',
      ...CORS_HEADERS,
      ...(init.headers || {}),
    },
  })
}

function text(data, init = {}) {
  return new Response(data, {
    ...init,
    headers: {
      'Content-Type': 'text/plain; charset=utf-8',
      ...CORS_HEADERS,
      ...(init.headers || {}),
    },
  })
}

function openapi(origin) {
  return {
    openapi: '3.0.3',
    info: {
      title: 'Hystersis API',
      version: '1.0.0',
      description: 'Persistent memory infrastructure for AI agents.',
    },
    servers: [{ url: origin, description: 'Production API edge' }],
    paths: {
      '/health': {
        get: {
          summary: 'Health check',
          responses: { 200: { description: 'API edge is healthy' } },
        },
      },
      '/ready': {
        get: {
          summary: 'Readiness check',
          responses: { 200: { description: 'API edge is ready' } },
        },
      },
      '/openapi.json': {
        get: {
          summary: 'OpenAPI document',
          responses: { 200: { description: 'OpenAPI schema' } },
        },
      },
    },
  }
}

function serverCard(origin) {
  return {
    name: 'Hystersis API',
    service: SERVICE,
    version: VERSION,
    base_url: origin,
    health: `${origin}/health`,
    openapi: `${origin}/openapi.json`,
    docs: 'https://hystersis.com/docs/',
    dashboard: 'https://app.hystersis.com',
  }
}

async function proxyToBackend(request, env) {
  if (!env.BACKEND_URL) {
    return null
  }

  const source = new URL(request.url)
  const target = new URL(env.BACKEND_URL)
  target.pathname = source.pathname
  target.search = source.search

  const headers = new Headers(request.headers)
  headers.set('X-Forwarded-Host', source.host)
  headers.set('X-Forwarded-Proto', source.protocol.replace(':', ''))

  return fetch(new Request(target.toString(), {
    method: request.method,
    headers,
    body: request.body,
    redirect: 'manual',
  }))
}

export default {
  async fetch(request, env) {
    const url = new URL(request.url)

    if (request.method === 'OPTIONS') {
      return new Response(null, { status: 204, headers: CORS_HEADERS })
    }

    if (url.pathname === '/health') {
      return json({
        status: 'ok',
        service: SERVICE,
        version: VERSION,
        edge: 'cloudflare-workers',
        backend_configured: Boolean(env.BACKEND_URL),
        timestamp: new Date().toISOString(),
      })
    }

    if (url.pathname === '/ready') {
      return json({
        status: 'ready',
        service: SERVICE,
        backend_configured: Boolean(env.BACKEND_URL),
      })
    }

    if (url.pathname === '/status') {
      return json(serverCard(url.origin))
    }

    if (url.pathname === '/openapi.json' || url.pathname === '/swagger.json') {
      return json(openapi(url.origin))
    }

    if (url.pathname === '/llms.txt') {
      return text([
        '# Hystersis API',
        '',
        `- API Base: ${url.origin}`,
        `- Health: ${url.origin}/health`,
        `- OpenAPI: ${url.origin}/openapi.json`,
        '- Docs: https://hystersis.com/docs/',
        '',
      ].join('\n'))
    }

    const backendResponse = await proxyToBackend(request, env)
    if (backendResponse) {
      const response = new Response(backendResponse.body, backendResponse)
      for (const [key, value] of Object.entries(CORS_HEADERS)) {
        response.headers.set(key, value)
      }
      return response
    }

    return json({
      error: 'backend_unavailable',
      message: 'api.hystersis.com is deployed at the edge, but the Go API backend is not configured for this Worker yet.',
      service: SERVICE,
      docs: 'https://hystersis.com/docs/',
      health: `${url.origin}/health`,
    }, { status: 503 })
  },
}
