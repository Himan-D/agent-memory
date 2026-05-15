import { createClient } from '@sanity/client'
import { blogs } from '../src/data/blogs.js'

// Sanity configuration
const client = createClient({
  projectId: '44liulah',
  dataset: 'production',
  apiVersion: '2025-05-15',
  token: 'sk60jF1KVBv9auTWfrEYFXZihuykWx3fn2AOkImc15t0LpotFgFYuPkqJT2YnwGRkMrQ5iElI9MJgCxufagUKRKaYKzn8fir7wPPxeoUgRRIqCKTNUAD03PKlhYGf4ynMmmbAybiWaygMquDBr25WOxPVULUYjywYxLWMW99ml8uPzfqgSUj',
  useCdn: false
})

/**
 * Convert markdown-like content to Portable Text blocks
 */
function convertToPortableText(markdown) {
  const lines = markdown.trim().split('\n')
  const blocks = []
  let inCodeBlock = false
  let codeContent = []
  let codeLanguage = ''
  let inTable = false
  let tableRows = []

  const flushCodeBlock = () => {
    if (codeContent.length > 0) {
      blocks.push({
        _type: 'code',
        code: codeContent.join('\n'),
        language: codeLanguage || 'text'
      })
      codeContent = []
      codeLanguage = ''
      inCodeBlock = false
    }
  }

  const flushTable = () => {
    if (tableRows.length > 0) {
      const headers = tableRows[0].split('|').map(h => h.trim()).filter(Boolean)
      const rows = tableRows.slice(2).map(row =>
        row.split('|').map(c => c.trim()).filter(Boolean)
      )

      let tableHtml = '<table>'
      tableHtml += '<thead><tr>' + headers.map(h => `<th>${h}</th>`).join('') + '</tr></thead>'
      tableHtml += '<tbody>'
      rows.forEach(row => {
        tableHtml += '<tr>' + row.map(c => `<td>${c}</td>`).join('') + '</tr>'
      })
      tableHtml += '</tbody></table>'

      blocks.push({
        _type: 'block',
        children: [{ _type: 'span', text: tableHtml }],
        style: 'normal',
        _key: 'table-' + blocks.length
      })
      tableRows = []
      inTable = false
    }
  }

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]

    // Code block detection
    if (line.startsWith('```')) {
      if (inCodeBlock) {
        flushCodeBlock()
      } else {
        flushTable()
        inCodeBlock = true
        codeLanguage = line.slice(3).trim() || 'text'
      }
      continue
    }

    if (inCodeBlock) {
      codeContent.push(line)
      continue
    }

    // Table detection
    if (line.startsWith('| ') && line.endsWith('|')) {
      if (!inTable) inTable = true
      tableRows.push(line)
      continue
    } else if (inTable) {
      flushTable()
    }

    // Skip separator lines (|---|---|)
    if (line.startsWith('|-') || line.startsWith('| -')) {
      continue
    }

    // Empty line
    if (line.trim() === '') {
      continue
    }

    // Headings
    if (line.startsWith('#### ')) {
      blocks.push({
        _type: 'block',
        children: [{ _type: 'span', marks: [], text: line.slice(5) }],
        style: 'h4',
        _key: 'h4-' + blocks.length
      })
    } else if (line.startsWith('### ')) {
      blocks.push({
        _type: 'block',
        children: [{ _type: 'span', marks: [], text: line.slice(4) }],
        style: 'h3',
        _key: 'h3-' + blocks.length
      })
    } else if (line.startsWith('## ')) {
      blocks.push({
        _type: 'block',
        children: [{ _type: 'span', marks: [], text: line.slice(3) }],
        style: 'h2',
        _key: 'h2-' + blocks.length
      })
    } else if (line.startsWith('# ')) {
      blocks.push({
        _type: 'block',
        children: [{ _type: 'span', marks: [], text: line.slice(2) }],
        style: 'h1',
        _key: 'h1-' + blocks.length
      })
    }
    // Bullet list
    else if (line.startsWith('- ') || line.startsWith('* ')) {
      blocks.push({
        _type: 'block',
        children: [{ _type: 'span', marks: [], text: line.slice(2) }],
        style: 'normal',
        listItem: 'bullet',
        _key: 'li-' + blocks.length
      })
    }
    // Numbered list
    else if (/^\d+\.\s/.test(line)) {
      blocks.push({
        _type: 'block',
        children: [{ _type: 'span', marks: [], text: line.replace(/^\d+\.\s/, '') }],
        style: 'normal',
        listItem: 'number',
        _key: 'li-' + blocks.length
      })
    }
    // Blockquote
    else if (line.startsWith('> ')) {
      blocks.push({
        _type: 'block',
        children: [{ _type: 'span', marks: [], text: line.slice(2) }],
        style: 'blockquote',
        _key: 'bq-' + blocks.length
      })
    }
    // Regular paragraph
    else {
      blocks.push({
        _type: 'block',
        children: [{ _type: 'span', marks: [], text: line }],
        style: 'normal',
        _key: 'p-' + blocks.length
      })
    }
  }

  // Flush any remaining code or table
  if (inCodeBlock) flushCodeBlock()
  if (inTable) flushTable()

  return blocks
}

/**
 * Parse date string to ISO format
 */
function parseDate(dateStr) {
  try {
    return new Date(dateStr).toISOString()
  } catch {
    return new Date().toISOString()
  }
}

/**
 * Main migration function
 */
async function migrate() {
  console.log('=== Hystersis Blog Migration ===')
  console.log(`Found ${blogs.length} blog posts to migrate\n`)

  let success = 0
  let skipped = 0
  let errors = 0

  for (const blog of blogs) {
    try {
      // Check if post already exists
      const existing = await client.fetch(
        `*[_type == "blogPost" && slug.current == $slug][0]`,
        { slug: blog.slug }
      )

      if (existing) {
        console.log(`⏭️  Skipped (exists): "${blog.title}"`)
        skipped++
        continue
      }

      const doc = {
        _type: 'blogPost',
        title: blog.title,
        slug: { _type: 'slug', current: blog.slug },
        excerpt: blog.excerpt,
        body: convertToPortableText(blog.content),
        category: blog.category,
        author: 'Hystersis Team',
        publishedAt: parseDate(blog.date),
        featured: blogs.indexOf(blog) < 3 // First 3 are featured
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
