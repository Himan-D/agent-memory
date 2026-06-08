/**
 * Cloudflare Worker: landing SPA + /docs Mintlify static site.
 *
 * Mintlify docs are built into landing/dist/docs at deploy time.
 * HTML asset URLs are rewritten to /docs/* at build; this worker also
 * maps legacy root-relative Mintlify paths (/_next/, /logo/, …) to /docs/*.
 */

const DOCS_ASSET_PREFIXES = [
  '/_next/',
  '/logo/',
  '/favicons/',
  '/images/',
  '/icons/',
  '/favicon.svg',
  '/sitemap.xml',
  '/llms.txt',
  '/public/',
]

function isDocsRequest(pathname) {
  return pathname === '/docs' || pathname.startsWith('/docs/')
}

function isDocsRootAsset(pathname) {
  if (pathname.startsWith('/docs/')) {
    return false
  }
  return DOCS_ASSET_PREFIXES.some((prefix) =>
    prefix.endsWith('/')
      ? pathname.startsWith(prefix)
      : pathname === prefix || pathname.startsWith(prefix + '/')
  )
}

function looksLikeAssetPath(pathname) {
  const last = pathname.split('/').pop() || ''
  return last.includes('.') && !last.endsWith('.html')
}

function isSpaNavigation(request, pathname) {
  return (
    request.method === 'GET' &&
    !looksLikeAssetPath(pathname) &&
    !isDocsRequest(pathname)
  )
}

async function serveBundledAsset(env, request, assetPath) {
  const url = new URL(request.url)
  url.pathname = assetPath
  return await env.ASSETS.fetch(new Request(url.toString(), request))
}

async function serveIndexHtml(env, request) {
  const indexUrl = new URL(request.url)
  indexUrl.pathname = '/index.html'
  return await env.ASSETS.fetch(new Request(indexUrl.toString(), request))
}

export default {
  async fetch(request, env) {
    try {
      const url = new URL(request.url)

      if (isDocsRequest(url.pathname)) {
        return await env.ASSETS.fetch(request)
      }

      if (isDocsRootAsset(url.pathname)) {
        const response = await serveBundledAsset(env, request, '/docs' + url.pathname)
        if (response.status !== 404) {
          return response
        }
      }

      // Serve index.html directly for SPA routes. Fetching /blog as a static
      // asset throws Worker 1101 when combined with assets.not_found_handling.
      if (isSpaNavigation(request, url.pathname)) {
        return await serveIndexHtml(env, request)
      }

      return await env.ASSETS.fetch(request)
    } catch (err) {
      return new Response('Worker error: ' + err.message, { status: 500 })
    }
  },
}
