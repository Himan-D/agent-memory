import { motion } from 'framer-motion'

function CTA() {
  return (
    <section className="cta-section section">
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="cta-content"
        >
          <h2 className="cta-title">Start building memory-powered agents today.</h2>
          <p className="cta-description">
            Open source. Self-hostable. Enterprise-ready. Get started in under 5 minutes.
          </p>
          <div className="cta-buttons">
            <a href="https://github.com/Himan-D/agent-memory" className="btn btn-primary" target="_blank" rel="noopener noreferrer">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
              </svg>
              Start Building Free
            </a>
            <a href="#" className="btn btn-secondary" onClick={(e) => { e.preventDefault(); if (window.Calendly) { window.Calendly.initPopupWidget({ url: 'https://calendly.com/himanshu-dixit-trinetralabs/30min' }); } else { window.open('https://calendly.com/himanshu-dixit-trinetralabs/30min', '_blank'); } return false; }}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <rect x="3" y="4" width="18" height="18" rx="2"/>
                <path d="M16 2v4M8 2v4M3 10h18"/>
              </svg>
              Schedule Demo
            </a>
          </div>
        </motion.div>
      </div>

      <style>{`
        .cta-section {
          background: var(--bg-primary);
          text-align: center;
          border-top: 1px solid var(--border-light);
        }

        .cta-content {
          max-width: 560px;
          margin: 0 auto;
        }

        .cta-title {
          font-size: clamp(28px, 5vw, 40px);
          font-weight: 700;
          margin-bottom: 16px;
          letter-spacing: -1px;
        }

        .cta-description {
          font-size: 16px;
          color: var(--text-secondary);
          margin-bottom: 32px;
        }

        .cta-buttons {
          display: flex;
          gap: 16px;
          justify-content: center;
          flex-wrap: wrap;
        }
      `}</style>
    </section>
  )
}

export default CTA
