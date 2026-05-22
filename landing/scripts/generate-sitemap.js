import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const SITE_URL = 'https://hystersis.ai';
const PUBLIC_DIR = path.join(__dirname, '..', 'public');
const SITEMAP_PATH = path.join(PUBLIC_DIR, 'sitemap.xml');
const TODAY = new Date().toISOString().split('T')[0];

const staticPages = [
  { loc: '/', changefreq: 'weekly', priority: '1.0' },
  { loc: '/use-cases', changefreq: 'weekly', priority: '0.9' },
  { loc: '/docs', changefreq: 'weekly', priority: '0.8' },
  { loc: '/blog', changefreq: 'weekly', priority: '0.8' },
  { loc: '/for-agents', changefreq: 'monthly', priority: '0.8' },
  { loc: '/demo', changefreq: 'monthly', priority: '0.7' },
  { loc: '/agents.md', changefreq: 'monthly', priority: '0.5' },
  { loc: '/llms.txt', changefreq: 'monthly', priority: '0.5' },
];

const hashSections = [
  { loc: '/#features', changefreq: 'monthly', priority: '0.8' },
  { loc: '/#pricing', changefreq: 'monthly', priority: '0.8' },
  { loc: '/#demo', changefreq: 'monthly', priority: '0.7' },
];

const urls = [];

for (const page of staticPages) {
  urls.push(`
  <url>
    <loc>${SITE_URL}${page.loc}</loc>
    <lastmod>${TODAY}</lastmod>
    <changefreq>${page.changefreq}</changefreq>
    <priority>${page.priority}</priority>
  </url>`);
}

for (const hash of hashSections) {
  urls.push(`
  <url>
    <loc>${SITE_URL}${hash.loc}</loc>
    <lastmod>${TODAY}</lastmod>
    <changefreq>${hash.changefreq}</changefreq>
    <priority>${hash.priority}</priority>
  </url>`);
}

const sitemap = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">${urls.join('')}
</urlset>
`;

fs.writeFileSync(SITEMAP_PATH, sitemap.trim() + '\n');
const count = staticPages.length + hashSections.length;
console.log(`✓ sitemap.xml generated with ${count} URLs`);
