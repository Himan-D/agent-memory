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
              <svg width="28" height="28" viewBox="0 0 32 32">
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
            </div>
            <p className="footer-tagline">
              Give your AI agents permanent, semantic memory with graph relationships.
            </p>
          </div>

          <div className="footer-links">
            <div className="footer-col">
              <h4>Product</h4>
              <a href="#features">Features</a>
              <a href="#usecases">Use Cases</a>
              <a href="https://github.com/Himan-D/agent-memory">GitHub</a>
            </div>
            <div className="footer-col">
              <h4>Resources</h4>
              <a href="/docs/openapi.yaml">API Reference</a>
              <a href="./QUICKSTART.md">Quick Start</a>
              <a href="./docs/use-cases.md">Examples</a>
            </div>
            <div className="footer-col">
              <h4>Company</h4>
              <a href="https://github.com/Himan-D/agent-memory">GitHub</a>
              <a href="#">Twitter</a>
              <a href="#">Discord</a>
            </div>
          </div>
        </motion.div>

        <div className="footer-bottom">
          <p>&copy; 2025 Agent Memory. MIT License.</p>
          <div className="footer-social">
            <a href="https://github.com/Himan-D/agent-memory" aria-label="GitHub">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
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
          font-family: var(--font-display);
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
          font-family: var(--font-display);
          font-size: 14px;
          font-weight: 600;
          margin-bottom: 16px;
          color: var(--text-primary);
        }

        .footer-col a {
          display: block;
          font-size: 14px;
          color: var(--text-secondary);
          margin-bottom: 10px;
        }

        .footer-col a:hover {
          color: var(--color-primary);
        }

        .footer-bottom {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding-top: 32px;
          border-top: 1px solid rgba(255, 255, 255, 0.06);
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
          color: var(--text-muted);
          transition: color 0.3s ease;
        }

        .footer-social a:hover {
          color: var(--color-primary);
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