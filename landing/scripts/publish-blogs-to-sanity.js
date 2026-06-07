/**
 * Publish detailed blog posts to Sanity with cover + inline images.
 *
 * Usage (from landing/):
 *   set -a && source .env.local && set +a
 *   node scripts/publish-blogs-to-sanity.js
 *
 * Requires SANITY_AUTH_TOKEN with Editor role.
 */
import { createClient } from '@sanity/client'
import { markdownToPortableText } from '@portabletext/markdown'
import fs from 'fs'
import https from 'https'
import http from 'http'
import path from 'path'
import { fileURLToPath } from 'url'
import { blogs } from '../src/data/blogs.js'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const TMP_DIR = path.join(__dirname, '../.tmp/blog-images')

const client = createClient({
  projectId: process.env.VITE_SANITY_PROJECT_ID,
  dataset: 'production',
  apiVersion: '2025-05-15',
  token: process.env.SANITY_AUTH_TOKEN || process.env.VITE_SANITY_READ_TOKEN,
  useCdn: false,
})

const IMAGE_RE = /!\[([^\]]*)\]\(([^)]+)\)/g

function parseDate(dateStr) {
  try {
    return new Date(dateStr).toISOString()
  } catch {
    return new Date().toISOString()
  }
}

function randomKey() {
  return Math.random().toString(36).slice(2, 14)
}

function downloadImage(url, filepath) {
  return new Promise((resolve, reject) => {
    const proto = url.startsWith('https') ? https : http
    const file = fs.createWriteStream(filepath)
    proto.get(url, (response) => {
      if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
        file.close()
        fs.unlinkSync(filepath)
        return downloadImage(response.headers.location, filepath).then(resolve).catch(reject)
      }
      if (response.statusCode !== 200) {
        file.close()
        return reject(new Error(`HTTP ${response.statusCode} for ${url}`))
      }
      response.pipe(file)
      file.on('finish', () => {
        file.close()
        resolve(filepath)
      })
    }).on('error', (err) => {
      file.close()
      reject(err)
    })
  })
}

const uploadCache = new Map()

async function uploadImageFromUrl(url, label) {
  if (uploadCache.has(url)) return uploadCache.get(url)

  fs.mkdirSync(TMP_DIR, { recursive: true })
  const ext = path.extname(new URL(url).pathname) || '.jpg'
  const safeExt = ext.length <= 5 ? ext : '.jpg'
  const filepath = path.join(TMP_DIR, `${label}${safeExt}`)

  await downloadImage(url, filepath)
  const asset = await client.assets.upload('image', fs.createReadStream(filepath), {
    filename: `${label}${safeExt}`,
  })
  fs.unlinkSync(filepath)

  uploadCache.set(url, asset._id)
  return asset._id
}

async function markdownToBlocksWithImages(markdown, slug) {
  const blocks = []
  let lastIndex = 0
  let imageIndex = 0
  let match

  while ((match = IMAGE_RE.exec(markdown)) !== null) {
    const textBefore = markdown.slice(lastIndex, match.index)
    if (textBefore.trim()) {
      blocks.push(...markdownToPortableText(textBefore))
    }

    const alt = match[1] || ''
    const imageUrl = match[2]
    const assetId = await uploadImageFromUrl(imageUrl, `${slug}-inline-${imageIndex++}`)
    blocks.push({
      _key: randomKey(),
      _type: 'image',
      asset: { _type: 'reference', _ref: assetId },
      alt,
    })

    lastIndex = match.index + match[0].length
  }

  const remainder = markdown.slice(lastIndex)
  if (remainder.trim()) {
    blocks.push(...markdownToPortableText(remainder))
  }

  return blocks
}

async function publishBlog(blog, index) {
  const existing = await client.fetch(
    `*[_type == "blogPost" && slug.current == $slug][0]{_id}`,
    { slug: blog.slug },
  )

  console.log(`\n📝 Publishing: ${blog.title}`)

  const coverAssetId = await uploadImageFromUrl(blog.image, `${blog.slug}-cover`)
  const body = await markdownToBlocksWithImages(blog.content.trim(), blog.slug)

  const doc = {
    _type: 'blogPost',
    title: blog.title,
    slug: { _type: 'slug', current: blog.slug },
    excerpt: blog.excerpt,
    body,
    category: blog.category,
    author: blog.author || 'Hystersis Team',
    publishedAt: parseDate(blog.date),
    featured: blog.featured ?? index < 3,
    tags: blog.tags || [],
    coverImage: {
      _type: 'image',
      asset: { _type: 'reference', _ref: coverAssetId },
      alt: blog.title,
    },
  }

  if (existing?._id) {
    await client.patch(existing._id).set(doc).commit()
    console.log(`   ✅ Updated ${existing._id} (${body.length} blocks, cover image set)`)
    return 'updated'
  }

  const created = await client.create(doc)
  console.log(`   ✅ Created ${created._id} (${body.length} blocks)`)
  return 'created'
}

async function main() {
  if (!process.env.VITE_SANITY_PROJECT_ID) {
    throw new Error('VITE_SANITY_PROJECT_ID not set')
  }
  if (!process.env.SANITY_AUTH_TOKEN && !process.env.VITE_SANITY_READ_TOKEN) {
    throw new Error('SANITY_AUTH_TOKEN (editor) required for uploads')
  }

  console.log('=== Publish blogs to Sanity ===')
  console.log(`Project: ${process.env.VITE_SANITY_PROJECT_ID}`)
  console.log(`Posts: ${blogs.length}`)

  let created = 0
  let updated = 0
  let errors = 0

  for (let i = 0; i < blogs.length; i++) {
    try {
      const result = await publishBlog(blogs[i], i)
      if (result === 'created') created++
      else updated++
    } catch (err) {
      console.error(`   ❌ ${blogs[i].slug}: ${err.message}`)
      errors++
    }
  }

  if (fs.existsSync(TMP_DIR)) {
    fs.rmSync(TMP_DIR, { recursive: true, force: true })
  }

  console.log('\n=== Done ===')
  console.log(`Created: ${created} | Updated: ${updated} | Errors: ${errors}`)

  const verify = await client.fetch(
    `*[_type == "blogPost"] | order(publishedAt desc) {
      title, "slug": slug.current,
      "hasCover": defined(coverImage.asset),
      "bodyBlocks": count(body),
      "inlineImages": count(body[_type == "image"])
    }`,
  )
  console.log('\nSanity contents:')
  for (const p of verify) {
    console.log(`  • ${p.slug}: cover=${p.hasCover} blocks=${p.bodyBlocks} inlineImages=${p.inlineImages}`)
  }

  if (errors) process.exit(1)
}

main().catch((err) => {
  console.error('Publish failed:', err)
  process.exit(1)
})
