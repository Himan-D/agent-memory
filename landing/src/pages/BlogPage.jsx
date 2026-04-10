import { motion } from 'framer-motion'
import { Link } from 'react-router-dom'
import { blogs } from '../data/blogs'

function BlogPage() {
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
        <motion.div 
          className="blog-grid"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.2 }}
        >
          {blogs.map((blog, index) => (
            <motion.article
              key={blog.slug}
              initial={{ opacity: 0, y: 30 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.5, delay: 0.1 * index }}
            >
              <Link to={`/blog/${blog.slug}`} className="blog-card">
                <div className="blog-image">
                  <img src={blog.image} alt={blog.title} />
                </div>
                <div className="blog-content">
                  <div className="blog-meta">
                    <span className="blog-category">{blog.category}</span>
                    <span className="blog-date">{blog.date}</span>
                  </div>
                  <h3>{blog.title}</h3>
                  <p>{blog.excerpt}</p>
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
          ))}
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

        .blog-grid {
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
          gap: 32px;
          padding: 48px 0;
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
        }

        .blog-image img {
          width: 100%;
          height: 100%;
          object-fit: cover;
          transition: transform 0.5s ease;
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

        @media (max-width: 768px) {
          .blog-grid {
            grid-template-columns: 1fr;
          }
        }
      `}</style>
    </div>
  )
}

export default BlogPage