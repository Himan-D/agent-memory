const DOH_ENDPOINT = 'https://cloudflare-dns.com/dns-query'

const DNS_HOSTS = [
  'hystersis.com',
  'www.hystersis.com',
  'app.hystersis.com',
  'blogs.hystersis.com',
]

const HTTP_ENDPOINTS = [
  { name: 'Landing', url: 'https://hystersis.com/' },
  { name: 'Blog page', url: 'https://hystersis.com/blog' },
  { name: 'Docs', url: 'https://hystersis.com/docs' },
  { name: 'Blog subdomain', url: 'https://blogs.hystersis.com/' },
  { name: 'Dashboard', url: 'https://app.hystersis.com/auth/signin' },
]

export const CLOUDFLARE_MCP_STATUS = [
  { server: 'Cloudflare-docs', status: 'ready', note: 'Documentation search available' },
  { server: 'Cloudflare-builds', status: 'needsAuth', note: 'Authenticate in Cursor Desktop to inspect Workers Builds' },
  { server: 'Cloudflare-bindings', status: 'needsAuth', note: 'Authenticate in Cursor Desktop to inspect bindings' },
  { server: 'Cloudflare-observability', status: 'needsAuth', note: 'Authenticate in Cursor Desktop to view logs and metrics' },
]

async function queryDNS(hostname, type = 'A') {
  const res = await fetch(`${DOH_ENDPOINT}?name=${encodeURIComponent(hostname)}&type=${type}`, {
    headers: { Accept: 'application/dns-json' },
  })
  if (!res.ok) throw new Error(`DNS query failed (${res.status})`)
  const data = await res.json()
  return (data.Answer || []).map((a) => a.data)
}

export async function checkDNS(hostname) {
  try {
    let records = await queryDNS(hostname, 'A')
    if (!records.length) {
      records = await queryDNS(hostname, 'CNAME')
    }
    return {
      hostname,
      ok: records.length > 0,
      records,
      error: records.length ? null : 'No A or CNAME record',
    }
  } catch (err) {
    return { hostname, ok: false, records: [], error: err.message }
  }
}

export async function checkAllDNS() {
  return Promise.all(DNS_HOSTS.map(checkDNS))
}

export async function checkHTTP(name, url) {
  try {
    const res = await fetch(url, { method: 'HEAD', cache: 'no-store', mode: 'cors' })
    return {
      name,
      url,
      ok: res.status < 500,
      status: res.status,
      error: res.status >= 500 ? `HTTP ${res.status}` : null,
    }
  } catch (err) {
    return { name, url, ok: false, status: null, error: err.message }
  }
}

export async function checkAllHTTP() {
  return Promise.all(HTTP_ENDPOINTS.map((e) => checkHTTP(e.name, e.url)))
}

export async function runDeployDiagnostics() {
  const [dns, http] = await Promise.all([checkAllDNS(), checkAllHTTP()])
  const dnsOk = dns.filter((d) => d.ok).length
  const httpOk = http.filter((h) => h.ok).length
  return {
    dns,
    http,
    summary: {
      dnsOk,
      dnsTotal: dns.length,
      httpOk,
      httpTotal: http.length,
      healthy: dnsOk === dns.length && httpOk === http.length,
    },
    checkedAt: new Date().toISOString(),
  }
}
