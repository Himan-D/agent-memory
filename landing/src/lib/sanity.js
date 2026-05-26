import { createClient } from '@sanity/client'
import imageUrlBuilder from '@sanity/image-url'
import { PortableText } from '@portabletext/react'

export const sanityClient = createClient({
  projectId: import.meta.env.VITE_SANITY_PROJECT_ID,
  dataset: 'production',
  apiVersion: '2025-05-15',
  useCdn: true
})

const builder = imageUrlBuilder(sanityClient)
export function urlFor(source) {
  return builder.image(source)
}

export async function getFeaturedBlogs(limit = 3) {
  return sanityClient.fetch(`
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
  return sanityClient.fetch(`
    *[_type == "blogPost" && slug.current == $slug][0] {
      ...,
      coverImage { ..., "url": asset->url }
    }
  `, { slug })
}

export async function getBlogCategories() {
  return sanityClient.fetch(`
    *[_type == "blogPost" && publishedAt < now()] { category }
    | order(category asc)
  `)
}

export { PortableText }
