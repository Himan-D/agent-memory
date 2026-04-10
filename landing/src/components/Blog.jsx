import { motion } from 'framer-motion'
import { Link } from 'react-router-dom'
import { blogs } from '../data/blogs'

function Blog() {
  return (
    <section className="blog-section section">
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
          {blogs.slice(0, 3).map((blog, index) => (
            <motion.article
              key={blog.slug}
              initial={{ opacity: 0, y: 30 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.5, delay: index * 0.1 }}
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
          color: #2563EB;
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