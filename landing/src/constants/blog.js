export const BLOG_DOMAIN = 'blogs.hystersis.com'
export const MAIN_SITE_URL = 'https://hystersis.com'

export function isBlogSubdomain() {
  if (typeof window === 'undefined') return false
  return window.location.hostname === BLOG_DOMAIN
}

export function blogListPath() {
  return isBlogSubdomain() ? '/' : '/blog'
}

export function blogPostPath(slug) {
  if (!slug) return blogListPath()
  return isBlogSubdomain() ? `/${slug}` : `/blog/${slug}`
}

export function blogCanonicalPath(slug) {
  return slug ? `/blog/${slug}` : '/blog'
}

export function siteBaseUrl() {
  return isBlogSubdomain() ? `https://${BLOG_DOMAIN}` : MAIN_SITE_URL
}
