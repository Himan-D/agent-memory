import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { getBlogBySlug, urlFor } from '../lib/sanity'
import { PortableText } from '@portabletext/react'
import { components } from '../components/RichTextComponents'

function BlogPost() {
  const { slug } = useParams()
  const [blog, setBlog] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    getBlogBySlug(slug)
      .then(data => {
        if (!data) setError('Blog post not found')
        else setBlog(data)
      })
      .catch(err => setError('Failed to load article'))
      .finally(() => setLoading(false))
  }, [slug])

  const formatDate = (dateStr) => {
    if (!dateStr) return ''
    return new Date(dateStr).toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' })
  }

  if (loading) {
    return (
      <div className="blog-post blog-post-loading">
        <div className="container">
          <div className="loading-spinner" />
          <p>Loading article...</p>
        </div>
      </div>
    )
  }

  if (error || !blog) {
    return (
      <div className="blog-post blog-post-error">
        <div className="container">
          <h1>Article not found</h1>
          <p>{error}</p>
          <Link to="/blog" className="btn btn-primary">Back to Blog</Link>
        </div>
      </div>
    )
  }

  return (
    <article className="blog-post">
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.5 }}
        className="blog-post-hero"
      >
        {blog.coverImage ? (
          <img src={urlFor(blog.coverImage).width(1200).height(500).fit('crop').url()} alt={blog.coverImage.alt || blog.title} />
        ) : (
          <div className="blog-post-hero-placeholder" />
        )}
        <div className="blog-post-overlay" />
        <div className="blog-post-header">
          <Link to="/blog" className="back-link">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M19 12H5M12 19l-7-7 7-7"/>
            </svg>
            Back to Blog
          </Link>
          <h1>{blog.title}</h1>
          <div className="blog-post-meta">
            <span className="blog-post-category">{blog.category}</span>
            <span className="blog-post-date">{formatDate(blog.publishedAt)}</span>
            <span className="blog-post-author">by {blog.author || 'Hystersis Team'}</span>
            <span className="blog-post-readtime">{blog.readTime || `${Math.round(blog.body?.length / 200 || 5)} min read`}</span>
          </div>
        </div>
      </motion.div>

      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.2 }}
          className="blog-post-content"
        >
          {blog.body && <PortableText value={blog.body} components={components} />}
        </motion.div>

        <div className="blog-post-footer">
          {blog.tags && blog.tags.length > 0 && (
            <div className="blog-post-tags">
              {blog.tags.map(tag => (
                <span key={tag} className="blog-post-tag">{tag}</span>
              ))}
            </div>
          )}
          <h3>More Articles</h3>
          <div className="related-blogs">
            {blog._relatedBlogs?.slice(0, 3).map((relatedBlog) => (
              <Link key={relatedBlog.slug.current} to={`/blog/${relatedBlog.slug.current}`} className="related-blog-card">
                <div className="related-blog-image">
                  {relatedBlog.coverImage ? (
                    <img src={urlFor(relatedBlog.coverImage).width(160).height(120).fit('crop').url()} alt={relatedBlog.title} />
                  ) : (
                    <div className="related-blog-placeholder" />
                  )}
                </div>
                <div>
                  <h4>{relatedBlog.title}</h4>
                  <span>{relatedBlog.readTime}</span>
                </div>
              </Link>
            ))}
          </div>
        </div>
      </div>

      <style>{`
        .blog-post {
          padding-bottom: 80px;
        }

        .blog-post-loading, .blog-post-error {
          padding-top: 120px;
          text-align: center;
        }

        .loading-spinner {
          width: 40px;
          height: 40px;
          border: 3px solid var(--border-light);
          border-top-color: var(--accent-primary);
          border-radius: 50%;
          animation: spin 1s linear infinite;
          margin: 0 auto 16px;
        }

        @keyframes spin {
          to { transform: rotate(360deg); }
        }

        .blog-post-hero {
          position: relative;
          height: 400px;
          overflow: hidden;
        }

        .blog-post-hero img {
          width: 100%;
          height: 100%;
          object-fit: cover;
        }

        .blog-post-hero-placeholder {
          width: 100%;
          height: 100%;
          background: linear-gradient(135deg, var(--bg-secondary), var(--bg-tertiary));
        }

        .blog-post-overlay {
          position: absolute;
          inset: 0;
          background: linear-gradient(to top, rgba(0,0,0,0.8), transparent 60%);
        }

        .blog-post-header {
          position: absolute;
          bottom: 0;
          left: 0;
          right: 0;
          padding: 40px;
          color: white;
        }

        .back-link {
          display: inline-flex;
          align-items: center;
          gap: 8px;
          color: rgba(255,255,255,0.8);
          font-size: 14px;
          margin-bottom: 16px;
          transition: color 0.2s ease;
        }

        .back-link:hover {
          color: white;
        }

        .blog-post-header h1 {
          font-family: var(--font-display);
          font-size: clamp(28px, 5vw, 42px);
          font-weight: 700;
          margin-bottom: 16px;
          max-width: 800px;
        }

        .blog-post-meta {
          display: flex;
          gap: 16px;
          font-size: 14px;
          flex-wrap: wrap;
          align-items: center;
        }

        .blog-post-meta span {
          color: rgba(255,255,255,0.8);
        }

        .blog-post-category {
          color: var(--accent-primary) !important;
          font-weight: 600;
          text-transform: uppercase;
          letter-spacing: 0.5px;
          font-size: 12px !important;
        }

        .blog-post-content {
          max-width: 720px;
          margin: 0 auto;
          padding: 48px 0;
        }

        .blog-post-content h1 {
          font-family: var(--font-display);
          font-size: 32px;
          font-weight: 700;
          margin: 40px 0 24px;
          color: var(--text-primary);
        }

        .blog-post-content h2 {
          font-family: var(--font-display);
          font-size: 24px;
          font-weight: 700;
          margin: 32px 0 16px;
          color: var(--text-primary);
        }

        .blog-post-content h3 {
          font-family: var(--font-display);
          font-size: 18px;
          font-weight: 600;
          margin: 24px 0 12px;
          color: var(--text-primary);
        }

        .blog-post-content h4 {
          font-family: var(--font-display);
          font-size: 16px;
          font-weight: 600;
          margin: 20px 0 12px;
          color: var(--text-primary);
        }

        .blog-post-content p {
          color: var(--text-secondary);
          line-height: 1.8;
          margin-bottom: 16px;
        }

        .blog-post-content ul, .blog-post-content ol {
          color: var(--text-secondary);
          line-height: 1.8;
          margin-bottom: 16px;
          padding-left: 24px;
        }

        .blog-post-content li {
          margin-bottom: 4px;
        }

        .blog-post-content blockquote {
          border-left: 4px solid var(--accent-primary);
          padding: 16px 24px;
          margin: 24px 0;
          background: var(--bg-surface);
          border-radius: 0 8px 8px 0;
          color: var(--text-secondary);
          font-style: italic;
        }

        .blog-post-content .blog-image {
          margin: 24px 0;
        }

        .blog-post-content .blog-image img {
          width: 100%;
          border-radius: 8px;
        }

        .blog-post-content .blog-image figcaption {
          text-align: center;
          font-size: 13px;
          color: var(--text-muted);
          margin-top: 8px;
        }

        .blog-post-content .blog-code {
          margin: 24px 0;
          border-radius: 8px;
          overflow: hidden;
        }

        .blog-post-content .blog-code-lang {
          display: block;
          padding: 8px 16px;
          background: #2d2d2d;
          color: #999;
          font-size: 12px;
          font-family: monospace;
          text-transform: uppercase;
        }

        .blog-post-content .blog-code pre {
          margin: 0;
          background: #1a1a1a;
          padding: 16px;
          overflow-x: auto;
        }

        .blog-post-content .blog-code code {
          font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
          font-size: 14px;
          color: #e8e8e8;
          line-height: 1.6;
        }

        .blog-post-content .inline-code {
          background: var(--bg-surface);
          padding: 2px 6px;
          border-radius: 4px;
          font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
          font-size: 0.9em;
          color: #e83e8c;
        }

        .blog-post-content a {
          color: var(--accent-primary);
          text-decoration: underline;
        }

        .blog-post-content table {
          width: 100%;
          border-collapse: collapse;
          margin: 24px 0;
        }

        .blog-post-content th, .blog-post-content td {
          padding: 12px;
          border: 1px solid var(--border-light);
          text-align: left;
        }

        .blog-post-content th {
          background: var(--bg-surface);
          font-weight: 600;
        }

        .blog-post-footer {
          max-width: 720px;
          margin: 0 auto;
          padding-top: 48px;
          border-top: 1px solid var(--border-light);
        }

        .blog-post-tags {
          display: flex;
          gap: 8px;
          flex-wrap: wrap;
          margin-bottom: 32px;
        }

        .blog-post-tag {
          font-size: 12px;
          color: var(--text-muted);
          background: var(--bg-surface);
          padding: 4px 12px;
          border-radius: 100px;
          border: 1px solid var(--border-light);
        }

        .blog-post-footer h3 {
          font-family: var(--font-display);
          font-size: 20px;
          margin-bottom: 24px;
        }

        .related-blogs {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
          gap: 20px;
        }

        .related-blog-card {
          display: flex;
          gap: 12px;
          text-decoration: none;
          color: var(--text-primary);
        }

        .related-blog-image {
          flex-shrink: 0;
          width: 80px;
          height: 60px;
          border-radius: 8px;
          overflow: hidden;
          background: var(--bg-secondary);
        }

        .related-blog-image img {
          width: 100%;
          height: 100%;
          object-fit: cover;
        }

        .related-blog-placeholder {
          width: 100%;
          height: 100%;
          background: linear-gradient(135deg, var(--bg-secondary), var(--bg-tertiary));
        }

        .related-blog-card h4 {
          font-size: 14px;
          font-weight: 600;
          margin-bottom: 4px;
          line-height: 1.3;
        }

        .related-blog-card span {
          font-size: 12px;
          color: var(--text-muted);
        }

        @media (max-width: 768px) {
          .blog-post-hero {
            height: 300px;
          }

          .blog-post-header {
            padding: 24px;
          }

          .blog-post-meta {
            gap: 8px;
          }
        }
      `}</style>
    </article>
  )
}

export default BlogPost
