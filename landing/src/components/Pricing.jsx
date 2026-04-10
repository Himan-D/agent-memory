import { motion } from 'framer-motion'

const plans = [
  {
    name: 'AGPL',
    price: 'Free',
    description: 'Open source license for self-hosting. Perfect for developers and research.',
    features: [
      'Full source code access',
      '1 Agent',
      'Self-hosted deployment',
      'Vector storage (Qdrant/Pinecone)',
      'Knowledge graph (Neo4j)',
      'Community support',
    ],
    cta: 'Get Started',
    highlighted: false,
    badge: 'Open Source',
  },
  {
    name: 'Developer',
    price: '$29',
    period: '/month',
    description: 'For developers building AI applications with multi-agent capabilities.',
    features: [
      'Up to 5 Agents',
      'Agent Groups',
      'Shared Memory Pool',
      'Redis Pub/Sub',
      'Skill Extraction & Synthesis',
      'Email support',
    ],
    cta: 'Buy Developer',
    highlighted: true,
    badge: 'Most Popular',
  },
  {
    name: 'Team',
    price: '$99',
    period: '/month',
    description: 'For teams requiring collaboration and human review workflows.',
    features: [
      'Up to 20 Agents',
      'Human Review Workflows',
      'Audit Logging',
      'Priority Support',
      'Advanced Analytics',
      'Custom Integrations',
    ],
    cta: 'Buy Team',
    highlighted: false,
    badge: null,
  },
  {
    name: 'Enterprise',
    price: 'Custom',
    description: 'For organizations with advanced security, compliance, and SLA requirements.',
    features: [
      'Unlimited Agents & Groups',
      'SSO/SAML/OIDC/LDAP',
      'SOC 2 & HIPAA compliance',
      'Dedicated Support & SLA',
      'On-premise deployment',
      'Custom development',
    ],
    cta: 'Contact Sales',
    highlighted: false,
    badge: null,
  },
]

function Pricing() {
  return (
    <section className="pricing-section section" id="pricing">
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="section-header"
        >
          <span className="section-label">Pricing</span>
          <h2 className="section-title">Simple and affordable</h2>
          <p className="section-description">
            Start free, scale as you grow. No hidden fees.
          </p>
        </motion.div>

        <div className="pricing-grid">
          {plans.map((plan, index) => (
            <motion.div
              key={plan.name}
              initial={{ opacity: 0, y: 30 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.5, delay: index * 0.1 }}
              className={`pricing-card ${plan.highlighted ? 'highlighted' : ''}`}
            >
              {plan.highlighted && <span className="popular-badge">Most Popular</span>}
              <h3 className="plan-name">{plan.name}</h3>
              <div className="plan-price">
                <span className="price">{plan.price}</span>
                {plan.period && <span className="period">{plan.period}</span>}
              </div>
              <p className="plan-description">{plan.description}</p>
              <ul className="plan-features">
                {plan.features.map((feature, i) => (
                  <li key={i}>
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <polyline points="20 6 9 17 4 12"/>
                    </svg>
                    {feature}
                  </li>
                ))}
              </ul>
              <button className={`plan-cta ${plan.highlighted ? 'btn-primary' : 'btn-secondary'}`}>
                {plan.cta}
              </button>
            </motion.div>
          ))}
        </div>

        <motion.div
          initial={{ opacity: 0 }}
          whileInView={{ opacity: 1 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.4 }}
          className="pricing-guarantee"
        >
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
          </svg>
          <span>All plans include a 14-day free trial. No credit card required.</span>
        </motion.div>
      </div>

      <style>{`
        .pricing-section {
          background: var(--bg-primary);
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
          margin-bottom: 12px;
        }

        .section-description {
          font-size: 16px;
          color: var(--text-secondary);
        }

        .pricing-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
          gap: 24px;
          max-width: 1000px;
          margin: 0 auto;
        }

        .pricing-card {
          position: relative;
          padding: 32px;
          background: var(--bg-surface);
          border: 1px solid var(--border-light);
          border-radius: 16px;
          transition: all 0.3s ease;
        }

        .pricing-card:hover {
          border-color: rgba(37, 99, 235, 0.3);
        }

        .pricing-card.highlighted {
          border-color: #2563EB;
          background: linear-gradient(135deg, rgba(37, 99, 235, 0.05) 0%, transparent 100%);
        }

        .popular-badge {
          position: absolute;
          top: -12px;
          left: 50%;
          transform: translateX(-50%);
          padding: 6px 16px;
          background: #2563EB;
          color: white;
          font-size: 12px;
          font-weight: 600;
          border-radius: 100px;
        }

        .plan-name {
          font-family: var(--font-display);
          font-size: 18px;
          font-weight: 600;
          margin-bottom: 8px;
        }

        .plan-price {
          display: flex;
          align-items: baseline;
          gap: 4px;
          margin-bottom: 8px;
        }

        .price {
          font-family: var(--font-display);
          font-size: 40px;
          font-weight: 800;
        }

        .period {
          font-size: 14px;
          color: var(--text-muted);
        }

        .plan-description {
          font-size: 14px;
          color: var(--text-secondary);
          margin-bottom: 24px;
        }

        .plan-features {
          list-style: none;
          margin-bottom: 24px;
        }

        .plan-features li {
          display: flex;
          align-items: center;
          gap: 10px;
          font-size: 14px;
          color: var(--text-secondary);
          margin-bottom: 12px;
        }

        .plan-features li svg {
          color: #2563EB;
          flex-shrink: 0;
        }

        .plan-cta {
          width: 100%;
          padding: 14px 24px;
          font-size: 15px;
          font-weight: 600;
          border-radius: 8px;
          border: none;
          cursor: pointer;
          transition: all 0.3s ease;
        }

        .plan-cta.btn-primary {
          background: #2563EB;
          color: white;
        }

        .plan-cta.btn-primary:hover {
          background: #1d4ed8;
        }

        .plan-cta.btn-secondary {
          background: transparent;
          color: var(--text-primary);
          border: 1px solid var(--border-medium);
        }

        .plan-cta.btn-secondary:hover {
          border-color: #2563EB;
          color: #2563EB;
        }

        .pricing-guarantee {
          display: flex;
          align-items: center;
          justify-content: center;
          gap: 8px;
          margin-top: 32px;
          font-size: 13px;
          color: var(--text-muted);
        }

        .pricing-guarantee svg {
          color: #2563EB;
        }

        @media (max-width: 768px) {
          .pricing-grid {
            grid-template-columns: 1fr;
            max-width: 400px;
          }
        }
      `}</style>
    </section>
  )
}

export default Pricing