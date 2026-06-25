/**
 * Cloudflare Worker: landing SPA.
 */

function isDocsRequest(pathname) {
  return pathname === '/docs' || pathname.startsWith('/docs/')
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
        const path = url.pathname.replace(/^\/docs/, '') || '/'
        const target = `https://docs.hystersis.com${path}${url.search}`
        return Response.redirect(target, 301)
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
