import { motion } from 'framer-motion'
import CalendlyWidget from './CalendlyWidget'

function Demo() {
  return (
    <section className="demo-section section" id="demo">
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="demo-content"
        >
          <div className="demo-header">
            <h2 className="demo-title">See Hystersis in Action</h2>
            <p className="demo-description">
              Schedule a personalized demo to see how Hystersis can transform your AI agents with persistent memory, semantic search, and real-time multi-agent sync.
            </p>
            <div className="demo-features">
              <div className="demo-feature">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
                </svg>
                <span>Enterprise-grade security</span>
              </div>
              <div className="demo-feature">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <circle cx="12" cy="12" r="10"/>
                  <polyline points="12 6 12 12 16 14"/>
                </svg>
                <span>30-minute focused session</span>
              </div>
              <div className="demo-feature">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
                  <circle cx="9" cy="7" r="4"/>
                  <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
                  <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
                </svg>
                <span>Tailored to your use case</span>
              </div>
            </div>
          </div>
          <div className="calendly-container">
            <CalendlyWidget />
          </div>
        </motion.div>
      </div>

      <style>{`
        .demo-section {
          background: var(--bg-primary);
          border-top: 1px solid var(--border-light);
        }

        .demo-content {
          display: grid;
          grid-template-columns: 1fr 2fr;
          gap: 48px;
          align-items: start;
        }

        .demo-header {
          position: sticky;
          top: 100px;
        }

        .demo-features {
          display: flex;
          flex-direction: column;
          gap: 16px;
        }

        .demo-feature {
          display: flex;
          align-items: center;
          gap: 12px;
          font-size: 15px;
          color: var(--text-secondary);
        }

        .demo-feature svg {
          color: var(--text-primary);
          flex-shrink: 0;
        }

        .demo-title {
          font-size: clamp(28px, 4vw, 36px);
          font-weight: 700;
          margin-bottom: 16px;
          letter-spacing: -1px;
        }

        .demo-description {
          font-size: 16px;
          color: var(--text-secondary);
          margin-bottom: 24px;
          line-height: 1.6;
        }

        .waitlist-container {
          display: flex;
          gap: 12px;
        }

        .waitlist-trigger {
          display: inline-flex;
          align-items: center;
          justify-content: center;
          padding: 12px 24px;
          background: var(--bg-secondary);
          border: 1px solid var(--border-light);
          border-radius: 8px;
          font-size: 14px;
          font-weight: 500;
          color: var(--text-primary);
          cursor: pointer;
          transition: all 0.2s ease;
        }

        .waitlist-trigger:hover {
          background: var(--bg-tertiary);
          border-color: var(--border-medium);
        }

        .calendly-container {
          background: white;
          border-radius: 16px;
          overflow: hidden;
          box-shadow: 0 4px 24px rgba(0, 0, 0, 0.08);
        }

        @media (max-width: 768px) {
          .demo-content {
            grid-template-columns: 1fr;
          }

          .demo-header {
            position: static;
          }
        }
      `}</style>
    </section>
  )
}

export default Demo
