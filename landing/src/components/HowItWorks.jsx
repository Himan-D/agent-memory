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
            </motion.div>
          ))}
        </div>
      </div>

      <style>{`
        .how-section {
          background: var(--bg-secondary);
          border-top: 1px solid var(--border-light);
        }

        .section-header {
          text-align: center;
          margin-bottom: 60px;
        }

        .steps-container {
          display: flex;
          justify-content: center;
          align-items: stretch;
          gap: 24px;
          flex-wrap: wrap;
          max-width: 900px;
          margin: 0 auto;
        }

        .step-card {
          flex: 1;
          min-width: 240px;
          max-width: 280px;
          padding: 32px 24px;
          background: var(--card-bg);
          border: 1px solid var(--border-light);
          border-radius: 12px;
          text-align: center;
          position: relative;
          transition: all 0.3s ease;
        }

        .step-card:hover {
          border-color: var(--text-primary);
        }

        .step-number {
          font-size: 14px;
          font-weight: 700;
          color: var(--text-primary);
          margin-bottom: 16px;
        }

        .step-title {
          font-size: 22px;
          font-weight: 700;
          margin-bottom: 8px;
        }

        .step-description {
          font-size: 14px;
          color: var(--text-secondary);
          line-height: 1.5;
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
