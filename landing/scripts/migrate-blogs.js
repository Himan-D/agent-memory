import { createClient } from '@sanity/client'
import { markdownToPortableText } from '@portabletext/markdown'
import { blogs } from '../src/data/blogs.js'

const client = createClient({
  projectId: '44liulah',
  dataset: 'production',
  apiVersion: '2025-05-15',
  token: 'your_sanity_read_token_here',
  useCdn: false
})

function parseDate(dateStr) {
  try {
    return new Date(dateStr).toISOString()
  } catch {
    return new Date().toISOString()
  }
}

async function migrate() {
  console.log('=== Hystersis Blog Migration ===')
  console.log(`Found ${blogs.length} blog posts to migrate\n`)

  let success = 0
  let skipped = 0
  let errors = 0

  for (const blog of blogs) {
    try {
      const existing = await client.fetch(
        `*[_type == "blogPost" && slug.current == $slug][0]`,
        { slug: blog.slug }
      )

      if (existing) {
        console.log(`⏭️  Skipped (exists): "${blog.title}"`)
        skipped++
        continue
      }

      const body = markdownToPortableText(blog.content)

      const doc = {
        _type: 'blogPost',
        title: blog.title,
        slug: { _type: 'slug', current: blog.slug },
        excerpt: blog.excerpt,
        body,
        category: blog.category,
        author: 'Hystersis Team',
        publishedAt: parseDate(blog.date),
        featured: blogs.indexOf(blog) < 3
      }

      const result = await client.create(doc)
      console.log(`✅ Created: "${blog.title}" (ID: ${result._id.slice(0, 8)}...)`)
      success++
    } catch (err) {
      console.error(`❌ Failed: "${blog.title}" — ${err.message}`)
      errors++
    }
  }

  console.log('\n=== Migration Complete ===')
  console.log(`✅ Created: ${success}`)
  console.log(`⏭️  Skipped: ${skipped}`)
  console.log(`❌ Errors: ${errors}`)
  console.log(`📝 Total: ${blogs.length}`)
}

migrate().catch(err => {
  console.error('Migration failed:', err)
  process.exit(1)
})
