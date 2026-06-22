const DOCS_PREFIX = '/docs'

function stripDocsPrefix(pathname) {
  if (pathname === DOCS_PREFIX) {
    return '/'
  }
  if (pathname.startsWith(DOCS_PREFIX + '/')) {
    return pathname.slice(DOCS_PREFIX.length) || '/'
  }
  return pathname
}

async function fetchAsset(env, request, pathname) {
  const url = new URL(request.url)
  url.pathname = pathname
  return env.ASSETS_BINDING.fetch(new Request(url.toString(), request))
}

export default {
  async fetch(request, env) {
    const url = new URL(request.url)
    const pathname = stripDocsPrefix(url.pathname)

    return fetchAsset(env, request, pathname)
  },
}
