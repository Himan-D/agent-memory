import { motion } from 'framer-motion'

const features = [
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8s-9-3.582-9-8 4.03-8 9-8 9 3.582 9 8z"/>
      </svg>
    ),
    title: 'Conversational Memory',
    description: 'Session-based message history that remembers every conversation. Store and retrieve past messages with semantic search.'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <circle cx="12" cy="12" r="3"/>
        <circle cx="19" cy="5" r="2"/>
        <circle cx="5" cy="5" r="2"/>
        <circle cx="19" cy="19" r="2"/>
        <circle cx="5" cy="19" r="2"/>
        <path d="M12 9V5M9 12H5M12 15v4M15 12h4"/>
      </svg>
    ),
    title: 'Knowledge Graph',
    description: 'Entities with typed relationships. Connect concepts, track connections, and traverse your knowledge graph.'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
        <path d="M10 7v6M7 10h6"/>
      </svg>
    ),
    title: 'Semantic Search',
    description: 'Vector-based similarity search. Find semantically similar content using OpenAI embeddings.'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <rect x="3" y="3" width="18" height="18" rx="2"/>
        <path d="M3 9h18M9 21V9"/>
      </svg>
    ),
    title: 'Multi-Tenant',
    description: 'Separate memory for different agents and tenants. Complete data isolation with API key mapping.'
  }
]

const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.1
    }
  }
}

const itemVariants = {
  hidden: { opacity: 0, y: 30 },
  visible: {
    opacity: 1,
    y: 0,
    transition: {
      duration: 0.6,
      ease: [0.25, 0.46, 0.45, 0.94]
    }
  }
}

function Features() {
  return (
    <section className="features-section section">
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="section-header"
        >
          <span className="section-label">Features</span>
          <h2 className="section-title">Everything you need to build memory-powered agents</h2>
          <p className="section-description">
            Four powerful memory types working together to create truly intelligent agents.
          </p>
        </motion.div>

        <motion.div
          variants={containerVariants}
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-100px" }}
          className="features-grid"
        >
          {features.map((feature, index) => (
            <motion.div key={index} variants={itemVariants} className="feature-card">
              <div className="feature-icon">{feature.icon}</div>
              <h3 className="feature-title">{feature.title}</h3>
              <p className="feature-description">{feature.description}</p>
              <div className="feature-glow" />
            </motion.div>
          ))}
        </motion.div>
      </div>

      <style>{`
        .features-section {
          background: var(--bg-primary);
        }

        .section-header {
          text-align: center;
          margin-bottom: 64px;
        }

        .section-title {
          font-family: var(--font-display);
          font-size: clamp(28px, 5vw, 40px);
          font-weight: 700;
          margin-bottom: 16px;
          letter-spacing: -1px;
        }

        .section-description {
          font-size: 16px;
          color: var(--text-secondary);
          max-width: 500px;
          margin: 0 auto;
        }

        .features-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
          gap: 24px;
        }

        .feature-card {
          position: relative;
          padding: 32px;
          background: rgba(26, 26, 26, 0.6);
          border: 1px solid rgba(255, 255, 255, 0.08);
          border-radius: 16px;
          transition: all 0.4s ease;
          overflow: hidden;
        }

        .feature-card:hover {
          border-color: rgba(37, 99, 235, 0.3);
          transform: translateY(-4px);
        }

        .feature-card:hover .feature-glow {
          opacity: 1;
        }

        .feature-glow {
          position: absolute;
          top: 0;
          left: 0;
          right: 0;
          height: 100px;
          background: radial-gradient(ellipse at top, rgba(37, 99, 235, 0.15) 0%, transparent 70%);
          opacity: 0;
          transition: opacity 0.4s ease;
          pointer-events: none;
        }

        .feature-icon {
          width: 48px;
          height: 48px;
          display: flex;
          align-items: center;
          justify-content: center;
          background: linear-gradient(135deg, rgba(37, 99, 235, 0.15) 0%, rgba(255, 107, 91, 0.1) 100%);
          border-radius: 12px;
          margin-bottom: 20px;
        }

        .feature-icon svg {
          width: 24px;
          height: 24px;
          color: var(--color-primary);
        }

        .feature-title {
          font-family: var(--font-display);
          font-size: 18px;
          font-weight: 600;
          margin-bottom: 12px;
        }

        .feature-description {
          font-size: 14px;
          color: var(--text-secondary);
          line-height: 1.7;
        }
      `}</style>
    </section>
  )
}

export default Features