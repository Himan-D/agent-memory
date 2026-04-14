import { motion } from 'framer-motion'
import analytics from '../utils/analytics.js'

function Hero() {
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
            <span className="hero-title-bold">Memory That Adapts.</span>
          </motion.h1>

          <motion.p
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.2 }}
            className="hero-subtitle"
          >
            Persistent context for AI agents.
            <br />
            Intelligence that compounds with every interaction.
          </motion.p>

          <motion.div
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.3 }}
            className="hero-buttons"
          >
            <a href="https://github.com/Himan-D/agent-memory" className="btn btn-primary" target="_blank" rel="noopener noreferrer" onClick={() => analytics.ctaClicked('github_star', 'hero')}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
              </svg>
              View on GitHub
            </a>
            <a href="#demo" className="btn btn-secondary" onClick={() => analytics.ctaClicked('see_demo', 'hero')}>
              See it in Action
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M5 12h14M12 5l7 7-7 7"/>
              </svg>
            </a>
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
          max-width: 700px;
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
          white-space: nowrap;
        }

        .hero-title-bold {
          font-weight: 800;
          white-space: nowrap;
        }

        .hero-subtitle {
          font-size: 18px;
          color: var(--text-secondary);
          max-width: 480px;
          margin: 0 auto 40px;
          line-height: 1.7;
        }

        .hero-buttons {
          display: flex;
          gap: 16px;
          justify-content: center;
          flex-wrap: wrap;
        }

        @media (max-width: 640px) {
          .hero-section {
            padding: 80px 24px;
            min-height: auto;
          }

          .hero-title {
            font-size: clamp(28px, 8vw, 36px);
            white-space: normal;
          }

          .hero-title-bold {
            white-space: normal;
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
