import { motion } from 'framer-motion'

const stats = [
  { value: '90%', label: 'Token Savings', description: 'Reduce prompt tokens significantly' },
  { value: '<100ms', label: 'Vector Search', description: 'Sub-100ms semantic search latency' },
  { value: '50+', label: 'Connection Pool', description: 'Efficient database connections' },
  { value: '100/min', label: 'Rate Limit', description: 'Requests per minute per API key' },
]

const logos = [
  { name: 'OpenAI', width: 70 },
  { name: 'Neo4j', width: 70 },
  { name: 'Qdrant', width: 70 },
  { name: 'LangChain', width: 80 },
  { name: 'CrewAI', width: 70 },
]

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
              <span key={index} className="trust-logo" style={{ width: logo.width }}>
                {logo.name}
              </span>
            ))}
          </div>
        </motion.div>
      </div>

      <style>{`
        .metrics-section {
          padding: 40px 0;
          background: var(--bg-surface);
          border-top: 1px solid var(--border-light);
          border-bottom: 1px solid var(--border-light);
        }

        .metrics-stats {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
          gap: 24px;
          margin-bottom: 32px;
        }

        .stat-card {
          text-align: center;
          padding: 24px 16px;
          background: var(--bg-primary);
          border: 1px solid var(--border-light);
          border-radius: 12px;
        }

        .stat-value {
          display: block;
          font-family: var(--font-display);
          font-size: 36px;
          font-weight: 800;
          color: #2563EB;
          margin-bottom: 4px;
        }

        .stat-label {
          display: block;
          font-size: 14px;
          font-weight: 600;
          color: var(--text-primary);
          margin-bottom: 4px;
        }

        .stat-description {
          display: block;
          font-size: 12px;
          color: var(--text-muted);
        }

        .metrics-trust {
          text-align: center;
        }

        .trust-label {
          display: block;
          font-size: 12px;
          color: var(--text-muted);
          margin-bottom: 16px;
          text-transform: uppercase;
          letter-spacing: 1px;
        }

        .trust-logos {
          display: flex;
          justify-content: center;
          align-items: center;
          gap: 32px;
          flex-wrap: wrap;
        }

        .trust-logo {
          font-family: var(--font-display);
          font-size: 14px;
          font-weight: 600;
          color: var(--text-muted);
          opacity: 0.6;
        }

        @media (max-width: 640px) {
          .metrics-stats {
            grid-template-columns: repeat(2, 1fr);
          }

          .trust-logos {
            gap: 20px;
          }
        }
      `}</style>
    </section>
  )
}

export default Metrics