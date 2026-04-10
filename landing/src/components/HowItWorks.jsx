import { motion } from 'framer-motion'

const steps = [
  { num: '01', title: 'Store', description: 'Agent stores messages, entities, and relationships' },
  { num: '02', title: 'Embed', description: 'Content is embedded using OpenAI (or custom)' },
  { num: '03', title: 'Search', description: 'Query using natural language' }
]

function HowItWorks() {
  return (
    <section className="how-section section">
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="section-header"
        >
          <span className="section-label">How It Works</span>
          <h2 className="section-title">Three simple steps</h2>
        </motion.div>

        <div className="steps-container">
          {steps.map((step, index) => (
            <motion.div
              key={step.num}
              className="step-card"
              initial={{ opacity: 0, y: 40 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.6, delay: index * 0.15 }}
            >
              <div className="step-number">{step.num}</div>
              <h3 className="step-title">{step.title}</h3>
              <p className="step-description">{step.description}</p>
              {index < steps.length - 1 && (
                <div className="step-arrow">
                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M5 12h14M12 5l7 7-7 7"/>
                  </svg>
                </div>
              )}
            </motion.div>
          ))}
        </div>
      </div>

      <style>{`
        .how-section {
          background: var(--bg-primary);
        }

        .section-header {
          text-align: center;
          margin-bottom: 60px;
        }

        .section-title {
          font-family: var(--font-display);
          font-size: clamp(28px, 5vw, 40px);
          font-weight: 700;
          letter-spacing: -1px;
        }

        .steps-container {
          display: flex;
          justify-content: center;
          align-items: stretch;
          gap: 16px;
          flex-wrap: wrap;
          max-width: 900px;
          margin: 0 auto;
        }

        .step-card {
          flex: 1;
          min-width: 240px;
          max-width: 280px;
          padding: 32px 24px;
          background: var(--bg-surface);
          border: 1px solid var(--border-light);
          border-radius: 16px;
          text-align: center;
          position: relative;
          transition: all 0.3s ease;
        }

        .step-card:hover {
          border-color: rgba(37, 99, 235, 0.3);
          transform: translateY(-4px);
        }

        .step-number {
          font-family: var(--font-display);
          font-size: 14px;
          font-weight: 700;
          color: #2563EB;
          margin-bottom: 16px;
        }

        .step-title {
          font-family: var(--font-display);
          font-size: 22px;
          font-weight: 700;
          margin-bottom: 8px;
        }

        .step-description {
          font-size: 14px;
          color: var(--text-secondary);
          line-height: 1.5;
        }

        .step-arrow {
          position: absolute;
          right: -20px;
          top: 50%;
          transform: translateY(-50%);
          color: var(--text-muted);
          display: none;
        }

        @media (min-width: 900px) {
          .step-arrow {
            display: block;
          }
        }

        @media (max-width: 768px) {
          .steps-container {
            flex-direction: column;
            align-items: center;
          }

          .step-card {
            max-width: 100%;
            width: 100%;
          }
        }
      `}</style>
    </section>
  )
}

export default HowItWorks