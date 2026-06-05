import { createClient } from '@sanity/client'
import { createImageUrlBuilder } from '@sanity/image-url'

const projectId = import.meta.env.VITE_SANITY_PROJECT_ID

let sanityClient
let builder

if (projectId) {
  try {
    sanityClient = createClient({
      projectId,
      dataset: 'production',
      apiVersion: '2025-05-15',
      useCdn: true
    })
    builder = createImageUrlBuilder(sanityClient)
  } catch (e) {
    console.warn('Sanity client init failed:', e.message)
  }
}



export function urlFor(source) {
  if (!builder) return ''
  try {
    return builder.image(source)
  } catch (e) {
    return ''
  }
}

async function safeFetch(query, params = {}) {
  if (!sanityClient) return []
  try {
    return await sanityClient.fetch(query, params)
  } catch (e) {
    console.warn('Sanity fetch failed:', e.message)
    return []
  }
}

export async function getFeaturedBlogs(limit = 3) {
  return safeFetch(`
    *[_type == "blogPost" && publishedAt < now()]
    | order(publishedAt desc)
    [0...$limit] {
      _id, title, slug, excerpt, category, author, publishedAt, tags, featured,
      coverImage { ..., "url": asset->url },
      "readTime": "5 min read"
    }
  `, { limit })
}

export async function getBlogs() {
  return safeFetch(`
    *[_type == "blogPost" && publishedAt < now()]
    | order(publishedAt desc) {
      _id, title, slug, excerpt, category, author, publishedAt, tags,
      coverImage { ..., "url": asset->url },
      "readTime": "5 min read"
    }
  `)
}

export async function getBlogBySlug(slug) {
  return safeFetch(`
    *[_type == "blogPost" && slug.current == $slug][0] {
      ...,
      coverImage { ..., "url": asset->url }
    }
  `, { slug })
}

export async function getBlogCategories() {
  return safeFetch(`
    *[_type == "blogPost" && publishedAt < now()] { category }
    | order(category asc)
  `)
}
