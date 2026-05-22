export const blogPostSchema = {
  name: 'blogPost',
  title: 'Blog Post',
  type: 'document',
  fields: [
    {
      name: 'title',
      title: 'Title',
      type: 'string',
      validation: Rule => Rule.required()
    },
    {
      name: 'slug',
      title: 'Slug',
      type: 'slug',
      options: { source: 'title', maxLength: 96 },
      validation: Rule => Rule.required()
    },
    {
      name: 'excerpt',
      title: 'Excerpt',
      type: 'text',
      rows: 3,
      validation: Rule => Rule.max(160).warning('Exceeds 160 characters — may be truncated in previews')
    },
    {
      name: 'body',
      title: 'Body',
      type: 'array',
      of: [
        { type: 'block' },
        {
          type: 'image',
          options: { hotspot: true },
          fields: [
            { name: 'caption', type: 'string', title: 'Caption' },
            { name: 'alt', type: 'string', title: 'Alt text' }
          ]
        },
        { type: 'code' }
      ]
    },
    {
      name: 'coverImage',
      title: 'Cover Image',
      type: 'image',
      options: { hotspot: true },
      fields: [
        { name: 'alt', type: 'string', title: 'Alt text', options: { isHighlighted: true } }
      ]
    },
    {
      name: 'category',
      title: 'Category',
      type: 'string',
      options: {
        list: [
          { title: 'Tutorial', value: 'Tutorial' },
          { title: 'Engineering', value: 'Engineering' },
          { title: 'Architecture', value: 'Architecture' },
          { title: 'News', value: 'News' }
        ],
        layout: 'dropdown'
      }
    },
    {
      name: 'tags',
      title: 'Tags',
      type: 'array',
      of: [{ type: 'string' }],
      options: { layout: 'tags' }
    },
    {
      name: 'author',
      title: 'Author',
      type: 'string',
      initialValue: 'Hystersis Team'
    },
    {
      name: 'publishedAt',
      title: 'Published At',
      type: 'datetime'
    },
    {
      name: 'featured',
      title: 'Featured on Homepage',
      type: 'boolean',
      initialValue: false
    }
  ],
  preview: {
    select: {
      title: 'title',
      media: 'coverImage',
      category: 'category',
      author: 'author',
      publishedAt: 'publishedAt'
    },
    prepare({ title, media, category, author, publishedAt }) {
      const date = publishedAt ? new Date(publishedAt).toLocaleDateString() : 'Draft'
      return {
        title,
        media,
        subtitle: `${category || 'Uncategorized'} · by ${author} · ${date}`
      }
    }
  }
}
