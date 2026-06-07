import fs from 'fs'
import path from 'path'
import { fileURLToPath } from 'url'
import { blogs } from '../src/data/blogs.js'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const SITE_URL = 'https://hystersis.com'
const BLOG_SITE_URL = 'https://blogs.hystersis.com'
const PUBLIC_DIR = path.join(__dirname, '..', 'public')
const SITEMAP_PATH = path.join(PUBLIC_DIR, 'sitemap.xml')
const AGENTS_MD_SOURCE = path.join(__dirname, '..', '..', 'api', 'agents.md')
const AGENTS_MD_DEST = path.join(PUBLIC_DIR, 'agents.md')
const INSTALL_SH_SOURCE = path.join(__dirname, '..', '..', 'install.sh')
const INSTALL_SH_DEST = path.join(PUBLIC_DIR, 'install.sh')
const INSTALL_DEST = path.join(PUBLIC_DIR, 'install')
const TODAY = new Date().toISOString().split('T')[0]

const staticPages = [
  { loc: '/', changefreq: 'weekly', priority: '1.0' },
  { loc: '/use-cases', changefreq: 'weekly', priority: '0.9' },
  { loc: '/docs', changefreq: 'weekly', priority: '0.9' },
  { loc: '/for-agents', changefreq: 'weekly', priority: '0.9' },
  { loc: '/blog', changefreq: 'weekly', priority: '0.8' },
  { loc: '/demo', changefreq: 'monthly', priority: '0.8' },
  { loc: '/status', changefreq: 'daily', priority: '0.6' },
  { loc: '/agents.md', changefreq: 'monthly', priority: '0.5' },
  { loc: '/llms.txt', changefreq: 'monthly', priority: '0.5' },
]

function formatUrl(page) {
  return `
  <url>
    <loc>${SITE_URL}${page.loc}</loc>
    <lastmod>${TODAY}</lastmod>
    <changefreq>${page.changefreq}</changefreq>
    <priority>${page.priority}</priority>
  </url>`
}

const blogPages = blogs.flatMap((blog) => [
  {
    loc: `/blog/${blog.slug}`,
    changefreq: 'monthly',
    priority: '0.7',
    site: SITE_URL,
  },
  {
    loc: `/${blog.slug}`,
    changefreq: 'monthly',
    priority: '0.7',
    site: BLOG_SITE_URL,
  },
])

function formatUrlEntry(page) {
  const base = page.site || SITE_URL
  return `
  <url>
    <loc>${base}${page.loc}</loc>
    <lastmod>${TODAY}</lastmod>
    <changefreq>${page.changefreq}</changefreq>
    <priority>${page.priority}</priority>
  </url>`
}

const urls = [
  ...staticPages.map(formatUrl),
  ...blogPages.map(formatUrlEntry),
  formatUrlEntry({ loc: '/', changefreq: 'weekly', priority: '0.9', site: BLOG_SITE_URL }),
]

const sitemap = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">${urls.join('')}
</urlset>
`

fs.writeFileSync(SITEMAP_PATH, sitemap.trim() + '\n')

if (fs.existsSync(AGENTS_MD_SOURCE)) {
  const agentsMd = fs
    .readFileSync(AGENTS_MD_SOURCE, 'utf8')
    .replaceAll('https://hystersis.ai', 'https://hystersis.com')
    .replaceAll('https://docs.hystersis.ai', 'https://hystersis.com/docs')
    .replaceAll('https://dashboard.hystersis.ai', 'https://app.hystersis.com')
    .replaceAll('https://status.hystersis.ai', 'https://hystersis.com/status')
  fs.writeFileSync(AGENTS_MD_DEST, agentsMd)
}

if (fs.existsSync(INSTALL_SH_SOURCE)) {
  const installSh = fs.readFileSync(INSTALL_SH_SOURCE, 'utf8')
  fs.writeFileSync(INSTALL_SH_DEST, installSh, { mode: 0o755 })
  fs.writeFileSync(INSTALL_DEST, installSh, { mode: 0o755 })
  console.log('✓ install.sh synced to public/')
}

const count = staticPages.length + blogPages.length + 1
console.log(`✓ sitemap.xml generated with ${count} URLs`)
console.log(`✓ agents.md synced to public/`)
