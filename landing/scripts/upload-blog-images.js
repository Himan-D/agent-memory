import { createClient } from '@sanity/client'
import fs from 'fs'
import https from 'https'
import path from 'path'
import { blogs } from '../src/data/blogs.js'

const client = createClient({
  projectId: process.env.VITE_SANITY_PROJECT_ID || 'yhvdqwt4',
  dataset: 'production',
  apiVersion: '2025-05-15',
  token: process.env.VITE_SANITY_READ_TOKEN,
  useCdn: false
})

function downloadImage(url, filepath) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(filepath)
    https.get(url, (response) => {
      response.pipe(file)
      file.on('finish', () => {
        file.close()
        resolve(filepath)
      })
    }).on('error', reject)
  })
}

async function uploadAndPatch() {
  console.log('=== Blog Image Upload ===\n')

  const tmpDir = '/tmp/blog-images'
  fs.mkdirSync(tmpDir, { recursive: true })

  for (const blog of blogs) {
    const { slug, image } = blog

    // Find the post by slug
    const post = await client.fetch(
      `*[_type == "blogPost" && slug.current == $slug][0]{_id, title}`,
      { slug }
    )

    if (!post) {
      console.log(`⏭️  Post not found: ${slug}`)
      continue
    }

    // Check if already has cover image
    const existing = await client.fetch(
      `*[_type == "blogPost" && _id == $id][0]{coverImage}`,
      { id: post._id }
    )
    if (existing?.coverImage) {
      console.log(`✅ "${post.title}" already has image`)
      continue
    }

    // Download image
    const filename = `${slug}.jpg`
    const filepath = path.join(tmpDir, filename)
    console.log(`📥 Downloading: ${slug}`)
    await downloadImage(image, filepath)

    // Upload to Sanity
    console.log(`📤 Uploading: ${filename}`)
    const doc = await client.assets.upload('image', fs.createReadStream(filepath))

    // Patch the post with the image reference
    await client.patch(post._id).set({
      coverImage: {
        _type: 'image',
        asset: { _type: 'reference', _ref: doc._id },
        alt: blog.title
      }
    }).commit()

    console.log(`✅ "${post.title}" patched with image`)
    console.log()
  }

  // Cleanup
  fs.rmSync(tmpDir, { recursive: true })
  console.log('=== All images uploaded ===')
}

uploadAndPatch().catch(err => {
  console.error('Failed:', err)
  process.exit(1)
})
