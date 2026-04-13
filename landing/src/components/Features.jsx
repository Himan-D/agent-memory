import { motion } from 'framer-motion'

const features = [
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"/>
      </svg>
    ),
    title: 'Procedural Memory',
    description: 'Agents learn reusable skills from interactions. Auto-extract, synthesize, and apply patterns across conversations.'
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
    description: 'Vector-based similarity search. Find semantically similar content using OpenAI embeddings with 85% compression.'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
      </svg>
    ),
    title: 'Multi-Agent Pool',
    description: 'Real-time shared memory between agents. Agent groups with pub/sub sync for collaborative AI systems.'
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
      duration: 0.5,
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
          <h2 className="section-title">Build agents that actually remember</h2>
          <p className="section-description">
            Four memory systems working together to create intelligent, persistent AI agents.
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
            </motion.div>
          ))}
        </motion.div>
      </div>

      <style>{`
        .features-section {
          background: var(--bg-primary);
          border-top: 1px solid var(--border-light);
        }

        .section-header {
          text-align: center;
          margin-bottom: 64px;
        }

        .feature-card {
          padding: 32px;
          border: 1px solid var(--border-light);
          border-radius: 12px;
          background: var(--card-bg);
          transition: all 0.3s ease;
        }

        .feature-card:hover {
          border-color: var(--text-primary);
        }

        .feature-icon {
          width: 48px;
          height: 48px;
          display: flex;
          align-items: center;
          justify-content: center;
          margin-bottom: 20px;
        }

        .feature-icon svg {
          width: 24px;
          height: 24px;
          color: var(--text-primary);
        }

        .feature-title {
          font-size: 18px;
          font-weight: 600;
          margin-bottom: 12px;
        }

        .feature-description {
          font-size: 14px;
          color: var(--text-secondary);
          line-height: 1.7;
        }

        .features-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
          gap: 24px;
        }
      `}</style>
    </section>
  )
}

export default Features
