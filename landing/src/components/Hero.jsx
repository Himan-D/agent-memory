import { useState, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import analytics from '../utils/analytics.js'
import { DASHBOARD_SIGNIN_URL } from '../constants'

const INSTALL_COMMANDS = {
  mac: {
    label: 'macOS / Linux',
    prompt: '$',
    command: 'curl -fsSL https://hystersis.com/install.sh | bash',
  },
  win: {
    label: 'Windows',
    prompt: '>',
    command: 'irm https://hystersis.com/install.ps1 | iex',
  },
}

function Hero() {
  const [copied, setCopied] = useState(false)
  const [activeOS, setActiveOS] = useState('mac')

  const activeCmd = INSTALL_COMMANDS[activeOS]

  const handleCopy = useCallback(() => {
    navigator.clipboard.writeText(activeCmd.command)
    analytics.ctaClicked('copy_install', 'hero')
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }, [activeCmd.command])

  const handleTabSwitch = useCallback((os) => {
    setActiveOS(os)
    setCopied(false)
  }, [])

  return (
    <section className="hero-section">
      <div className="container hero-grid">
        <div className="hero-copy">
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
            semantic search, and enterprise SSO. Build agents that learn and
            remember across every conversation.
          </motion.p>

          <motion.div
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.35 }}
            className="hero-buttons"
          >
            <a
              href="https://github.com/Himan-D/agent-memory"
              className="btn btn-primary"
              target="_blank"
              rel="noopener noreferrer"
              onClick={() => analytics.ctaClicked('github_star', 'hero')}
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
              </svg>
              View on GitHub
            </a>
            <a
              href={DASHBOARD_SIGNIN_URL}
              className="btn btn-primary"
              onClick={() => analytics.ctaClicked('dashboard_signin', 'hero')}
            >
              Open Dashboard
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M5 12h14M12 5l7 7-7 7" />
              </svg>
            </a>
            <Link to="/demo" className="btn btn-secondary" onClick={() => analytics.ctaClicked('see_demo', 'hero')}>
              See it in Action
            </Link>
          </motion.div>
        </div>

        <motion.div
          initial={{ opacity: 0, y: 30 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.25 }}
          className="hero-laptop"
        >
          <div className="laptop-shell">
            <div className="laptop-bezel">
              <div className="laptop-camera" aria-hidden="true" />
              <div className="laptop-screen">
                <motion.div
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.6, delay: 0.3 }}
                  className="hero-install"
                >
                  <div className="install-header">
                    <span className="install-label">Install in one command</span>
                    <div className="install-dots">
                      <span />
                      <span />
                      <span />
                    </div>
                  </div>

                  {/* OS Tabs */}
                  <div className="install-tabs">
                    {Object.entries(INSTALL_COMMANDS).map(([key, { label }]) => (
                      <button
                        key={key}
                        className={`install-tab${activeOS === key ? ' active' : ''}`}
                        onClick={() => handleTabSwitch(key)}
                        type="button"
                      >
                        {key === 'mac' ? (
                          <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" style={{ flexShrink: 0 }}>
                            <path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.8-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z" />
                          </svg>
                        ) : (
                          <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" style={{ flexShrink: 0 }}>
                            <path d="M0 3.449L9.75 2.1v9.451H0m10.949-9.602L24 0v11.4H10.949M0 12.6h9.75v9.451L0 20.699M10.949 12.6H24V24l-12.9-1.801" />
                          </svg>
                        )}
                        {label}
                      </button>
                    ))}
                  </div>

                  <div className="install-body" onClick={handleCopy}>
                    <span className="install-prompt">{activeCmd.prompt}</span>
                    <code>{activeCmd.command}</code>
                    <span className={`install-copy ${copied ? 'copied' : ''}`} title={copied ? 'Copied!' : 'Copy command'}>
                      {copied ? (
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                          <polyline points="20 6 9 17 4 12" />
                        </svg>
                      ) : (
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                          <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                          <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" />
                        </svg>
                      )}
                    </span>
                  </div>
                </motion.div>
              </div>
            </div>
            <div className="laptop-base">
              <div className="laptop-notch" />
            </div>
          </div>
        </motion.div>
      </div>

      <style>{`
        .hero-section {
          min-height: 90vh;
          display: flex;
          align-items: center;
          justify-content: center;
          padding: 120px 24px 80px;
          background: var(--bg-primary);
        }

        .hero-grid {
          display: grid;
          grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
          gap: 56px;
          align-items: center;
        }

        .hero-copy {
          max-width: 560px;
        }

        .hero-badge {
          display: inline-block;
          padding: 8px 16px;
          font-size: 12px;
          font-weight: 500;
          color: var(--text-secondary);
          border: 1px solid var(--border-light);
          border-radius: 100px;
          margin-bottom: 24px;
        }

        .hero-title {
          font-size: clamp(36px, 5vw, 56px);
          font-weight: 700;
          line-height: 1.15;
          margin-bottom: 20px;
          letter-spacing: -1px;
          text-align: left;
        }

        .hero-title-bold,
        .hero-title-highlight {
          font-weight: 800;
        }

        .hero-title-highlight {
          background: linear-gradient(135deg, var(--text-primary) 0%, var(--accent) 100%);
          -webkit-background-clip: text;
          -webkit-text-fill-color: transparent;
          background-clip: text;
        }

        .hero-subtitle {
          font-size: 18px;
          color: var(--text-secondary);
          margin-bottom: 32px;
          line-height: 1.7;
          text-align: left;
        }

        .hero-buttons {
          display: flex;
          gap: 12px;
          flex-wrap: wrap;
        }

        .hero-laptop {
          display: flex;
          justify-content: center;
          align-items: center;
          width: 100%;
        }

        .laptop-shell {
          width: 100%;
          max-width: 540px;
          min-width: 0;
        }

        .laptop-bezel {
          border: 2px solid var(--border-medium);
          border-radius: 18px 18px 0 0;
          background: linear-gradient(180deg, #1a1a1a 0%, #0f0f0f 100%);
          padding: 14px 14px 0;
          box-shadow: 0 24px 60px var(--card-shadow);
        }

        .laptop-camera {
          width: 8px;
          height: 8px;
          border-radius: 50%;
          background: #2a2a2a;
          margin: 0 auto 10px;
        }

        .laptop-screen {
          min-height: 280px;
          border-radius: 10px 10px 0 0;
          background: #0d1117;
          display: flex;
          align-items: center;
          justify-content: center;
          padding: 28px 24px;
        }

        .laptop-base {
          height: 16px;
          margin: 0 auto;
          width: calc(100% + 48px);
          margin-left: -24px;
          border-radius: 0 0 14px 14px;
          background: linear-gradient(180deg, #d4d4d4 0%, #a3a3a3 100%);
          position: relative;
        }

        [data-theme="dark"] .laptop-base {
          background: linear-gradient(180deg, #3a3a3a 0%, #1f1f1f 100%);
        }

        .laptop-notch {
          position: absolute;
          top: 0;
          left: 50%;
          transform: translateX(-50%);
          width: 96px;
          height: 6px;
          border-radius: 0 0 8px 8px;
          background: rgba(0, 0, 0, 0.18);
        }

        .hero-install {
          width: 100%;
          border-radius: 12px;
          overflow: hidden;
          border: 1px solid rgba(255, 255, 255, 0.08);
          background: rgba(0, 0, 0, 0.35);
          cursor: pointer;
          transition: border-color 0.2s;
          min-width: 0;
        }

        .hero-install:hover {
          border-color: var(--accent);
        }

        .install-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 8px 14px;
          background: rgba(255, 255, 255, 0.04);
          border-bottom: 1px solid rgba(255, 255, 255, 0.06);
        }

        .install-label {
          font-size: 11px;
          font-weight: 500;
          color: rgba(255, 255, 255, 0.5);
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
          background: rgba(255, 255, 255, 0.15);
        }

        /* OS Tabs */
        .install-tabs {
          display: flex;
          border-bottom: 1px solid rgba(255, 255, 255, 0.06);
        }

        .install-tab {
          flex: 1;
          display: flex;
          align-items: center;
          justify-content: center;
          gap: 6px;
          padding: 8px 12px;
          font-size: 11px;
          font-weight: 500;
          color: rgba(255, 255, 255, 0.35);
          background: transparent;
          border: none;
          cursor: pointer;
          transition: color 0.2s, background 0.2s, box-shadow 0.2s;
          position: relative;
          font-family: inherit;
          letter-spacing: 0.3px;
        }

        .install-tab:hover {
          color: rgba(255, 255, 255, 0.6);
          background: rgba(255, 255, 255, 0.03);
        }

        .install-tab.active {
          color: rgba(255, 255, 255, 0.85);
          background: rgba(255, 255, 255, 0.05);
        }

        .install-tab.active::after {
          content: '';
          position: absolute;
          bottom: 0;
          left: 12px;
          right: 12px;
          height: 2px;
          background: var(--accent);
          border-radius: 2px 2px 0 0;
        }

        .install-tab + .install-tab {
          border-left: 1px solid rgba(255, 255, 255, 0.06);
        }

        .install-body {
          display: flex;
          align-items: center;
          gap: 8px;
          padding: 12px 14px;
          font-family: 'SF Mono', 'Monaco', 'Menlo', 'Consolas', monospace;
          font-size: 12px;
        }

        .install-prompt {
          color: rgba(255, 255, 255, 0.3);
          flex-shrink: 0;
        }

        .install-body code {
          color: #e6edf3;
          font-family: inherit;
          font-size: inherit;
          text-align: left;
          white-space: nowrap;
          overflow-x: auto;
          min-width: 0;
          flex: 1;
          -ms-overflow-style: none;
          scrollbar-width: none;
        }

        .install-body code::-webkit-scrollbar {
          display: none;
        }

        .install-copy {
          margin-left: auto;
          flex-shrink: 0;
          color: rgba(255, 255, 255, 0.35);
          transition: color 0.2s;
        }

        .hero-install:hover .install-copy {
          color: var(--accent);
        }

        @media (max-width: 960px) {
          .hero-grid {
            grid-template-columns: minmax(0, 1fr);
            gap: 40px;
          }

          .hero-copy {
            max-width: 100%;
            text-align: center;
          }

          .hero-title,
          .hero-subtitle {
            text-align: center;
          }

          .hero-buttons {
            justify-content: center;
          }
        }

        @media (max-width: 640px) {
          .hero-section {
            padding: 64px 0 48px;
            min-height: auto;
            overflow-x: hidden;
            width: 100%;
          }

          .hero-laptop {
            padding: 0 16px;
          }

          .hero-title {
            font-size: clamp(32px, 10vw, 40px);
          }

          .hero-subtitle {
            font-size: 15px;
            margin-bottom: 24px;
          }

          .laptop-bezel {
            padding: 10px 10px 0;
            border-radius: 12px 12px 0 0;
            box-shadow: 0 12px 30px var(--card-shadow);
          }

          .laptop-screen {
            min-height: 200px;
            padding: 16px;
          }

          .laptop-base {
            height: 12px;
            width: calc(100% + 24px);
            margin-left: -12px;
            border-radius: 0 0 12px 12px;
          }

          .laptop-notch {
            width: 72px;
            height: 5px;
            border-radius: 0 0 6px 6px;
          }

          .install-body {
            padding: 12px;
            gap: 8px;
          }

          .install-body code {
            font-size: 11px;
          }

          .hero-buttons {
            flex-direction: column;
            align-items: stretch;
          }

          .hero-buttons .btn {
            width: 100%;
            max-width: 100%;
            justify-content: center;
          }
        }
      `}</style>
    </section>
  )
}

export default Hero
