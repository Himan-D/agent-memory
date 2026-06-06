/**
 * Cloudflare Worker: landing SPA + /docs proxy to Mintlify static site.
 *
 * /docs and /docs/* are proxied to docs.hystersis.com so hystersis.com/docs
 * serves full Mintlify documentation on the same brand domain.
 */

const DOCS_ORIGIN = 'https://docs.hystersis.com'

const DOCS_ASSET_PREFIXES = [
  '/_next/',
  '/logo/',
  '/favicons/',
  '/images/',
  '/icons/',
]

function isDocsRequest(pathname) {
  return pathname === '/docs' || pathname.startsWith('/docs/')
}

function isDocsAsset(pathname) {
  return DOCS_ASSET_PREFIXES.some((prefix) => pathname.startsWith(prefix))
}

function docsTargetUrl(url) {
  const target = new URL(url)
  if (target.pathname === '/docs') {
    target.pathname = '/'
  } else if (target.pathname.startsWith('/docs/')) {
    target.pathname = target.pathname.slice('/docs'.length) || '/'
  }
  target.hostname = new URL(DOCS_ORIGIN).hostname
  target.protocol = 'https:'
  return target
}

async function proxyToDocs(request) {
  const targetUrl = docsTargetUrl(new URL(request.url))
  const headers = new Headers(request.headers)
  headers.set('Host', targetUrl.hostname)
  headers.delete('cf-connecting-ip')

  const proxyRequest = new Request(targetUrl.toString(), {
    method: request.method,
    headers,
    body: request.method !== 'GET' && request.method !== 'HEAD' ? request.body : undefined,
    redirect: 'follow',
  })

  const response = await fetch(proxyRequest)
  const outHeaders = new Headers(response.headers)
  outHeaders.set('X-Docs-Proxy', 'hystersis')

  return new Response(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers: outHeaders,
  })
}

export default {
  async fetch(request, env) {
    const url = new URL(request.url)

    if (isDocsRequest(url.pathname)) {
      return proxyToDocs(request)
    }

    // Mintlify client-side navigation may request root asset paths
    if (isDocsAsset(url.pathname)) {
      const referer = request.headers.get('Referer') || ''
      if (referer.includes('/docs')) {
        const target = new URL(url.pathname + url.search, DOCS_ORIGIN)
        return fetch(target.toString())
      }
    }

    return env.ASSETS.fetch(request)
  },
}
