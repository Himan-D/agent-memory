import { motion } from 'framer-motion'

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
            </div>
            <p className="footer-tagline">
              Persistent memory infrastructure for AI agents. Remember more, forget less.
            </p>
          </div>

          <div className="footer-links">
            <div className="footer-col">
              <h4>Product</h4>
              <a href="#features">Features</a>
              <a href="#usecases">Use Cases</a>
              <a href="#pricing">Pricing</a>
              <a href="https://github.com/Himan-D/agent-memory" target="_blank" rel="noopener noreferrer">GitHub</a>
            </div>
            <div className="footer-col">
              <h4>Resources</h4>
              <a href="https://docs.hystersis.ai">Documentation</a>
              <a href="https://docs.hystersis.ai/quickstart">Quick Start</a>
              <a href="/use-cases">Examples</a>
              <a href="/blog">Blog</a>
            </div>
            <div className="footer-col">
              <h4>Connect</h4>
              <a href="https://x.com/HHystersis" target="_blank" rel="noopener noreferrer">Twitter</a>
              <a href="https://www.linkedin.com/company/hystersis-ai/" target="_blank" rel="noopener noreferrer">LinkedIn</a>
              <a href="https://github.com/Himan-D/agent-memory" target="_blank" rel="noopener noreferrer">GitHub</a>
            </div>
            <div className="footer-col">
              <h4>Status</h4>
              <a href="/status">System Status</a>
              <a href="https://status.hystersis.ai" target="_blank" rel="noopener noreferrer">Incident History</a>
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

        @media (max-width: 640px) {
          .footer-content {
            flex-direction: column;
          }

          .footer-links {
            gap: 32px;
          }
        }
      `}</style>
    </footer>
  )
}

export default Footer
