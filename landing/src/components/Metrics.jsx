import { motion } from 'framer-motion'

const stats = [
  { value: '97%+', label: 'Accuracy Retention', description: 'ProMem extraction' },
  { value: '85-93%', label: 'Token Reduction', description: 'Proprietary compression' },
  { value: '+23%', label: 'Multi-hop Reasoning', description: 'Spreading activation' },
  { value: '<100ms', label: 'Vector Search', description: 'Semantic query latency' }
]

const logos = ['Python', 'Node.js', 'LangChain', 'CrewAI', 'Mastra', 'Agno']

function Metrics() {
  return (
    <section className="metrics-section">
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="metrics-stats"
        >
          {stats.map((stat, index) => (
            <motion.div
              key={index}
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.5, delay: index * 0.1 }}
              className="stat-card"
            >
              <span className="stat-value">{stat.value}</span>
              <span className="stat-label">{stat.label}</span>
              <span className="stat-description">{stat.description}</span>
            </motion.div>
          ))}
        </motion.div>

        <motion.div
          initial={{ opacity: 0 }}
          whileInView={{ opacity: 1 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.4 }}
          className="metrics-trust"
        >
          <span className="trust-label">Works with</span>
          <div className="trust-logos">
            {logos.map((logo, index) => (
              <span key={index} className="trust-logo">{logo}</span>
            ))}
          </div>
        </motion.div>
      </div>

      <style>{`
        .metrics-section {
          padding: 48px 0;
          background: var(--bg-secondary);
          border-top: 1px solid var(--border-light);
          border-bottom: 1px solid var(--border-light);
        }

        .metrics-stats {
          display: grid;
          grid-template-columns: repeat(4, 1fr);
          gap: 24px;
          margin-bottom: 32px;
        }

        @media (max-width: 900px) {
          .metrics-stats {
            grid-template-columns: repeat(2, 1fr);
          }
        }

        @media (max-width: 480px) {
          .metrics-stats {
            grid-template-columns: 1fr;
          }

          .trust-logos {
            flex-wrap: wrap;
            justify-content: center;
          }
        }

        .stat-card {
          text-align: center;
          padding: 24px 16px;
          background: var(--card-bg);
          border: 1px solid var(--border-light);
          border-radius: 8px;
        }

        .stat-value {
          display: block;
          font-size: 32px;
          font-weight: 700;
          margin-bottom: 4px;
        }

        .stat-label {
          display: block;
          font-size: 14px;
          font-weight: 600;
          margin-bottom: 4px;
        }

        .stat-description {
          display: block;
          font-size: 12px;
          color: var(--text-secondary);
        }

        .metrics-trust {
          text-align: center;
        }

        .trust-label {
          display: block;
          font-size: 11px;
          color: var(--text-muted);
          margin-bottom: 16px;
          text-transform: uppercase;
          letter-spacing: 2px;
        }

        .trust-logos {
          display: flex;
          justify-content: center;
          align-items: center;
          gap: 32px;
          flex-wrap: wrap;
        }

        .trust-logo {
          font-size: 14px;
          font-weight: 500;
          color: var(--text-secondary);
        }
      `}</style>
    </section>
  )
}

export default Metrics