/**
 * Cloudflare Worker: landing SPA + /docs Mintlify static site.
 *
 * Mintlify docs are built into landing/dist/docs at deploy time.
 * Serves /docs from the bundled export and maps root-relative Mintlify
 * asset paths (/_next/, /logo/, etc.) to /docs/* when viewing docs.
 */

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

function isDocsRootAsset(pathname) {
  return DOCS_ASSET_PREFIXES.some((prefix) => pathname.startsWith(prefix))
}

function isViewingDocs(request) {
  const referer = request.headers.get('Referer') || ''
  return referer.includes('/docs')
}

async function serveBundledAsset(env, request, assetPath) {
  const url = new URL(request.url)
  url.pathname = assetPath
  return env.ASSETS.fetch(new Request(url.toString(), request))
}

export default {
  async fetch(request, env) {
    const url = new URL(request.url)

    // Docs pages and /docs/_next/* etc. — serve from landing/dist/docs
    if (isDocsRequest(url.pathname)) {
      return env.ASSETS.fetch(request)
    }

    // Mintlify HTML uses root-relative asset URLs (/_next/, /logo/, …).
    // Map those to the bundled /docs/* paths when the user is on /docs.
    if (isDocsRootAsset(url.pathname) && isViewingDocs(request)) {
      const response = await serveBundledAsset(env, request, '/docs' + url.pathname)
      if (response.status !== 404) {
        return response
      }
    }

    return env.ASSETS.fetch(request)
  },
}
