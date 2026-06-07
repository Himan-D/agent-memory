import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import BrandLogo from './BrandLogo'

function Footer() {
  return (
    <footer className="footer">
      <div className="container">
        <motion.div
          initial={{ opacity: 0 }}
          whileInView={{ opacity: 1 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="footer-content"
        >
          <div className="footer-brand">
            <div className="footer-logo">
              <BrandLogo size={28} />
            </div>
            <p className="footer-tagline">
              Persistent memory infrastructure for AI agents. Remember more, forget less.
            </p>
          </div>

          <div className="footer-links">
            <div className="footer-col">
              <h4>Product</h4>
              <Link to="/#features">Features</Link>
              <Link to="/#usecases">Use Cases</Link>
              <Link to="/#pricing">Pricing</Link>
              <a href="https://github.com/Himan-D/agent-memory" target="_blank" rel="noopener noreferrer">GitHub</a>
            </div>
            <div className="footer-col">
              <h4>Resources</h4>
              <a href="/docs">Documentation</a>
              <a href="/docs/getting-started/quickstart">Quick Start</a>
              <Link to="/use-cases">Examples</Link>
              <Link to="/blog">Blog</Link>
            </div>
            <div className="footer-col">
              <h4>Connect</h4>
              <a href="https://discord.gg/Q7bfvqKG" target="_blank" rel="noopener noreferrer">Discord</a>
              <a href="https://x.com/HHystersis" target="_blank" rel="noopener noreferrer">Twitter</a>
              <a href="https://www.linkedin.com/company/hystersis-ai/" target="_blank" rel="noopener noreferrer">LinkedIn</a>
              <a href="https://github.com/Himan-D/agent-memory" target="_blank" rel="noopener noreferrer">GitHub</a>
            </div>
            <div className="footer-col">
              <h4>Status</h4>
              <Link to="/status">System Status</Link>
              <a href="https://hystersis.com/status" target="_blank" rel="noopener noreferrer">Incident History</a>
            </div>
          </div>
        </motion.div>

        <div className="footer-bottom">
          <p>&copy; {new Date().getFullYear()} Hystersis. MIT License.</p>
          <div className="footer-social">
            <a href="https://github.com/Himan-D/agent-memory" aria-label="GitHub" target="_blank" rel="noopener noreferrer">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
              </svg>
            </a>
            <a href="https://x.com/HHystersis" aria-label="Twitter" target="_blank" rel="noopener noreferrer">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z"/>
              </svg>
            </a>
            <a href="https://www.linkedin.com/company/hystersis-ai/" aria-label="LinkedIn" target="_blank" rel="noopener noreferrer">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                <path d="M20.447 20.452h-3.554v-5.569c0-1.328-.027-3.037-1.852-3.037-1.853 0-2.136 1.445-2.136 2.939v5.667H9.351V9h3.414v1.561h.046c.477-.9 1.637-1.85 3.37-1.85 3.601 0 4.267 2.37 4.267 5.455v6.286zM5.337 7.433c-1.144 0-2.063-.926-2.063-2.065 0-1.138.92-2.063 2.063-2.063 1.14 0 2.064.925 2.064 2.063 0 1.139-.925 2.065-2.064 2.065zm1.782 13.019H3.555V9h3.564v11.452zM22.225 0H1.771C.792 0 0 .774 0 1.729v20.542C0 23.227.792 24 1.771 24h20.451C23.2 24 24 23.227 24 22.271V1.729C24 .774 23.2 0 22.222 0h.003z"/>
              </svg>
            </a>
            <a href="https://discord.gg/Q7bfvqKG" aria-label="Discord" target="_blank" rel="noopener noreferrer">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                <path d="M20.317 4.37a19.791 19.791 0 00-4.885-1.515.074.074 0 00-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 00-5.487 0 12.64 12.64 0 00-.617-1.25.077.077 0 00-.079-.037A19.736 19.736 0 003.677 4.37a.07.07 0 00-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 00.031.057 19.9 19.9 0 005.993 3.03.078.078 0 00.084-.028 14.09 14.09 0 001.226-1.994.076.076 0 00-.041-.106 13.107 13.107 0 01-1.872-.892.077.077 0 01-.008-.128 10.2 10.2 0 00.372-.292.074.074 0 01.077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 01.078.01c.12.098.246.198.373.292a.077.077 0 01-.006.127 12.299 12.299 0 01-1.873.892.077.077 0 00-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 00.084.028 19.839 19.839 0 006.002-3.03.077.077 0 00.032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 00-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.157 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.157 2.418z"/>
              </svg>
            </a>
          </div>
        </div>
      </div>

      <style>{`
        .footer {
          background: var(--bg-primary);
          padding: 64px 0 32px;
          border-top: 1px solid var(--border-light);
        }

        .footer-content {
          display: flex;
          justify-content: space-between;
          gap: 48px;
          flex-wrap: wrap;
          margin-bottom: 48px;
        }

        .footer-brand {
          max-width: 280px;
        }

        .footer-logo {
          display: flex;
          align-items: center;
          gap: 10px;
          font-size: 18px;
          font-weight: 600;
          margin-bottom: 16px;
        }

        .footer-tagline {
          font-size: 14px;
          color: var(--text-secondary);
          line-height: 1.6;
        }

        .footer-links {
          display: flex;
          gap: 48px;
          flex-wrap: wrap;
        }

        .footer-col h4 {
          font-size: 13px;
          font-weight: 600;
          text-transform: uppercase;
          letter-spacing: 1px;
          margin-bottom: 16px;
        }

        .footer-col a {
          display: block;
          font-size: 14px;
          color: var(--text-secondary);
          margin-bottom: 10px;
          text-decoration: none;
        }

        .footer-col a:hover {
          color: var(--text-primary);
        }

        .footer-bottom {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding-top: 32px;
          border-top: 1px solid var(--border-light);
          flex-wrap: wrap;
          gap: 16px;
        }

        .footer-bottom p {
          font-size: 13px;
          color: var(--text-muted);
        }

        .footer-social {
          display: flex;
          gap: 16px;
        }

        .footer-social a {
          color: var(--text-secondary);
          transition: color 0.3s ease;
        }

        .footer-social a:hover {
          color: var(--text-primary);
        }

        @media (max-width: 900px) {
          .footer-links {
            display: grid;
            grid-template-columns: repeat(2, minmax(140px, 1fr));
            gap: 32px 24px;
            width: 100%;
          }
        }

        @media (max-width: 640px) {
          .footer {
            padding: 48px 0 24px;
          }

          .footer-content {
            flex-direction: column;
          }

          .footer-links {
            grid-template-columns: 1fr 1fr;
            gap: 24px 16px;
          }

          .footer-bottom {
            flex-direction: column;
            text-align: center;
          }
        }
      `}</style>
    </footer>
  )
}

export default Footer
