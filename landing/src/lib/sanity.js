import { createClient } from '@sanity/client'
import imageUrlBuilder from '@sanity/image-url'
import { PortableText } from '@portabletext/react'

export const sanityClient = createClient({
  projectId: '44liulah',
  dataset: 'production',
  apiVersion: '2025-05-15',
  useCdn: true,
  token: import.meta.env.VITE_SANITY_READ_TOKEN,
  ignoreBrowserTokenWarning: true
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
      "coverImageUrl": coverImage.asset->url,
      "readTime": round(length(pt::text(body)) / 200) + " min read"
    }
  `, { limit })
}

export async function getBlogs() {
  return sanityClient.fetch(`
    *[_type == "blogPost" && publishedAt < now()]
    | order(publishedAt desc) {
      _id, title, slug, excerpt, category, author, publishedAt, tags,
      "coverImageUrl": coverImage.asset->url,
      "readTime": round(length(pt::text(body)) / 200) + " min read"
    }
  `)
}

export async function getBlogBySlug(slug) {
  return sanityClient.fetch(`
    *[_type == "blogPost" && slug.current == $slug][0] {
      ...,
      "coverImageUrl": coverImage.asset->url
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
