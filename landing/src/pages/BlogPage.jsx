import { useState, useEffect, useMemo } from 'react'
import { motion } from 'framer-motion'
import { Link } from 'react-router-dom'
import { getBlogs, getCoverImageUrl } from '../lib/blog'
import { blogPostPath } from '../constants/blog'

function BlogPage() {
  const [blogs, setBlogs] = useState([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [activeCategory, setActiveCategory] = useState('All')

  useEffect(() => {
    getBlogs()
      .then(data => setBlogs(data))
      .catch(err => console.error('Failed to fetch blogs:', err))
      .finally(() => setLoading(false))
  }, [])

  const categories = useMemo(() => {
    const cats = new Set(blogs.map(b => b.category).filter(Boolean))
    return ['All', ...Array.from(cats)]
  }, [blogs])

  const filteredBlogs = useMemo(() => {
    return blogs.filter(blog => {
      const matchesCategory = activeCategory === 'All' || blog.category === activeCategory
      const matchesSearch = !searchQuery || 
        blog.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
        blog.excerpt?.toLowerCase().includes(searchQuery.toLowerCase()) ||
        blog.tags?.some(t => t.toLowerCase().includes(searchQuery.toLowerCase()))
      return matchesCategory && matchesSearch
    })
  }, [blogs, activeCategory, searchQuery])

  const formatDate = (dateStr) => {
    if (!dateStr) return ''
    return new Date(dateStr).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
  }

  if (loading) {
    return (
      <div className="blog-page">
        <div className="page-hero">
          <div className="container">
            <span className="section-label">Blog</span>
            <h1>Loading...</h1>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="blog-page">
      <motion.div 
        className="page-hero"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.5 }}
      >
        <div className="container">
          <span className="section-label">Blog</span>
          <h1>Latest insights & tutorials</h1>
          <p>Learn how to build memory-powered AI agents with our tutorials, guides, and engineering best practices.</p>
        </div>
      </motion.div>

      <div className="container">
        {/* Search & Filter */}
        <motion.div
          className="blog-filters"
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.1 }}
        >
          <div className="search-bar">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>
            </svg>
            <input
              type="text"
              placeholder="Search articles..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>
          <div className="category-tabs">
            {categories.map(cat => (
              <button
                key={cat}
                className={`category-tab ${activeCategory === cat ? 'active' : ''}`}
                onClick={() => setActiveCategory(cat)}
              >
                {cat}
              </button>
            ))}
          </div>
        </motion.div>

        {/* Blog Grid */}
        <motion.div 
          className="blog-grid"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.2 }}
        >
          {filteredBlogs.length === 0 ? (
            <div className="no-results">
              <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                <circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/><path d="M8 11h6"/>
              </svg>
              <h3>No articles found</h3>
              <p>Try adjusting your search or filter criteria.</p>
            </div>
          ) : (
            filteredBlogs.map((blog, index) => (
              <motion.article
                key={blog._id}
                initial={{ opacity: 0, y: 30 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5, delay: 0.05 * index }}
              >
                <Link to={blogPostPath(blog.slug.current)} className="blog-card">
                  <div className="blog-image">
                    {blog.coverImage ? (
                      <img src={getCoverImageUrl(blog.coverImage)} alt={blog.coverImage.alt || blog.title} />
                    ) : (
                      <div className="blog-image-placeholder" />
                    )}
                  </div>
                  <div className="blog-content">
                    <div className="blog-meta">
                      <span className="blog-category">{blog.category}</span>
                      <span className="blog-date">{formatDate(blog.publishedAt)}</span>
                    </div>
                    <h3>{blog.title}</h3>
                    <p>{blog.excerpt}</p>
                    {blog.tags && blog.tags.length > 0 && (
                      <div className="blog-tags">
                        {blog.tags.slice(0, 3).map(tag => (
                          <span key={tag} className="blog-tag">{tag}</span>
                        ))}
                      </div>
                    )}
                    <div className="blog-footer">
                      <span className="blog-readtime">{blog.readTime}</span>
                      <span className="read-more">
                        Read more
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                          <path d="M5 12h14M12 5l7 7-7 7"/>
                        </svg>
                      </span>
                    </div>
                  </div>
                </Link>
              </motion.article>
            ))
          )}
        </motion.div>
      </div>

      <style>{`
        .blog-page {
          padding-bottom: 80px;
        }

        .page-hero {
          padding: 80px 0 60px;
          text-align: center;
          border-bottom: 1px solid var(--border-light);
        }

        .page-hero h1 {
          font-family: var(--font-display);
          font-size: clamp(36px, 6vw, 56px);
          font-weight: 800;
          margin-bottom: 16px;
          letter-spacing: -2px;
        }

        .page-hero p {
          font-size: 18px;
          color: var(--text-secondary);
          max-width: 500px;
          margin: 0 auto;
        }

        .blog-filters {
          padding: 32px 0 16px;
        }

        .search-bar {
          display: flex;
          align-items: center;
          gap: 12px;
          padding: 12px 16px;
          background: var(--bg-surface);
          border: 1px solid var(--border-light);
          border-radius: 12px;
          margin-bottom: 20px;
          max-width: 400px;
        }

        .search-bar svg {
          color: var(--text-muted);
          flex-shrink: 0;
        }

        .search-bar input {
          border: none;
          background: transparent;
          font-size: 15px;
          color: var(--text-primary);
          width: 100%;
          outline: none;
        }

        .search-bar input::placeholder {
          color: var(--text-muted);
        }

        .category-tabs {
          display: flex;
          gap: 8px;
          flex-wrap: wrap;
        }

        .category-tab {
          padding: 8px 16px;
          font-size: 13px;
          font-weight: 500;
          color: var(--text-secondary);
          background: transparent;
          border: 1px solid var(--border-light);
          border-radius: 100px;
          cursor: pointer;
          transition: all 0.2s;
        }

        .category-tab:hover {
          border-color: var(--accent-primary);
          color: var(--text-primary);
        }

        .category-tab.active {
          background: var(--accent-primary);
          border-color: var(--accent-primary);
          color: white;
        }

        .blog-grid {
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
          gap: 32px;
          padding: 32px 0;
        }

        .blog-card {
          display: block;
          background: var(--bg-surface);
          border: 1px solid var(--border-light);
          border-radius: 16px;
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
          padding: 24px;
        }

        .blog-meta {
          display: flex;
          gap: 12px;
          margin-bottom: 12px;
        }

        .blog-category {
          font-size: 12px;
          font-weight: 600;
          color: #2563EB;
          text-transform: uppercase;
          letter-spacing: 0.5px;
        }

        .blog-date {
          font-size: 12px;
          color: var(--text-muted);
        }

        .blog-content h3 {
          font-family: var(--font-display);
          font-size: 18px;
          font-weight: 600;
          margin-bottom: 8px;
          line-height: 1.4;
        }

        .blog-content p {
          font-size: 14px;
          color: var(--text-secondary);
          line-height: 1.6;
          margin-bottom: 16px;
          display: -webkit-box;
          -webkit-line-clamp: 2;
          -webkit-box-orient: vertical;
          overflow: hidden;
        }

        .blog-tags {
          display: flex;
          gap: 6px;
          flex-wrap: wrap;
          margin-bottom: 12px;
        }

        .blog-tag {
          font-size: 11px;
          color: var(--text-muted);
          background: var(--bg-secondary);
          padding: 3px 10px;
          border-radius: 100px;
        }

        .blog-footer {
          display: flex;
          justify-content: space-between;
          align-items: center;
        }

        .blog-readtime {
          font-size: 12px;
          color: var(--text-muted);
        }

        .read-more {
          display: flex;
          align-items: center;
          gap: 4px;
          font-size: 13px;
          font-weight: 600;
          color: #2563EB;
        }

        .no-results {
          grid-column: 1 / -1;
          text-align: center;
          padding: 60px 20px;
        }

        .no-results svg {
          color: var(--text-muted);
          margin-bottom: 16px;
        }

        .no-results h3 {
          font-size: 18px;
          margin-bottom: 8px;
          color: var(--text-primary);
        }

        .no-results p {
          font-size: 14px;
          color: var(--text-secondary);
        }

        @media (max-width: 768px) {
          .blog-grid {
            grid-template-columns: 1fr;
          }

          .search-bar {
            max-width: 100%;
          }
        }
      `}</style>
    </div>
  )
}

export default BlogPage
