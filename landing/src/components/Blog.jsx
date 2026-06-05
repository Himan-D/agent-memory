import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { Link } from 'react-router-dom'
import { getFeaturedBlogs, urlFor } from '../lib/sanity'

function Blog() {
  const [blogs, setBlogs] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)

  useEffect(() => {
    getFeaturedBlogs(3)
      .then(data => setBlogs(data || []))
      .catch(err => {
        console.error('Failed to fetch featured blogs:', err)
        setError(true)
      })
      .finally(() => setLoading(false))
  }, [])

  const formatDate = (dateStr) => {
    if (!dateStr) return ''
    return new Date(dateStr).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
  }

  if (loading) return null

  if (error || !blogs || blogs.length === 0) {
    return (
      <section className="blog-section section" id="blog">
        <div className="container">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.6 }}
            className="section-header"
          >
            <span className="section-label">Blog</span>
            <h2 className="section-title">Latest insights</h2>
            <p className="section-description">
              Tutorials, guides, and engineering best practices for building memory-powered AI agents.
            </p>
          </motion.div>
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.6, delay: 0.2 }}
            className="blog-empty"
          >
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
              <path d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10l4 4v10a2 2 0 01-2 2z"/>
              <path d="M17 20v-8H7v8M7 4v4h8"/>
            </svg>
            <p>Blog posts coming soon.</p>
            <span>Follow us for updates on memory-powered AI.</span>
          </motion.div>
        </div>

        <style>{`
          .blog-section {
            background: var(--bg-surface);
          }
          .section-header {
            text-align: center;
            margin-bottom: 48px;
          }
          .section-title {
            font-family: var(--font-display);
            font-size: clamp(28px, 5vw, 40px);
            font-weight: 700;
            letter-spacing: -1px;
          }
          .section-description {
            font-size: 16px;
            color: var(--text-secondary);
            max-width: 500px;
            margin: 0 auto;
          }
          .blog-empty {
            text-align: center;
            padding: 48px 24px;
            border: 1px dashed var(--border-light);
            border-radius: 12px;
            max-width: 400px;
            margin: 0 auto;
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 12px;
            color: var(--text-secondary);
          }
          .blog-empty p {
            font-size: 16px;
            font-weight: 600;
            color: var(--text-primary);
            margin: 0;
          }
          .blog-empty span {
            font-size: 13px;
          }
          .blog-empty svg {
            opacity: 0.4;
          }
        `}</style>
      </section>
    )
  }

  return (
    <section className="blog-section section" id="blog">
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="section-header"
        >
          <span className="section-label">Blog</span>
          <h2 className="section-title">Latest insights</h2>
          <p className="section-description">
            Tutorials, guides, and engineering best practices for building memory-powered AI agents.
          </p>
        </motion.div>

        <motion.div
          initial={{ opacity: 0 }}
          whileInView={{ opacity: 1 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.2 }}
          className="blog-grid"
        >
          {blogs.map((blog, index) => (
            <motion.article
              key={blog._id}
              initial={{ opacity: 0, y: 30 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.5, delay: index * 0.1 }}
            >
              <Link to={`/blog/${blog.slug.current}`} className="blog-card">
                <div className="blog-image">
                  {blog.coverImage ? (
                    <img src={urlFor(blog.coverImage).width(600).height(375).fit('crop').url()} alt={blog.coverImage.alt || blog.title} />
                  ) : (
                    <div className="blog-image-placeholder" />
                  )}
                </div>
                <div className="blog-content">
                  <div className="blog-meta">
                    <span className="blog-category">{blog.category}</span>
                    <span className="blog-date">{formatDate(blog.publishedAt)}</span>
                  </div>
                  <h3 className="blog-title">{blog.title}</h3>
                  <p className="blog-excerpt">{blog.excerpt}</p>
                  <span className="blog-readtime">{blog.readTime}</span>
                </div>
              </Link>
            </motion.article>
          ))}
        </motion.div>

        <motion.div
          initial={{ opacity: 0 }}
          whileInView={{ opacity: 1 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.4 }}
          className="blog-cta"
        >
          <Link to="/blog" className="btn btn-secondary">View All Posts</Link>
        </motion.div>
      </div>

      <style>{`
        .blog-section {
          background: var(--bg-surface);
        }

        .section-header {
          text-align: center;
          margin-bottom: 48px;
        }

        .section-title {
          font-family: var(--font-display);
          font-size: clamp(28px, 5vw, 40px);
          font-weight: 700;
          letter-spacing: -1px;
        }

        .section-description {
          font-size: 16px;
          color: var(--text-secondary);
          max-width: 500px;
          margin: 0 auto;
        }

        .blog-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
          gap: 24px;
          margin-bottom: 32px;
        }

        .blog-card {
          display: block;
          background: var(--bg-primary);
          border: 1px solid var(--border-light);
          border-radius: 12px;
          overflow: hidden;
          transition: all 0.3s ease;
        }

        .blog-card:hover {
          transform: translateY(-4px);
          box-shadow: 0 12px 40px rgba(0, 0, 0, 0.08);
        }

        .blog-image {
          aspect-ratio: 16/10;
          overflow: hidden;
          background: var(--bg-secondary);
        }

        .blog-image img {
          width: 100%;
          height: 100%;
          object-fit: cover;
          transition: transform 0.5s ease;
        }

        .blog-image-placeholder {
          width: 100%;
          height: 100%;
          background: linear-gradient(135deg, var(--bg-secondary), var(--bg-tertiary));
        }

        .blog-card:hover .blog-image img {
          transform: scale(1.05);
        }

        .blog-content {
          padding: 20px;
        }

        .blog-meta {
          display: flex;
          gap: 12px;
          margin-bottom: 10px;
        }

        .blog-category {
          font-size: 11px;
          font-weight: 600;
          color: var(--accent-primary);
          text-transform: uppercase;
          letter-spacing: 0.5px;
        }

        .blog-date {
          font-size: 11px;
          color: var(--text-muted);
        }

        .blog-title {
          font-family: var(--font-display);
          font-size: 16px;
          font-weight: 600;
          margin-bottom: 8px;
          line-height: 1.4;
        }

        .blog-excerpt {
          font-size: 13px;
          color: var(--text-secondary);
          line-height: 1.5;
          margin-bottom: 10px;
          display: -webkit-box;
          -webkit-line-clamp: 2;
          -webkit-box-orient: vertical;
          overflow: hidden;
        }

        .blog-readtime {
          font-size: 11px;
          color: var(--text-muted);
        }

        .blog-cta {
          text-align: center;
        }

        @media (max-width: 768px) {
          .blog-grid {
            grid-template-columns: 1fr;
          }
        }
      `}</style>
    </section>
  )
}

export default Blog
