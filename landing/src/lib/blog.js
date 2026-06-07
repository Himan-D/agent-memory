import { markdownToPortableText } from '@portabletext/markdown'
import { createClient } from '@sanity/client'
import imageUrlBuilder from '@sanity/image-url'
import { PortableText } from '@portabletext/react'
import { blogs as staticBlogs } from '../data/blogs.js'

export const SANITY_PROJECT_ID = import.meta.env.VITE_SANITY_PROJECT_ID || 'yhvdqwt4'
const SANITY_DATASET = 'production'

const sanityClient = createClient({
  projectId: SANITY_PROJECT_ID,
  dataset: SANITY_DATASET,
  apiVersion: '2025-05-15',
  useCdn: true,
})

const builder = imageUrlBuilder(sanityClient)

function parseStaticDate(dateStr) {
  try {
    return new Date(dateStr).toISOString()
  } catch {
    return new Date().toISOString()
  }
}

function staticToSanityFormat(blog, index) {
  return {
    _id: `static-${blog.slug}`,
    _source: 'static',
    title: blog.title,
    slug: { current: blog.slug },
    excerpt: blog.excerpt,
    category: blog.category,
    author: blog.author || 'Hystersis Team',
    publishedAt: parseStaticDate(blog.date),
    tags: blog.tags || [],
    featured: index < 3,
    readTime: blog.readTime || '5 min read',
    coverImage: blog.image
      ? { _type: 'image', asset: { _ref: 'external', url: blog.image }, alt: blog.title }
      : null,
    body: markdownToPortableText(blog.content.trim()),
  }
}

const staticPosts = staticBlogs.map(staticToSanityFormat)

function isPublished(post) {
  if (!post?.publishedAt) return true
  return new Date(post.publishedAt) <= new Date()
}

function sortByDate(posts) {
  return [...posts].sort(
    (a, b) => new Date(b.publishedAt || 0) - new Date(a.publishedAt || 0),
  )
}

async function fetchSanityPosts() {
  return sanityClient.fetch(`
    *[_type == "blogPost" && publishedAt < now()]
    | order(publishedAt desc) {
      _id, title, slug, excerpt, category, author, publishedAt, tags, featured, body,
      coverImage { ..., "url": asset->url },
      "readTime": "5 min read"
    }
  `)
}

function mergePosts(sanityPosts) {
  const sanitySlugs = new Set(
    (sanityPosts || []).map((p) => p.slug?.current).filter(Boolean),
  )
  const merged = [
    ...(sanityPosts || []).map((p) => ({ ...p, _source: 'sanity' })),
    ...staticPosts.filter((p) => !sanitySlugs.has(p.slug.current)),
  ]
  return sortByDate(merged.filter(isPublished))
}

export function urlFor(source) {
  if (!source) return { url: () => '' }
  if (source.asset?.url) {
    const url = source.asset.url
    return {
      width: () => ({ height: () => ({ fit: () => ({ url: () => url }) }) }),
      height: () => ({ fit: () => ({ url: () => url }) }),
      fit: () => ({ url: () => url }),
      url: () => url,
    }
  }
  if (source.url) {
    const url = source.url
    return {
      width: () => ({ height: () => ({ fit: () => ({ url: () => url }) }) }),
      height: () => ({ fit: () => ({ url: () => url }) }),
      fit: () => ({ url: () => url }),
      url: () => url,
    }
  }
  return builder.image(source)
}

export function getCoverImageUrl(coverImage, { width = 600, height = 375 } = {}) {
  if (!coverImage) return null
  if (coverImage.asset?.url) return coverImage.asset.url
  if (coverImage.url) return coverImage.url
  return urlFor(coverImage).width(width).height(height).fit('crop').url()
}

export async function getFeaturedBlogs(limit = 3) {
  const posts = await getBlogs()
  return posts.filter((p) => p.featured).slice(0, limit).length
    ? posts.filter((p) => p.featured).slice(0, limit)
    : posts.slice(0, limit)
}

export async function getBlogs() {
  try {
    const sanityPosts = await fetchSanityPosts()
    if (sanityPosts?.length) return mergePosts(sanityPosts)
  } catch (err) {
    console.warn('Sanity fetch failed, using static blogs:', err)
  }
  return staticPosts
}

export async function getBlogBySlug(slug) {
  try {
    const sanityPost = await sanityClient.fetch(
      `*[_type == "blogPost" && slug.current == $slug][0] {
        ..., coverImage { ..., "url": asset->url }
      }`,
      { slug },
    )
    if (sanityPost) {
      const all = await getBlogs()
      return {
        ...sanityPost,
        _source: 'sanity',
        readTime: sanityPost.readTime || '5 min read',
        _relatedBlogs: all
          .filter((p) => p.slug?.current !== slug)
          .slice(0, 3)
          .map((p) => ({
            slug: p.slug,
            title: p.title,
            readTime: p.readTime,
            coverImage: p.coverImage,
          })),
      }
    }
  } catch (err) {
    console.warn('Sanity slug fetch failed:', err)
  }

  const post = staticPosts.find((p) => p.slug.current === slug)
  if (!post) return null

  return {
    ...post,
    _relatedBlogs: staticPosts
      .filter((p) => p.slug.current !== slug)
      .slice(0, 3)
      .map((p) => ({
        slug: p.slug,
        title: p.title,
        readTime: p.readTime,
        coverImage: p.coverImage,
      })),
  }
}

export async function getBlogCategories() {
  const posts = await getBlogs()
  return [...new Set(posts.map((p) => p.category).filter(Boolean))]
}

export { PortableText }
