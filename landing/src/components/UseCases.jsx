import { motion } from 'framer-motion'

const useCases = [
  {
    title: 'Customer Support',
    description: 'Remember past tickets and customer history. Resolve issues faster.',
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 01-3.46 0"/>
      </svg>
    )
  },
  {
    title: 'Code Assistant',
    description: 'Index codebase patterns. Understand your project structure.',
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M16 18l6-6-6-6M8 6l-6 6 6 6"/>
      </svg>
    )
  },
  {
    title: 'Research Agent',
    description: 'Build literature graphs. Connect ideas across papers.',
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M4 19.5A2.5 2.5 0 016.5 17H20"/>
        <path d="M6.5 2H20v20H6.5A2.5 2.5 0 014 19.5v-15A2.5 2.5 0 016.5 2z"/>
      </svg>
    )
  },
  {
    title: 'Personal Assistant',
    description: 'Remember preferences and important dates.',
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2"/>
        <circle cx="12" cy="7" r="4"/>
      </svg>
    )
  }
]

function UseCases() {
  return (
    <section className="usecases-section section">
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="section-header"
        >
          <span className="section-label">Use Cases</span>
          <h2 className="section-title">Built for real applications</h2>
        </motion.div>

        <motion.div
          initial={{ opacity: 0 }}
          whileInView={{ opacity: 1 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.2 }}
          className="usecases-grid"
        >
          {useCases.map((useCase, index) => (
            <motion.div
              key={index}
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.5, delay: index * 0.1 }}
              className="usecase-card"
            >
              <div className="usecase-icon">{useCase.icon}</div>
              <h3 className="usecase-title">{useCase.title}</h3>
              <p className="usecase-description">{useCase.description}</p>
            </motion.div>
          ))}
        </motion.div>
      </div>

      <style>{`
        .usecases-section {
          background: var(--bg-surface);
          overflow: hidden;
        }

        .section-header {
          text-align: center;
          margin-bottom: 48px;
        }

        .section-title {
          font-family: var(--font-display);
          font-size: clamp(28px, 5vw, 40px);
          font-weight: 700;
          letter-spacing: -1px;
        }

        .usecases-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
          gap: 20px;
          max-width: 900px;
          margin: 0 auto;
        }

        .usecase-card {
          padding: 28px;
          background: var(--bg-primary);
          border: 1px solid rgba(255, 255, 255, 0.06);
          border-radius: 12px;
          transition: all 0.3s ease;
        }

        .usecase-card:hover {
          border-color: rgba(37, 99, 235, 0.3);
          transform: translateY(-2px);
        }

        .usecase-icon {
          width: 44px;
          height: 44px;
          display: flex;
          align-items: center;
          justify-content: center;
          background: rgba(37, 99, 235, 0.1);
          border-radius: 10px;
          margin-bottom: 16px;
        }

        .usecase-icon svg {
          width: 22px;
          height: 22px;
          color: var(--color-primary);
        }

        .usecase-title {
          font-family: var(--font-display);
          font-size: 17px;
          font-weight: 600;
          margin-bottom: 8px;
        }

        .usecase-description {
          font-size: 14px;
          color: var(--text-secondary);
          line-height: 1.6;
        }
      `}</style>
    </section>
  )
}

export default UseCases