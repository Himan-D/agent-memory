import { useParams, Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { getBlogBySlug, blogs } from '../data/blogs'

function BlogPost() {
  const { slug } = useParams()
  const blog = getBlogBySlug(slug)

  if (!blog) {
    return (
      <div className="not-found">
        <h1>Blog post not found</h1>
        <Link to="/blog" className="btn btn-primary">Back to Blog</Link>
      </div>
    )
  }

  const renderContent = (content) => {
    const lines = content.split('\n')
    const elements = []
    let inCodeBlock = false
    let codeContent = []
    let codeKey = 0

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i]

      if (line.startsWith('```')) {
        if (!inCodeBlock) {
          inCodeBlock = true
          codeContent = []
        } else {
          elements.push(
            <pre key={`code-${codeKey}`}>
              <code>{codeContent.join('\n')}</code>
            </pre>
          )
          codeContent = []
          inCodeBlock = false
          codeKey++
        }
        continue
      }

      if (inCodeBlock) {
        codeContent.push(line)
        continue
      }

      if (line.startsWith('### ')) {
        elements.push(<h3 key={i}>{line.replace('### ', '')}</h3>)
      } else if (line.startsWith('## ')) {
        elements.push(<h2 key={i}>{line.replace('## ', '')}</h2>)
      } else if (line.startsWith('# ')) {
        elements.push(<h1 key={i}>{line.replace('# ', '')}</h1>)
      } else if (line.startsWith('| ')) {
        elements.push(<p key={i} className="table-row">{line}</p>)
      } else if (line.trim() === '') {
        elements.push(<br key={i} />)
      } else {
        elements.push(<p key={i}>{line}</p>)
      }
    }

    return elements
  }

  return (
    <article className="blog-post">
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.5 }}
        className="blog-post-hero"
      >
        <img src={blog.image} alt={blog.title} />
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
            <span className="blog-category">{blog.category}</span>
            <span className="blog-date">{blog.date}</span>
            <span className="blog-readtime">{blog.readTime}</span>
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
          {renderContent(blog.content)}
        </motion.div>

        <div className="blog-post-footer">
          <h3>More Articles</h3>
          <div className="related-blogs">
            {blogs.filter(b => b.slug !== slug).slice(0, 3).map((relatedBlog) => (
              <Link key={relatedBlog.slug} to={`/blog/${relatedBlog.slug}`} className="related-blog-card">
                <img src={relatedBlog.image} alt={relatedBlog.title} />
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
        }

        .blog-post-meta span {
          color: rgba(255,255,255,0.8);
        }

        .blog-category {
          color: #3B82F6 !important;
          font-weight: 600;
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
        }

        .blog-post-content h2 {
          font-family: var(--font-display);
          font-size: 24px;
          font-weight: 700;
          margin: 32px 0 16px;
        }

        .blog-post-content h3 {
          font-family: var(--font-display);
          font-size: 18px;
          font-weight: 600;
          margin: 24px 0 12px;
        }

        .blog-post-content p {
          color: var(--text-secondary);
          line-height: 1.8;
          margin-bottom: 16px;
        }

        .blog-post-content pre {
          background: #1a1a1a;
          padding: 16px;
          border-radius: 8px;
          overflow-x: auto;
          margin: 16px 0;
        }

        .blog-post-content code {
          font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
          font-size: 14px;
          color: #e8e8e8;
        }

        .blog-post-footer {
          max-width: 720px;
          margin: 0 auto;
          padding-top: 48px;
          border-top: 1px solid var(--border-light);
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
        }

        .related-blog-card img {
          width: 80px;
          height: 60px;
          object-fit: cover;
          border-radius: 8px;
        }

        .related-blog-card h4 {
          font-size: 14px;
          font-weight: 600;
          margin-bottom: 4px;
        }

        .related-blog-card span {
          font-size: 12px;
          color: var(--text-muted);
        }

        .not-found {
          text-align: center;
          padding: 100px 24px;
        }

        .not-found h1 {
          font-family: var(--font-display);
          font-size: 24px;
          margin-bottom: 24px;
        }

        @media (max-width: 768px) {
          .blog-post-hero {
            height: 300px;
          }

          .blog-post-header {
            padding: 24px;
          }
        }
      `}</style>
    </article>
  )
}

export default BlogPost