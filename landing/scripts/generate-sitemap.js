import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const SITE_URL = 'https://hystersis.ai';
const PUBLIC_DIR = path.join(__dirname, '..', 'public');
const SITEMAP_PATH = path.join(PUBLIC_DIR, 'sitemap.xml');
const TODAY = new Date().toISOString().split('T')[0];

const blogsModule = fs.readFileSync(
  path.join(__dirname, '..', 'src', 'data', 'blogs.js'),
  'utf-8'
);

const slugMatches = blogsModule.matchAll(/slug:\s*'([^']+)'/g);
const blogSlugs = [...slugMatches].map((m) => m[1]);

const staticPages = [
  { loc: '/', changefreq: 'weekly', priority: '1.0' },
  { loc: '/use-cases', changefreq: 'weekly', priority: '0.9' },
  { loc: '/docs', changefreq: 'weekly', priority: '0.8' },
  { loc: '/blog', changefreq: 'weekly', priority: '0.8' },
  { loc: '/demo', changefreq: 'monthly', priority: '0.7' },
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

for (const slug of blogSlugs) {
  urls.push(`
  <url>
    <loc>${SITE_URL}/blog/${slug}</loc>
    <lastmod>${TODAY}</lastmod>
    <changefreq>monthly</changefreq>
    <priority>0.6</priority>
  </url>`);
}

const sitemap = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">${urls.join('')}
</urlset>
`;

fs.writeFileSync(SITEMAP_PATH, sitemap.trim() + '\n');
const count = staticPages.length + hashSections.length + blogSlugs.length;
console.log(`✓ sitemap.xml generated with ${count} URLs (${blogSlugs.length} blog posts)`);
