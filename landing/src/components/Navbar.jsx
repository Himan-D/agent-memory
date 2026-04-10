import { useState, useEffect } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { motion, AnimatePresence } from 'framer-motion'

function Navbar() {
  const [scrolled, setScrolled] = useState(false)
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const location = useLocation()

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
    { path: '/docs', label: 'Docs' },
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
        <div className="container navbar-content">
          <Link to="/" className="navbar-logo">
            <svg width="32" height="32" viewBox="0 0 32 32">
              <circle cx="16" cy="16" r="4" fill="#2563EB"/>
              <circle cx="8" cy="10" r="2.5" fill="#3B82F6"/>
              <circle cx="24" cy="10" r="2.5" fill="#3B82F6"/>
              <circle cx="8" cy="22" r="2.5" fill="#3B82F6"/>
              <circle cx="24" cy="22" r="2.5" fill="#3B82F6"/>
              <line x1="16" y1="16" x2="8" y2="10" stroke="#2563EB" strokeWidth="1"/>
              <line x1="16" y1="16" x2="24" y2="10" stroke="#2563EB" strokeWidth="1"/>
              <line x1="16" y1="16" x2="8" y2="22" stroke="#2563EB" strokeWidth="1"/>
              <line x1="16" y1="16" x2="24" y2="22" stroke="#2563EB" strokeWidth="1"/>
            </svg>
            <span>Agent Memory</span>
          </Link>

          <div className="navbar-links">
            {navLinks.map((link) => (
              <Link 
                key={link.path} 
                to={link.path} 
                className={`navbar-link ${location.pathname === link.path ? 'active' : ''}`}
              >
                {link.label}
              </Link>
            ))}
          </div>

          <div className="navbar-actions">
            <a href="https://github.com/Himan-D/agent-memory" className="btn btn-ghost" target="_blank" rel="noopener noreferrer">
              GitHub
            </a>
            <button onClick={() => window.Calendly ? window.Calendly.initPopupWidget({ url: 'https://calendly.com/your-username/demo' }) : window.open('https://calendly.com/your-username/demo', '_blank')} className="btn btn-primary btn-sm">
              Book Demo
            </button>
          </div>

          <button 
            className="navbar-mobile-toggle"
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
          >
            <span className={`hamburger ${mobileMenuOpen ? 'open' : ''}`} />
          </button>
        </div>
      </motion.nav>

      <AnimatePresence>
        {mobileMenuOpen && (
          <motion.div 
            className="navbar-mobile-menu"
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
          >
            {navLinks.map((link) => (
              <Link 
                key={link.path} 
                to={link.path} 
                className={`navbar-mobile-link ${location.pathname === link.path ? 'active' : ''}`}
              >
                {link.label}
              </Link>
            ))}
            <div className="navbar-mobile-actions">
              <a href="https://github.com/Himan-D/agent-memory" className="btn btn-ghost btn-full" target="_blank" rel="noopener noreferrer">
                GitHub
              </a>
              <button onClick={() => window.Calendly ? window.Calendly.initPopupWidget({ url: 'https://calendly.com/your-username/demo' }) : window.open('https://calendly.com/your-username/demo', '_blank')} className="btn btn-primary btn-full">
                Book Demo
              </button>
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
          padding: 16px 0;
          transition: all 0.3s ease;
        }

        .navbar.scrolled {
          background: rgba(250, 250, 250, 0.95);
          backdrop-filter: blur(10px);
          box-shadow: 0 1px 0 var(--border-light);
        }

        .navbar-content {
          display: flex;
          align-items: center;
          justify-content: space-between;
        }

        .navbar-logo {
          display: flex;
          align-items: center;
          gap: 10px;
          font-family: var(--font-display);
          font-size: 18px;
          font-weight: 600;
          color: var(--text-primary);
        }

        .navbar-links {
          display: flex;
          gap: 32px;
        }

        .navbar-link {
          font-size: 14px;
          font-weight: 500;
          color: var(--text-secondary);
          transition: color 0.2s ease;
        }

        .navbar-link:hover,
        .navbar-link.active {
          color: var(--color-primary);
        }

        .navbar-actions {
          display: flex;
          gap: 12px;
          align-items: center;
        }

        .btn-ghost {
          background: transparent;
          color: var(--text-primary);
          padding: 10px 16px;
        }

        .btn-ghost:hover {
          color: var(--color-accent);
        }

        .btn-sm {
          padding: 10px 20px;
          font-size: 14px;
        }

        .btn-full {
          width: 100%;
          justify-content: center;
        }

        .navbar-mobile-toggle {
          display: none;
          background: none;
          border: none;
          padding: 8px;
          cursor: pointer;
        }

        .hamburger {
          display: block;
          width: 24px;
          height: 2px;
          background: var(--text-primary);
          position: relative;
          transition: all 0.3s ease;
        }

        .hamburger::before,
        .hamburger::after {
          content: '';
          position: absolute;
          width: 24px;
          height: 2px;
          background: var(--text-primary);
          transition: all 0.3s ease;
        }

        .hamburger::before { top: -7px; }
        .hamburger::after { top: 7px; }

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

        .navbar-mobile-menu {
          display: none;
          position: fixed;
          top: 70px;
          left: 0;
          right: 0;
          background: var(--bg-primary);
          border-bottom: 1px solid var(--border-light);
          padding: 24px;
          overflow: hidden;
        }

        .navbar-mobile-link {
          display: block;
          padding: 12px 0;
          font-size: 16px;
          font-weight: 500;
          color: var(--text-secondary);
          border-bottom: 1px solid var(--border-light);
        }

        .navbar-mobile-link.active {
          color: var(--color-accent);
        }

        .navbar-mobile-actions {
          display: flex;
          flex-direction: column;
          gap: 12px;
          margin-top: 16px;
        }

        .main-content {
          padding-top: 70px;
        }

        @media (max-width: 768px) {
          .navbar-links,
          .navbar-actions {
            display: none;
          }

          .navbar-mobile-toggle {
            display: block;
          }

          .navbar-mobile-menu {
            display: block;
          }
        }
      `}</style>
    </>
  )
}

export default Navbar