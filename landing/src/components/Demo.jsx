import { useEffect } from 'react'
import { motion } from 'framer-motion'
import Waitlist from './Waitlist'

function Demo() {
  useEffect(() => {
    const script = document.createElement('script')
    script.src = 'https://assets.calendly.com/assets/external/widget.js'
    script.async = true
    document.head.appendChild(script)
    return () => {
      const existing = document.querySelector(`script[src="https://assets.calendly.com/assets/external/widget.js"]`)
      if (existing) existing.remove()
    }
  }, [])

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
            <h2 className="demo-title">Schedule a Demo</h2>
            <p className="demo-description">
              See Hystersis in action. Book a 30-minute call with our team.
            </p>
            <div className="waitlist-container">
              <Waitlist />
            </div>
          </div>
          <div className="calendly-container">
            <div
              className="calendly-inline-widget"
              data-url="https://calendly.com/hystersis/30min"
              style={{ minWidth: '320px', height: '630px' }}
            />
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
