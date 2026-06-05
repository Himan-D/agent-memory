import { createClient } from '@sanity/client'
import imageUrlBuilder from '@sanity/image-url'
import { PortableText } from '@portabletext/react'

const projectId = import.meta.env.VITE_SANITY_PROJECT_ID

const sanityClient = projectId
  ? createClient({
      projectId,
      dataset: 'production',
      apiVersion: '2025-05-15',
      useCdn: true,
    })
  : null

const builder = sanityClient ? imageUrlBuilder(sanityClient) : null

export function urlFor(source) {
  if (!builder || !source) {
    return { url: () => '' }
  }
  return builder.image(source)
}

export async function getFeaturedBlogs(limit = 3) {
  if (!sanityClient) return []
  return sanityClient.fetch(
    `
    *[_type == "blogPost" && publishedAt < now()]
    | order(publishedAt desc)
    [0...$limit] {
      _id, title, slug, excerpt, category, author, publishedAt, tags, featured,
      coverImage { ..., "url": asset->url },
      "readTime": "5 min read"
    }
  `,
    { limit },
  )
}

export async function getBlogs() {
  if (!sanityClient) return []
  return sanityClient.fetch(`
    *[_type == "blogPost" && publishedAt < now()]
    | order(publishedAt desc) {
      _id, title, slug, excerpt, category, author, publishedAt, tags,
      coverImage { ..., "url": asset->url },
      "readTime": "5 min read"
    }
  `)
}

export async function getBlogBySlug(slug) {
  if (!sanityClient) return null
  return sanityClient.fetch(
    `
    *[_type == "blogPost" && slug.current == $slug][0] {
      ...,
      coverImage { ..., "url": asset->url }
    }
  `,
    { slug },
  )
}

export async function getBlogCategories() {
  if (!sanityClient) return []
  return sanityClient.fetch(`
    *[_type == "blogPost" && publishedAt < now()] { category }
    | order(category asc)
  `)
}

export { PortableText }
