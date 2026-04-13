import { motion } from 'framer-motion'

const plans = [
  {
    name: 'Self-Hosted',
    price: 'Free',
    description: 'Full-featured self-hosted option for individuals and small projects.',
    features: [
      '1 Seat',
      'Unlimited Agents',
      'Self-hosted deployment',
      'Vector storage (Qdrant)',
      'Knowledge graph (Neo4j)',
      'Community support',
    ],
    cta: 'Start Free',
    highlighted: false
  },
  {
    name: 'Pro',
    price: '$29',
    period: '/seat/month',
    description: 'For professional developers building production AI applications.',
    features: [
      '5 Seats',
      'Unlimited Agents',
      'Agent Groups',
      'Shared Memory Pool',
      'Skill Extraction & Synthesis',
      'Priority Email support',
    ],
    cta: 'Get Pro',
    highlighted: true
  },
  {
    name: 'Team',
    price: '$99',
    period: '/seat/month',
    description: 'For teams requiring collaboration, audit logs, and analytics.',
    features: [
      '20 Seats',
      'Unlimited Agents',
      'Human Review Workflows',
      'Audit Logging',
      'Advanced Analytics',
      'Custom Integrations',
      'Priority support',
    ],
    cta: 'Get Team',
    highlighted: false
  },
  {
    name: 'Enterprise',
    price: 'Custom',
    description: 'For organizations with advanced security, compliance, and SLA requirements.',
    features: [
      'Unlimited Seats',
      'Unlimited Agents',
      'SSO/SAML/OIDC/LDAP',
      'SOC 2 & HIPAA compliance',
      'Dedicated Support & SLA',
      'On-premise deployment',
      'Custom development',
    ],
    cta: 'Contact Sales',
    highlighted: false
  }
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
          <h2 className="section-title">Simple, transparent pricing</h2>
          <p className="section-description">
            Start free. Scale as you grow. No hidden fees.
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
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
          </svg>
          <span>14-day free trial on all paid plans. No credit card required.</span>
        </motion.div>
      </div>

      <style>{`
        .pricing-section {
          background: var(--bg-secondary);
          border-top: 1px solid var(--border-light);
        }

        .section-header {
          text-align: center;
          margin-bottom: 48px;
        }

        .pricing-grid {
          display: grid;
          grid-template-columns: repeat(4, 1fr);
          gap: 24px;
          max-width: 1200px;
          margin: 0 auto;
        }

        .pricing-card {
          position: relative;
          padding: 32px 24px;
          background: var(--card-bg);
          border: 1px solid var(--border-light);
          border-radius: 12px;
          transition: all 0.3s ease;
          display: flex;
          flex-direction: column;
        }

        .pricing-card:hover {
          border-color: var(--text-primary);
          transform: translateY(-4px);
        }

        .pricing-card.highlighted {
          border-color: var(--text-primary);
          border-width: 2px;
        }

        .popular-badge {
          position: absolute;
          top: -12px;
          left: 50%;
          transform: translateX(-50%);
          padding: 6px 16px;
          background: var(--text-primary);
          color: var(--bg-primary);
          font-size: 11px;
          font-weight: 600;
          border-radius: 100px;
        }

        .plan-name {
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
          font-size: 40px;
          font-weight: 700;
        }

        .period {
          font-size: 14px;
          color: var(--text-secondary);
        }

        .plan-description {
          font-size: 14px;
          color: var(--text-secondary);
          margin-bottom: 24px;
          flex-grow: 1;
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
          color: var(--text-primary);
          flex-shrink: 0;
        }

        .plan-cta {
          width: 100%;
          padding: 14px 24px;
          font-size: 14px;
          font-weight: 600;
          border-radius: 8px;
          border: none;
          cursor: pointer;
          transition: all 0.3s ease;
          margin-top: auto;
        }

        .plan-cta.btn-primary {
          background: var(--text-primary);
          color: var(--bg-primary);
        }

        .plan-cta.btn-primary:hover {
          opacity: 0.8;
        }

        .plan-cta.btn-secondary {
          background: var(--card-bg);
          color: var(--text-primary);
          border: 1px solid var(--border-light);
        }

        .plan-cta.btn-secondary:hover {
          border-color: var(--text-primary);
        }

        .pricing-guarantee {
          display: flex;
          align-items: center;
          justify-content: center;
          gap: 8px;
          margin-top: 48px;
          font-size: 13px;
          color: var(--text-secondary);
        }

        @media (max-width: 1024px) {
          .pricing-grid {
            grid-template-columns: repeat(2, 1fr);
            max-width: 600px;
          }
        }

        @media (max-width: 640px) {
          .pricing-grid {
            grid-template-columns: 1fr;
            max-width: 400px;
          }

          .price {
            font-size: 32px;
          }

          .pricing-card {
            padding: 24px 20px;
          }
        }
      `}</style>
    </section>
  )
}

export default Pricing
