import { Link } from 'react-router-dom'
import { useTheme } from '../context/ThemeContext'
import BrandLogo from './BrandLogo'
import { MAIN_SITE_URL } from '../constants/blog'

function BlogNavbar() {
  const { theme, toggleTheme } = useTheme()

  return (
    <nav className="blog-navbar">
      <div className="container blog-navbar-inner">
        <a href={MAIN_SITE_URL} className="blog-navbar-brand">
          <BrandLogo size={28} />
          <span>Hystersis</span>
        </a>
        <div className="blog-navbar-links">
          <span className="blog-navbar-label">Blog</span>
          <a href={`${MAIN_SITE_URL}/docs`}>Docs</a>
          <a href={`${MAIN_SITE_URL}/demo`}>Demo</a>
          <a href="https://app.hystersis.com">Dashboard</a>
          <button type="button" onClick={toggleTheme} aria-label="Toggle theme" className="theme-toggle">
            {theme === 'dark' ? '☀️' : '🌙'}
          </button>
        </div>
      </div>
      <style>{`
        .blog-navbar {
          position: sticky;
          top: 0;
          z-index: 100;
          background: var(--bg-primary);
          border-bottom: 1px solid var(--border-light);
          backdrop-filter: blur(12px);
        }
        .blog-navbar-inner {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 16px 0;
        }
        .blog-navbar-brand {
          display: flex;
          align-items: center;
          gap: 10px;
          font-weight: 700;
          font-size: 18px;
          color: var(--text-primary);
          text-decoration: none;
        }
        .blog-navbar-links {
          display: flex;
          align-items: center;
          gap: 20px;
        }
        .blog-navbar-links a {
          color: var(--text-secondary);
          text-decoration: none;
          font-size: 14px;
          font-weight: 500;
        }
        .blog-navbar-links a:hover {
          color: var(--accent-primary);
        }
        .blog-navbar-label {
          font-size: 13px;
          font-weight: 600;
          color: var(--accent-primary);
          text-transform: uppercase;
          letter-spacing: 0.5px;
        }
        .theme-toggle {
          background: var(--bg-secondary);
          border: 1px solid var(--border-light);
          border-radius: 8px;
          padding: 6px 10px;
          cursor: pointer;
        }
      `}</style>
    </nav>
  )
}

export default BlogNavbar
