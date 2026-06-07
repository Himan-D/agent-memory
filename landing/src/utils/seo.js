import { isBlogSubdomain, BLOG_DOMAIN, MAIN_SITE_URL } from '../constants/blog'

const SITE_URL = 'https://hystersis.com'
const DEFAULT_OG_IMAGE = `${SITE_URL}/og-image.svg`
const SITE_NAME = 'Hystersis'

function upsertMeta(attr, key, content) {
  if (!content) return
  let el = document.querySelector(`meta[${attr}="${key}"]`)
  if (!el) {
    el = document.createElement('meta')
    el.setAttribute(attr, key)
    document.head.appendChild(el)
  }
  el.setAttribute('content', content)
}

function upsertLink(rel, href, extra = {}) {
  if (!href) return
  let el = document.querySelector(`link[rel="${rel}"]`)
  if (!el) {
    el = document.createElement('link')
    el.rel = rel
    document.head.appendChild(el)
  }
  el.href = href
  Object.entries(extra).forEach(([key, value]) => el.setAttribute(key, value))
}

function upsertJsonLd(id, data) {
  let el = document.getElementById(id)
  if (!data) {
    el?.remove()
    return
  }
  if (!el) {
    el = document.createElement('script')
    el.type = 'application/ld+json'
    el.id = id
    document.head.appendChild(el)
  }
  el.textContent = JSON.stringify(data)
}

export function canonicalUrl(path = '/') {
  const base = isBlogSubdomain() ? `https://${BLOG_DOMAIN}` : MAIN_SITE_URL
  if (!path || path === '/') return `${base}/`
  return `${base}${path.startsWith('/') ? path : `/${path}`}`
}

export function setSEO({
  title,
  description,
  path = '/',
  image = DEFAULT_OG_IMAGE,
  imageAlt = `${SITE_NAME} - Persistent memory for AI agents`,
  imageWidth = 1200,
  imageHeight = 630,
  type = 'website',
  noindex = false,
  keywords = [],
  article = null,
  jsonLd = null,
}) {
  const url = canonicalUrl(path)
  const fullTitle = title.includes(SITE_NAME) ? title : `${title} | ${SITE_NAME}`
  const keywordStr = Array.isArray(keywords) ? keywords.filter(Boolean).join(', ') : keywords

  document.title = fullTitle

  upsertMeta('name', 'title', fullTitle)
  upsertMeta('name', 'description', description)
  upsertMeta('name', 'robots', noindex ? 'noindex, nofollow' : 'index, follow, max-image-preview:large')
  if (keywordStr) upsertMeta('name', 'keywords', keywordStr)

  upsertMeta('property', 'og:type', type)
  upsertMeta('property', 'og:site_name', SITE_NAME)
  upsertMeta('property', 'og:locale', 'en_US')
  upsertMeta('property', 'og:title', fullTitle)
  upsertMeta('property', 'og:description', description)
  upsertMeta('property', 'og:url', url)
  upsertMeta('property', 'og:image', image)
  upsertMeta('property', 'og:image:secure_url', image.startsWith('http://') ? image.replace('http://', 'https://') : image)
  upsertMeta('property', 'og:image:width', String(imageWidth))
  upsertMeta('property', 'og:image:height', String(imageHeight))
  upsertMeta('property', 'og:image:alt', imageAlt)

  if (type === 'article' && article) {
    if (article.publishedTime) upsertMeta('property', 'article:published_time', article.publishedTime)
    if (article.modifiedTime) upsertMeta('property', 'article:modified_time', article.modifiedTime)
    if (article.section) upsertMeta('property', 'article:section', article.section)
    if (article.author) upsertMeta('property', 'article:author', article.author)
    if (article.tags?.length) {
      article.tags.forEach((tag) => {
        const existing = document.querySelector(`meta[property="article:tag"][content="${tag}"]`)
        if (!existing) {
          const el = document.createElement('meta')
          el.setAttribute('property', 'article:tag')
          el.setAttribute('content', tag)
          document.head.appendChild(el)
        }
      })
    }
  }

  upsertMeta('name', 'twitter:card', 'summary_large_image')
  upsertMeta('name', 'twitter:site', '@HHystersis')
  upsertMeta('name', 'twitter:creator', '@HHystersis')
  upsertMeta('name', 'twitter:title', fullTitle)
  upsertMeta('name', 'twitter:description', description)
  upsertMeta('name', 'twitter:url', url)
  upsertMeta('name', 'twitter:image', image)
  upsertMeta('name', 'twitter:image:alt', imageAlt)

  upsertLink('canonical', url)
  upsertJsonLd('page-jsonld', jsonLd)
}

export function articleJsonLd({
  title,
  description,
  path,
  image,
  datePublished,
  dateModified,
  keywords = [],
  section,
  author = SITE_NAME,
}) {
  const jsonLd = {
    '@context': 'https://schema.org',
    '@type': 'Article',
    headline: title,
    description,
    image: image || DEFAULT_OG_IMAGE,
    datePublished,
    dateModified: dateModified || datePublished,
    author: {
      '@type': 'Organization',
      name: author,
      url: SITE_URL,
    },
    publisher: {
      '@type': 'Organization',
      name: SITE_NAME,
      logo: {
        '@type': 'ImageObject',
        url: `${SITE_URL}/logo.svg`,
      },
    },
    mainEntityOfPage: {
      '@type': 'WebPage',
      '@id': canonicalUrl(path),
    },
  }

  if (section) jsonLd.articleSection = section
  if (keywords?.length) jsonLd.keywords = keywords.join(', ')

  return jsonLd
}

export { SITE_URL, DEFAULT_OG_IMAGE, SITE_NAME }
