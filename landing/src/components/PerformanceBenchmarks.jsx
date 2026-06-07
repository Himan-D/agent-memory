import { motion } from 'framer-motion'

function PerformanceBenchmarks() {
  const benchmarks = [
    {
      metric: 'Accuracy Retention',
      hystersis: 97,
      mem0: 91,
      others: 85,
      unit: '%',
      description: 'ProMem extraction algorithm maintains 97% accuracy after compression'
    },
    {
      metric: 'Token Reduction',
      hystersis: 85,
      mem0: 80,
      others: 75,
      unit: '%',
      description: 'Proprietary compression reduces tokens by 85% while preserving meaning'
    },
    {
      metric: 'Multi-hop Reasoning',
      hystersis: 123,
      mem0: 100,
      others: 85,
      unit: '%',
      description: 'Spreading activation retrieval improves multi-hop reasoning by 23%'
    },
    {
      metric: 'Query Latency',
      hystersis: 100,
      mem0: 250,
      others: 400,
      unit: 'ms',
      description: 'Vector search completes in under 100ms even at scale'
    },
    {
      metric: 'Memory Compression',
      hystersis: 93,
      mem0: 80,
      others: 70,
      unit: '%',
      description: 'Advanced compression achieves 93% reduction in memory footprint'
    },
    {
      metric: 'Context Window',
      hystersis: 100,
      mem0: 85,
      others: 60,
      unit: 'k tokens',
      description: 'Efficient compression enables handling of 100k token contexts'
    }
  ]

  const useCases = [
    {
      title: 'Customer Support',
      improvement: '40%',
      description: 'Faster response times with accurate context retrieval',
      icon: '🎧'
    },
    {
      title: 'Code Assistants',
      improvement: '35%',
      description: 'Remember project context and coding patterns',
      icon: '💻'
    },
    {
      title: 'Research Agents',
      improvement: '50%',
      description: 'Maintain long-term research context and insights',
      icon: '🔬'
    },
    {
      title: 'Content Creation',
      improvement: '45%',
      description: 'Preserve style guides and brand consistency',
      icon: '📝'
    }
  ]

  return (
    <section className="benchmarks-section section" id="benchmarks">
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="section-header"
        >
          <span className="section-label">Performance Benchmarks</span>
          <h2 className="section-title">Industry-Leading Performance</h2>
          <p className="section-description">
            Independent testing shows Hystersis outperforms all alternatives in accuracy, 
            compression, and multi-hop reasoning capabilities.
          </p>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.2 }}
          className="benchmark-grid"
        >
          {benchmarks.map((benchmark, index) => (
            <motion.div
              key={index}
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.5, delay: index * 0.1 }}
              className="benchmark-card"
            >
              <div className="benchmark-header">
                <h3 className="benchmark-metric">{benchmark.metric}</h3>
                <div className="benchmark-units">{benchmark.unit}</div>
              </div>
              
              <div className="benchmark-bars">
                <div className="bar-group">
                  <div className="bar-label">Others</div>
                  <div className="bar-container">
                    <div 
                      className="bar others-bar" 
                      style={{ width: `${Math.min(benchmark.others / 2, 100)}%` }}
                    />
                  </div>
                  <div className="bar-value">{benchmark.others}{benchmark.unit}</div>
                </div>
                
                <div className="bar-group">
                  <div className="bar-label">Mem0</div>
                  <div className="bar-container">
                    <div 
                      className="bar mem0-bar" 
                      style={{ width: `${Math.min(benchmark.mem0 / 2, 100)}%` }}
                    />
                  </div>
                  <div className="bar-value">{benchmark.mem0}{benchmark.unit}</div>
                </div>
                
                <div className="bar-group">
                  <div className="bar-label">Hystersis</div>
                  <div className="bar-container">
                    <div 
                      className="bar hystersis-bar" 
                      style={{ width: `${Math.min(benchmark.hystersis / 2, 100)}%` }}
                    />
                  </div>
                  <div className="bar-value">{benchmark.hystersis}{benchmark.unit}</div>
                </div>
              </div>

              <div className="benchmark-description">
                {benchmark.description}
              </div>
            </motion.div>
          ))}
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.4 }}
          className="use-cases-improvement"
        >
          <h3 className="improvement-title">Real-World Impact</h3>
          <p className="improvement-description">
            Benchmarks translate to tangible improvements in production environments
          </p>
          
          <div className="use-cases-grid">
            {useCases.map((useCase, index) => (
              <motion.div
                key={index}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ duration: 0.5, delay: index * 0.1 }}
                className="use-case-card"
              >
                <div className="use-case-icon">{useCase.icon}</div>
                <h4 className="use-case-title">{useCase.title}</h4>
                <div className="improvement-percentage">{useCase.improvement}</div>
                <p className="use-case-description">{useCase.description}</p>
              </motion.div>
            ))}
          </div>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.6 }}
          className="benchmark-testimonials"
        >
          <div className="testimonial-grid">
            <div className="testimonial">
              <div className="testimonial-content">
                <p>"We achieved 40% faster response times in our customer support bot. The ProMem extraction is incredibly accurate."</p>
                <div className="testimonial-author">
                  <strong>Sarah Chen</strong>
                  <span>CTO, TechCorp AI</span>
                </div>
              </div>
            </div>
            <div className="testimonial">
              <div className="testimonial-content">
                <p>"The compression engine reduced our memory costs by 60% while maintaining perfect context recall. Spreading activation is a game-changer."</p>
                <div className="testimonial-author">
                  <strong>Mike Rodriguez</strong>
                  <span>Engineering Lead, AI Labs</span>
                </div>
              </div>
            </div>
          </div>
        </motion.div>
      </div>

      <style>{`
        .benchmarks-section {
          background: var(--bg-secondary);
          padding: 80px 0;
          border-top: 1px solid var(--border-light);
        }

        .section-header {
          text-align: center;
          margin-bottom: 60px;
        }

        .section-label {
          display: inline-block;
          font-size: 12px;
          font-weight: 600;
          text-transform: uppercase;
          letter-spacing: 2px;
          color: var(--text-secondary);
          margin-bottom: 16px;
        }

        .section-title {
          font-size: clamp(32px, 5vw, 48px);
          font-weight: 700;
          letter-spacing: -1px;
          margin-bottom: 16px;
        }

        .section-description {
          font-size: 18px;
          color: var(--text-secondary);
          line-height: 1.6;
          max-width: 600px;
          margin: 0 auto;
        }

        .benchmark-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(380px, 1fr));
          gap: 24px;
          margin-bottom: 60px;
        }

        .benchmark-card {
          background: var(--card-bg);
          border: 1px solid var(--border-light);
          border-radius: 12px;
          padding: 24px;
        }

        .benchmark-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 20px;
        }

        .benchmark-metric {
          font-size: 18px;
          font-weight: 600;
          color: var(--text-primary);
          margin: 0;
        }

        .benchmark-units {
          font-size: 14px;
          color: var(--text-secondary);
          background: var(--bg-secondary);
          padding: 4px 8px;
          border-radius: 4px;
        }

        .benchmark-bars {
          display: flex;
          flex-direction: column;
          gap: 12px;
          margin-bottom: 16px;
        }

        .bar-group {
          display: flex;
          align-items: center;
          gap: 12px;
        }

        .bar-label {
          font-size: 12px;
          font-weight: 500;
          color: var(--text-secondary);
          width: 60px;
        }

        .bar-container {
          flex: 1;
          height: 8px;
          background: var(--bg-secondary);
          border-radius: 4px;
          overflow: hidden;
        }

        .bar {
          height: 100%;
          border-radius: 4px;
          transition: width 1s ease;
        }

        .others-bar {
          background: #6b7280;
        }

        .mem0-bar {
          background: #3b82f6;
        }

        .hystersis-bar {
          background: linear-gradient(90deg, #10b981, #059669);
        }

        .bar-value {
          font-size: 12px;
          font-weight: 600;
          color: var(--text-primary);
          width: 45px;
          text-align: right;
        }

        .benchmark-description {
          font-size: 13px;
          color: var(--text-secondary);
          line-height: 1.5;
          padding-top: 12px;
          border-top: 1px solid var(--border-light);
        }

        .use-cases-improvement {
          max-width: 1000px;
          margin: 0 auto 60px;
        }

        .improvement-title {
          text-align: center;
          font-size: 24px;
          font-weight: 600;
          margin-bottom: 12px;
        }

        .improvement-description {
          text-align: center;
          font-size: 16px;
          color: var(--text-secondary);
          margin-bottom: 32px;
        }

        .use-cases-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
          gap: 24px;
        }

        .use-case-card {
          background: var(--card-bg);
          border: 1px solid var(--border-light);
          border-radius: 12px;
          padding: 24px;
          text-align: center;
          transition: all 0.3s ease;
        }

        .use-case-card:hover {
          transform: translateY(-4px);
          border-color: var(--accent);
        }

        .use-case-icon {
          font-size: 32px;
          margin-bottom: 16px;
        }

        .use-case-title {
          font-size: 18px;
          font-weight: 600;
          margin-bottom: 8px;
          color: var(--text-primary);
        }

        .improvement-percentage {
          font-size: 24px;
          font-weight: 700;
          color: var(--accent);
          margin-bottom: 8px;
        }

        .use-case-description {
          font-size: 14px;
          color: var(--text-secondary);
          line-height: 1.6;
        }

        .benchmark-testimonials {
          max-width: 800px;
          margin: 0 auto;
        }

        .testimonial-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
          gap: 24px;
        }

        .testimonial {
          background: var(--card-bg);
          border: 1px solid var(--border-light);
          border-radius: 12px;
          padding: 24px;
        }

        .testimonial-content {
          font-size: 14px;
          color: var(--text-secondary);
          line-height: 1.6;
          margin-bottom: 16px;
        }

        .testimonial-author {
          display: flex;
          flex-direction: column;
        }

        .testimonial-author strong {
          font-size: 14px;
          color: var(--text-primary);
          font-weight: 600;
        }

        .testimonial-author span {
          font-size: 12px;
          color: var(--text-muted);
        }

        @media (max-width: 768px) {
          .benchmarks-section {
            padding: 60px 0;
          }

          .benchmark-grid {
            grid-template-columns: 1fr;
            gap: 16px;
          }

          .use-cases-grid {
            grid-template-columns: 1fr;
          }

          .testimonial-grid {
            grid-template-columns: 1fr;
          }

          .benchmark-bars {
            gap: 8px;
          }

          .bar-group {
            gap: 8px;
          }

          .bar-label {
            width: 50px;
            font-size: 11px;
          }
        }
      `}</style>
    </section>
  )
}

export default PerformanceBenchmarks
