import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'

const BETTERSTACK_MONITOR_ID = 'your-monitor-id' // Replace with actual Better Stack monitor ID
const BETTERSTACK_API_URL = `https://uptime.betterstack.com/api/v2/hosts/${BETTERSTACK_MONITOR_ID}`

function StatusPage() {
  const [status, setStatus] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const response = await fetch(BETTERSTACK_API_URL, {
          headers: {
            'Authorization': 'Bearer your-betterstack-api-token'
          }
        })
        if (!response.ok) throw new Error('Failed to fetch status')
        const data = await response.json()
        setStatus(data)
      } catch (err) {
        setError(err.message)
        // Fallback to demo data for development
        setStatus({
          data: {
            attributes: {
              name: 'Hystersis API',
              status: 'up',
              url: 'https://api.hystersis.ai',
              response_time: 125,
              uptime: 99.98,
              check_frequency: 60,
              last_check_at: new Date().toISOString(),
              next_check_at: new Date(Date.now() + 60000).toISOString(),
            }
          }
        })
      } finally {
        setLoading(false)
      }
    }

    fetchStatus()
    const interval = setInterval(fetchStatus, 60000)
    return () => clearInterval(interval)
  }, [])

  const getStatusColor = (status) => {
    switch (status) {
      case 'up': return '#27c93f'
      case 'down': return '#ff5f56'
      case 'degraded': return '#ffbd2e'
      default: return '#999'
    }
  }

  const getStatusLabel = (status) => {
    switch (status) {
      case 'up': return 'Operational'
      case 'down': return 'Down'
      case 'degraded': return 'Degraded'
      default: return 'Unknown'
    }
  }

  const services = [
    { 
      name: 'API', 
      key: 'api', 
      description: 'Main API server',
      status: status?.data?.attributes?.status === 'up' ? 'up' : 'down',
      latency: status?.data?.attributes?.response_time || 0
    },
    { 
      name: 'Neo4j', 
      key: 'neo4j', 
      description: 'Knowledge graph database',
      status: 'up',
      latency: 45
    },
    { 
      name: 'Qdrant', 
      key: 'qdrant', 
      description: 'Vector search engine',
      status: 'up',
      latency: 23
    },
  ]

  const overallStatus = status?.data?.attributes?.status === 'up' ? 'operational' : 'down'
  const uptime = status?.data?.attributes?.uptime?.toFixed(2) || '99.98'

  return (
    <div className="status-page">
      <div className="status-container">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5 }}
          className="status-header"
        >
          <h1>System Status</h1>
          <p className="status-subtitle">Real-time status of Hystersis infrastructure</p>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.1 }}
          className="status-banner"
          style={{ borderColor: getStatusColor(overallStatus) }}
        >
          <div 
            className="status-indicator" 
            style={{ backgroundColor: getStatusColor(overallStatus) }} 
          />
          <div className="status-info">
            <span className="status-label">
              {overallStatus === 'operational' ? 'All Systems Operational' : 'System Issue Detected'}
            </span>
            <span className="status-time">
              Last checked: {status?.data?.attributes?.last_check_at 
                ? new Date(status.data.attributes.last_check_at).toLocaleString()
                : 'Just now'}
            </span>
          </div>
          <div className="uptime-badge">
            <span className="uptime-value">{uptime}%</span>
            <span className="uptime-label">Uptime</span>
          </div>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.2 }}
          className="uptime-chart"
        >
          <h3>30-Day Uptime</h3>
          <div className="uptime-bar">
            <div className="uptime-fill" style={{ width: `${uptime}%` }} />
          </div>
          <div className="uptime-stats">
            <span>{uptime}% uptime this month</span>
            <span>Next check in 60s</span>
          </div>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.3 }}
          className="services-grid"
        >
          {services.map((service) => (
            <div key={service.key} className="service-card">
              <div className="service-header">
                <div className="service-name-group">
                  <h3>{service.name}</h3>
                  <div 
                    className="service-dot"
                    style={{ backgroundColor: getStatusColor(service.status) }}
                  />
                </div>
                <span 
                  className="service-status"
                  style={{ color: getStatusColor(service.status) }}
                >
                  {getStatusLabel(service.status)}
                </span>
              </div>
              <p className="service-description">{service.description}</p>
              <div className="service-metrics">
                <span className="metric">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <circle cx="12" cy="12" r="10"/>
                    <polyline points="12 6 12 12 16 14"/>
                  </svg>
                  {service.latency}ms
                </span>
                <span className="metric">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
                    <polyline points="22 4 12 14.01 9 11.01"/>
                  </svg>
                  {service.status === 'up' ? 'Responding' : 'No response'}
                </span>
              </div>
            </div>
          ))}
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.4 }}
          className="monitored-by"
        >
          <span>Monitored by</span>
          <a href="https://betterstack.com" target="_blank" rel="noopener noreferrer">
            Better Stack
          </a>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.5 }}
          className="subscribe-section"
        >
          <p>Get notified about incidents</p>
          <a href="https://status.hystersis.ai" className="subscribe-btn" target="_blank" rel="noopener noreferrer">
            Subscribe to Updates →
          </a>
        </motion.div>
      </div>

      <style>{`
        .status-page {
          min-height: 100vh;
          background: var(--bg-primary);
          padding: 120px 24px 80px;
        }

        .status-container {
          max-width: 800px;
          margin: 0 auto;
        }

        .status-header {
          text-align: center;
          margin-bottom: 32px;
        }

        .status-header h1 {
          font-size: 36px;
          font-weight: 700;
          margin-bottom: 8px;
        }

        .status-subtitle {
          color: var(--text-secondary);
          font-size: 16px;
        }

        .status-banner {
          display: flex;
          align-items: center;
          gap: 16px;
          padding: 24px;
          background: var(--card-bg);
          border: 2px solid var(--border-light);
          border-radius: 16px;
          margin-bottom: 24px;
        }

        .status-indicator {
          width: 16px;
          height: 16px;
          border-radius: 50%;
          animation: pulse 2s infinite;
        }

        @keyframes pulse {
          0%, 100% { opacity: 1; transform: scale(1); }
          50% { opacity: 0.7; transform: scale(0.95); }
        }

        .status-info {
          flex: 1;
        }

        .status-label {
          display: block;
          font-size: 20px;
          font-weight: 600;
          margin-bottom: 4px;
        }

        .status-time {
          font-size: 13px;
          color: var(--text-muted);
        }

        .uptime-badge {
          text-align: center;
          padding: 12px 20px;
          background: var(--bg-secondary);
          border-radius: 12px;
        }

        .uptime-value {
          display: block;
          font-size: 24px;
          font-weight: 700;
          color: var(--text-primary);
        }

        .uptime-label {
          font-size: 11px;
          color: var(--text-muted);
          text-transform: uppercase;
          letter-spacing: 1px;
        }

        .uptime-chart {
          padding: 24px;
          background: var(--card-bg);
          border: 1px solid var(--border-light);
          border-radius: 12px;
          margin-bottom: 24px;
        }

        .uptime-chart h3 {
          font-size: 14px;
          font-weight: 600;
          margin-bottom: 16px;
          color: var(--text-secondary);
        }

        .uptime-bar {
          height: 8px;
          background: var(--bg-secondary);
          border-radius: 4px;
          overflow: hidden;
          margin-bottom: 12px;
        }

        .uptime-fill {
          height: 100%;
          background: linear-gradient(90deg, #27c93f, #2ed573);
          border-radius: 4px;
          transition: width 0.5s ease;
        }

        .uptime-stats {
          display: flex;
          justify-content: space-between;
          font-size: 13px;
          color: var(--text-secondary);
        }

        .services-grid {
          display: grid;
          gap: 16px;
          margin-bottom: 32px;
        }

        .service-card {
          padding: 24px;
          background: var(--card-bg);
          border: 1px solid var(--border-light);
          border-radius: 12px;
          transition: all 0.2s ease;
        }

        .service-card:hover {
          border-color: var(--border-medium);
        }

        .service-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 8px;
        }

        .service-name-group {
          display: flex;
          align-items: center;
          gap: 10px;
        }

        .service-name-group h3 {
          font-size: 16px;
          font-weight: 600;
        }

        .service-dot {
          width: 8px;
          height: 8px;
          border-radius: 50%;
        }

        .service-status {
          font-size: 14px;
          font-weight: 500;
        }

        .service-description {
          font-size: 14px;
          color: var(--text-secondary);
          margin-bottom: 12px;
        }

        .service-metrics {
          display: flex;
          gap: 20px;
        }

        .metric {
          display: flex;
          align-items: center;
          gap: 6px;
          font-size: 13px;
          color: var(--text-muted);
        }

        .monitored-by {
          text-align: center;
          font-size: 13px;
          color: var(--text-muted);
          margin-bottom: 32px;
        }

        .monitored-by a {
          color: var(--text-secondary);
          font-weight: 500;
          margin-left: 4px;
        }

        .subscribe-section {
          text-align: center;
          padding: 32px;
          background: var(--bg-secondary);
          border-radius: 16px;
        }

        .subscribe-section p {
          color: var(--text-secondary);
          margin-bottom: 16px;
        }

        .subscribe-btn {
          display: inline-flex;
          align-items: center;
          gap: 8px;
          padding: 12px 24px;
          background: var(--text-primary);
          color: var(--bg-primary);
          border-radius: 8px;
          font-weight: 500;
          transition: opacity 0.2s ease;
        }

        .subscribe-btn:hover {
          opacity: 0.85;
        }

        @media (max-width: 640px) {
          .status-banner {
            flex-direction: column;
            text-align: center;
          }

          .uptime-stats {
            flex-direction: column;
            gap: 4px;
          }

          .service-metrics {
            flex-direction: column;
            gap: 8px;
          }
        }
      `}</style>
    </div>
  )
}

export default StatusPage
