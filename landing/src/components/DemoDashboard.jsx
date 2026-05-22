import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'

function DemoDashboard({ data }) {
  const [dashboardData, setDashboardData] = useState(data)

  useEffect(() => {
    if (data) {
      setDashboardData(data)
    }
  }, [data])

  // Format number with commas
  const formatNumber = (num) => {
    return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",")
  }

  if (!dashboardData) {
    return (
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        transition={{ duration: 0.6 }}
        className="dashboard-loading"
      >
        <div className="dashboard-skeleton">
          <div className="skeleton-line" style={{ width: '60%' }} />
          <div className="skeleton-line" style={{ width: '40%' }} />
          <div className="skeleton-line" style={{ width: '50%' }} />
          <div className="skeleton-line" style={{ width: '70%' }} />
        </div>
      </motion.div>
    )
  }

  const memoryGrowth = dashboardData.memory_growth || {}
  const searchAnalytics = dashboardData.search_analytics || {}
  const agentActivity = dashboardData.agent_activity || []

  return (
    <motion.div
      initial={{ opacity: 0, y: 30 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6 }}
      className="demo-dashboard"
    >
      <div className="dashboard-header">
        <h2 className="dashboard-title">📊 Live Dashboard</h2>
        <p className="dashboard-subtitle">
          Real-time metrics from Neo4j database
        </p>
      </div>

      <div className="dashboard-stats">
        <div className="stat-card">
          <div className="stat-value">{formatNumber(memoryGrowth.TotalCreated || 0)}</div>
          <div className="stat-label">Total Memories</div>
        </div>

        <div className="stat-card">
          <div className="stat-value">{formatNumber(searchAnalytics.TotalSearches || 0)}</div>
          <div className="stat-label">Total Searches</div>
        </div>

        <div className="stat-card">
          <div className="stat-value">{agentActivity.length}</div>
          <div className="stat-label">Active Agents</div>
        </div>

        <div className="stat-card">
          <div className="stat-value">85%</div>
          <div className="stat-label">Avg Compression</div>
        </div>
      </div>

      <div className="dashboard-charts">
        <div className="chart-container">
          <h3 className="chart-title">Memory Growth (7-day trend)</h3>
          <div className="chart-placeholder">
            <div className="sparkline">
              {/* Simple sparkline representation */}
              <div className="sparkline-bar" style={{ height: '20%' }}></div>
              <div className="sparkline-bar" style={{ height: '35%' }}></div>
              <div className="sparkline-bar" style={{ height: '50%' }}></div>
              <div className="sparkline-bar" style={{ height: '40%' }}></div>
              <div className="sparkline-bar" style={{ height: '60%' }}></div>
              <div className="sparkline-bar" style={{ height: '45%' }}></div>
              <div className="sparkline-bar" style={{ height: '55%' }}></div>
              <div className="sparkline-bar" style={{ height: '30%' }}></div>
            </div>
            <p className="chart-label">Daily memory creation</p>
          </div>
        </div>

        <div className="chart-container">
          <h3 className="chart-title">Top Search Queries</h3>
          <div className="search-list">
            {(searchAnalytics.TopQueries || []).slice(0, 5).map((query, idx) => (
              <div className="search-item" key={idx}>
                <span className="search-term">"{query.Query}"</span>
                <span className="search-count">{formatNumber(query.Count)}</span>
              </div>
            ))}
            {!(searchAnalytics.TopQueries || []).length && (
              <p className="no-data">No search data yet</p>
            )}
          </div>
        </div>
      </div>

      <div className="dashboard-footer">
        <p className="dashboard-timestamp">
          Last updated: {new Date().toLocaleTimeString()}
        </p>
        <p className="dashboard-note">
          Data sourced from Neo4j knowledge graph
        </p>
      </div>
    </motion.div>
  )
}

export default DemoDashboard