import { defineConfig } from 'sanity'
import { structureTool } from 'sanity/structure'
import { visionTool } from '@sanity/vision'
import { codeInput } from '@sanity/code-input'
import { blogPostSchema } from './schemaTypes/blogPost'

export default defineConfig({
  name: 'hystersis-blog',
  title: 'Hystersis Blog',
  basePath: '/',
  projectId: process.env.SANITY_STUDIO_PROJECT_ID || process.env.VITE_SANITY_PROJECT_ID || 'yhvdqwt4',
  dataset: 'production',
  plugins: [structureTool(), visionTool(), codeInput()],
  schema: {
    types: [blogPostSchema]
  }
})
