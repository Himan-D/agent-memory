import { useState, useEffect } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { useTheme } from '../context/ThemeContext'
import { motion, AnimatePresence } from 'framer-motion'

const CALENDLY_URL = 'https://calendly.com/hystersis-support/30min'

function Navbar() {
  const [scrolled, setScrolled] = useState(false)
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const location = useLocation()
  const { theme, toggleTheme } = useTheme()

  useEffect(() => {
    const handleScroll = () => {
      setScrolled(window.scrollY > 50)
    }
    window.addEventListener('scroll', handleScroll)
    return () => window.removeEventListener('scroll', handleScroll)
  }, [])

  useEffect(() => {
    setMobileMenuOpen(false)
  }, [location])

  const navLinks = [
    { path: '/', label: 'Home' },
    { path: '/use-cases', label: 'Use Cases' },
    { path: 'https://docs.hystersis.ai', label: 'Docs' },
    { path: '/blog', label: 'Blog' },
  ]

  return (
    <>
      <motion.nav 
        className={`navbar ${scrolled ? 'scrolled' : ''}`}
        initial={{ y: -100 }}
        animate={{ y: 0 }}
        transition={{ duration: 0.5 }}
      >
        <div className="navbar-content">
          <Link to="/" className="navbar-logo">
            <svg width="24" height="24" viewBox="0 0 32 32" fill="none">
              <circle cx="16" cy="16" r="4" fill="var(--text-primary)"/>
              <circle cx="8" cy="10" r="2.5" fill="var(--text-primary)"/>
              <circle cx="24" cy="10" r="2.5" fill="var(--text-primary)"/>
              <circle cx="8" cy="22" r="2.5" fill="var(--text-primary)"/>
              <circle cx="24" cy="22" r="2.5" fill="var(--text-primary)"/>
              <line x1="16" y1="16" x2="8" y2="10" stroke="var(--text-primary)" strokeWidth="1.5"/>
              <line x1="16" y1="16" x2="24" y2="10" stroke="var(--text-primary)" strokeWidth="1.5"/>
              <line x1="16" y1="16" x2="8" y2="22" stroke="var(--text-primary)" strokeWidth="1.5"/>
              <line x1="16" y1="16" x2="24" y2="22" stroke="var(--text-primary)" strokeWidth="1.5"/>
            </svg>
            <span>Hystersis</span>
          </Link>

          <div className="navbar-center">
            {navLinks.map((link) => (
              link.path.startsWith('http') ? (
                <a 
                  key={link.path} 
                  href={link.path}
                  className="nav-link"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  {link.label}
                </a>
              ) : (
                <Link 
                  key={link.path} 
                  to={link.path} 
                  className={`nav-link ${location.pathname === link.path ? 'active' : ''}`}
                >
                  {link.label}
                </Link>
              )
            ))}
          </div>

          <div className="navbar-actions">
            <button className="icon-btn" onClick={toggleTheme} aria-label="Toggle theme">
              {theme === 'light' ? (
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
                  <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
                </svg>
              ) : (
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
                  <circle cx="12" cy="12" r="5"/>
                  <line x1="12" y1="1" x2="12" y2="3"/>
                  <line x1="12" y1="21" x2="12" y2="23"/>
                  <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/>
                  <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/>
                  <line x1="1" y1="12" x2="3" y2="12"/>
                  <line x1="21" y1="12" x2="23" y2="12"/>
                  <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/>
                  <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>
                </svg>
              )}
            </button>
            <a href="https://github.com/Himan-D/agent-memory" className="icon-btn" target="_blank" rel="noopener noreferrer" aria-label="GitHub">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
              </svg>
            </a>
            <a href="https://calendly.com/hystersis-support/30min" className="nav-cta" target="_blank" rel="noopener noreferrer">
              Book Demo
            </a>
          </div>

          <button 
            className="mobile-toggle"
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            aria-label="Menu"
          >
            <span className={`hamburger ${mobileMenuOpen ? 'open' : ''}`} />
          </button>
        </div>
      </motion.nav>

      <AnimatePresence>
        {mobileMenuOpen && (
          <motion.div 
            className="mobile-menu"
            initial={{ opacity: 0, y: -20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.2 }}
          >
            <div className="mobile-links">
              {navLinks.map((link) => (
                link.path.startsWith('http') ? (
                  <a 
                    key={link.path} 
                    href={link.path}
                    className="mobile-link"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    {link.label}
                  </a>
                ) : (
                  <Link 
                    key={link.path} 
                    to={link.path} 
                    className={`mobile-link ${location.pathname === link.path ? 'active' : ''}`}
                  >
                    {link.label}
                  </Link>
                )
              ))}
            </div>
            <div className="mobile-actions">
              <button className="mobile-theme-btn" onClick={toggleTheme}>
                {theme === 'light' ? 'Dark Mode' : 'Light Mode'}
              </button>
              <a href="https://github.com/Himan-D/agent-memory" className="mobile-link-external" target="_blank" rel="noopener noreferrer">
                GitHub
              </a>
              <a href="https://calendly.com/hystersis-support/30min" className="mobile-cta" target="_blank" rel="noopener noreferrer">
                Book Demo
              </a>
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      <style>{`
        .navbar {
          position: fixed;
          top: 0;
          left: 0;
          right: 0;
          z-index: 1000;
          background: var(--nav-bg);
          border-bottom: 1px solid var(--nav-border);
          transition: all 0.3s ease;
        }

        .navbar-content {
          max-width: 1200px;
          margin: 0 auto;
          padding: 16px 24px;
          display: flex;
          align-items: center;
          justify-content: space-between;
          gap: 32px;
        }

        .navbar.scrolled .navbar-content {
          padding: 12px 24px;
        }

        .navbar-logo {
          display: flex;
          align-items: center;
          gap: 10px;
          font-size: 16px;
          font-weight: 600;
          color: var(--text-primary);
          text-decoration: none;
        }

        .navbar-center {
          display: flex;
          align-items: center;
          gap: 8px;
        }

        .nav-link {
          padding: 8px 16px;
          font-size: 14px;
          font-weight: 500;
          color: var(--text-secondary);
          text-decoration: none;
          border-radius: 6px;
          transition: all 0.2s ease;
        }

        .nav-link:hover {
          color: var(--text-primary);
          background: var(--bg-secondary);
        }

        .nav-link.active {
          color: var(--text-primary);
        }

        .navbar-actions {
          display: flex;
          align-items: center;
          gap: 8px;
        }

        .icon-btn {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 36px;
          height: 36px;
          background: transparent;
          border: 1px solid var(--border-light);
          border-radius: 8px;
          cursor: pointer;
          color: var(--text-secondary);
          text-decoration: none;
          transition: all 0.2s ease;
        }

        .icon-btn:hover {
          color: var(--text-primary);
          background: var(--bg-secondary);
          border-color: var(--border-medium);
        }

        .nav-cta {
          display: inline-flex;
          align-items: center;
          padding: 8px 18px;
          font-size: 14px;
          font-weight: 500;
          color: var(--btn-primary-text);
          background: var(--btn-primary-bg);
          border-radius: 8px;
          text-decoration: none;
          transition: all 0.2s ease;
          margin-left: 8px;
        }

        .nav-cta:hover {
          opacity: 0.85;
        }

        .mobile-toggle {
          display: none;
          background: none;
          border: none;
          padding: 8px;
          cursor: pointer;
        }

        .hamburger {
          display: block;
          width: 20px;
          height: 2px;
          background: var(--text-primary);
          position: relative;
          transition: all 0.3s ease;
        }

        .hamburger::before,
        .hamburger::after {
          content: '';
          position: absolute;
          width: 20px;
          height: 2px;
          background: var(--text-primary);
          transition: all 0.3s ease;
        }

        .hamburger::before { top: -6px; }
        .hamburger::after { top: 6px; }

        .hamburger.open {
          background: transparent;
        }

        .hamburger.open::before {
          top: 0;
          transform: rotate(45deg);
        }

        .hamburger.open::after {
          top: 0;
          transform: rotate(-45deg);
        }

        .mobile-menu {
          position: fixed;
          top: 61px;
          left: 0;
          right: 0;
          background: var(--nav-bg);
          border-bottom: 1px solid var(--nav-border);
          padding: 16px 24px 24px;
          z-index: 999;
        }

        .mobile-links {
          display: flex;
          flex-direction: column;
          gap: 4px;
          margin-bottom: 16px;
        }

        .mobile-link {
          padding: 12px 0;
          font-size: 15px;
          font-weight: 500;
          color: var(--text-secondary);
          text-decoration: none;
          border-bottom: 1px solid var(--border-light);
        }

        .mobile-link:last-child {
          border-bottom: none;
        }

        .mobile-link.active {
          color: var(--text-primary);
        }

        .mobile-actions {
          display: flex;
          flex-direction: column;
          gap: 8px;
          padding-top: 16px;
          border-top: 1px solid var(--border-light);
        }

        .mobile-theme-btn,
        .mobile-link-external {
          padding: 12px 0;
          font-size: 15px;
          font-weight: 500;
          color: var(--text-secondary);
          text-decoration: none;
          background: none;
          border: none;
          text-align: left;
          cursor: pointer;
        }

        .mobile-cta {
          display: block;
          padding: 14px 24px;
          font-size: 14px;
          font-weight: 600;
          color: var(--btn-primary-text);
          background: var(--btn-primary-bg);
          border-radius: 8px;
          text-decoration: none;
          text-align: center;
          margin-top: 8px;
        }

        .main-content {
          padding-top: 61px;
        }

        @media (max-width: 768px) {
          .navbar-center,
          .navbar-actions {
            display: none;
          }

          .mobile-toggle {
            display: block;
          }

          .navbar-content {
            padding: 14px 20px;
          }
        }

        @media (min-width: 769px) {
          .mobile-menu {
            display: none;
          }
        }
      `}</style>
    </>
  )
}

export default Navbar
