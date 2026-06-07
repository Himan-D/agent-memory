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

async function serveBundledAsset(env, request, assetPath) {
  const url = new URL(request.url)
  url.pathname = assetPath
  return env.ASSETS.fetch(new Request(url.toString(), request))
}

export default {
  async fetch(request, env) {
    const url = new URL(request.url)

    if (isDocsRequest(url.pathname)) {
      return env.ASSETS.fetch(request)
    }

    if (isDocsRootAsset(url.pathname)) {
      const response = await serveBundledAsset(env, request, '/docs' + url.pathname)
      if (response.status !== 404) {
        return response
      }
    }

    return env.ASSETS.fetch(request)
  },
}
