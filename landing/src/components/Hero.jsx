import { useState, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import analytics from '../utils/analytics.js'
import { DASHBOARD_SIGNIN_URL } from '../constants'

function Hero() {
  const [copied, setCopied] = useState(false)

  const handleCopy = useCallback(() => {
    navigator.clipboard.writeText('curl -fsSL https://hystersis.com/install.sh | bash')
    analytics.ctaClicked('copy_install', 'hero')
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }, [])

  return (
    <section className="hero-section">
      <div className="container">
        <div className="hero-content">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6 }}
            className="hero-badge"
          >
            Open Source
          </motion.div>

          <motion.h1
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.1 }}
            className="hero-title"
          >
            <span className="hero-title-bold">AI Agents That</span>
            <br />
            <span className="hero-title-highlight">Actually Remember.</span>
          </motion.h1>

          <motion.p
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.2 }}
            className="hero-subtitle"
          >
            Give your AI agents persistent memory with graph-powered storage,
            semantic search, and enterprise SSO.
            <br />
            Build agents that learn and remember across every conversation.
          </motion.p>

          <motion.div
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.3 }}
            className="hero-install"
          >
            <div className="install-header">
              <span className="install-label">Install in one command</span>
              <div className="install-dots">
                <span></span><span></span><span></span>
              </div>
            </div>
            <div className="install-body" onClick={handleCopy}>
              <span className="install-prompt">$</span>
              <code>curl -fsSL https://hystersis.com/install.sh | bash</code>
              <span className={`install-copy ${copied ? 'copied' : ''}`} title={copied ? 'Copied!' : 'Copy command'}>
                {copied ? (
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <polyline points="20 6 9 17 4 12"/>
                  </svg>
                ) : (
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                    <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/>
                  </svg>
                )}
              </span>
            </div>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.35 }}
            className="hero-buttons"
          >
            <a href="https://github.com/Himan-D/agent-memory" className="btn btn-primary" target="_blank" rel="noopener noreferrer" onClick={() => analytics.ctaClicked('github_star', 'hero')}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
              </svg>
              View on GitHub
            </a>
            <a href={DASHBOARD_SIGNIN_URL} className="btn btn-primary" onClick={() => analytics.ctaClicked('dashboard_signin', 'hero')}>
              Open Dashboard
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M5 12h14M12 5l7 7-7 7"/>
              </svg>
            </a>
            <Link to="/demo" className="btn btn-secondary" onClick={() => analytics.ctaClicked('see_demo', 'hero')}>
              See it in Action
            </Link>
          </motion.div>
        </div>
      </div>

      <style>{`
        .hero-section {
          min-height: 90vh;
          display: flex;
          align-items: center;
          justify-content: center;
          padding: 120px 24px;
          background: var(--bg-primary);
        }

        .hero-content {
          max-width: 780px;
          text-align: center;
        }

        .hero-badge {
          display: inline-block;
          padding: 8px 16px;
          font-size: 12px;
          font-weight: 500;
          color: var(--text-secondary);
          border: 1px solid var(--border-light);
          border-radius: 100px;
          margin-bottom: 32px;
        }

        .hero-title {
          font-size: clamp(36px, 6vw, 56px);
          font-weight: 700;
          line-height: 1.15;
          margin-bottom: 24px;
          letter-spacing: -1px;
        }

        .hero-title-bold {
          font-weight: 800;
        }

        .hero-title-highlight {
          font-weight: 800;
          background: linear-gradient(135deg, var(--text-primary) 0%, var(--accent) 100%);
          -webkit-background-clip: text;
          -webkit-text-fill-color: transparent;
          background-clip: text;
        }

        .hero-subtitle {
          font-size: 18px;
          color: var(--text-secondary);
          max-width: 480px;
          margin: 0 auto 40px;
          line-height: 1.7;
        }

        .hero-install {
          margin: 0 auto 32px;
          max-width: 600px;
          border-radius: 12px;
          overflow: hidden;
          border: 1px solid var(--border-light);
          background: #0d1117;
          cursor: pointer;
          transition: border-color 0.2s;
        }

        .hero-install:hover {
          border-color: var(--accent);
        }

        .install-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 8px 14px;
          background: rgba(255,255,255,0.04);
          border-bottom: 1px solid rgba(255,255,255,0.06);
        }

        .install-label {
          font-size: 11px;
          font-weight: 500;
          color: rgba(255,255,255,0.5);
          text-transform: uppercase;
          letter-spacing: 0.5px;
        }

        .install-dots {
          display: flex;
          gap: 6px;
        }

        .install-dots span {
          width: 10px;
          height: 10px;
          border-radius: 50%;
          background: rgba(255,255,255,0.15);
        }

        .install-body {
          display: flex;
          align-items: center;
          gap: 10px;
          padding: 14px 16px;
          font-family: 'SF Mono', 'Monaco', 'Menlo', 'Consolas', monospace;
          font-size: 14px;
          overflow-x: auto;
          -webkit-overflow-scrolling: touch;
        }

        .install-prompt {
          color: rgba(255,255,255,0.3);
          flex-shrink: 0;
        }

        .install-body code {
          color: #e6edf3;
          font-family: inherit;
          font-size: inherit;
          text-align: left;
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
        }

        .install-copy {
          margin-left: auto;
          flex-shrink: 0;
          font-size: 11px;
          color: rgba(255,255,255,0.35);
          text-transform: uppercase;
          letter-spacing: 0.5px;
          transition: color 0.2s;
        }

        .hero-install:hover .install-copy {
          color: var(--accent);
        }

        .hero-buttons {
          display: flex;
          gap: 16px;
          justify-content: center;
          flex-wrap: wrap;
        }

        @media (max-width: 768px) {
          .hero-section {
            padding: 96px 20px 64px;
            min-height: auto;
          }

          .hero-subtitle {
            font-size: 16px;
            padding: 0 8px;
          }
        }

        @media (max-width: 640px) {
          .hero-section {
            padding: 80px 16px;
          }

          .hero-title {
            font-size: clamp(28px, 8vw, 36px);
            letter-spacing: -0.5px;
          }

          .hero-install {
            max-width: 100%;
          }

          .install-body code {
            font-size: 12px;
          }

          .hero-buttons {
            flex-direction: column;
            align-items: center;
          }

          .hero-buttons .btn {
            width: 100%;
            max-width: 280px;
            justify-content: center;
          }
        }
      `}</style>
    </section>
  )
}

export default Hero
