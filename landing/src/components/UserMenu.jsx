import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useAuth } from '../context/AuthContext'

const DASHBOARD_URL = import.meta.env.VITE_DASHBOARD_URL || 'https://dashboard.hystersis.ai'

export function UserMenu() {
  const [isOpen, setIsOpen] = useState(false)
  const { user, logout } = useAuth()

  const handleLogout = () => {
    logout()
    setIsOpen(false)
  }

  const handleDashboard = () => {
    window.open(DASHBOARD_URL, '_blank')
    setIsOpen(false)
  }

  return (
    <>
      <div className="user-menu-trigger" onClick={() => setIsOpen(true)}>
        <div className="user-avatar">
          {user?.name?.[0]?.toUpperCase() || 'U'}
        </div>
        <span className="user-name">
          {user?.name || user?.email?.split('@')[0]}
        </span>
      </div>

      <AnimatePresence>
        {isOpen && (
          <div className="user-menu-overlay" onClick={() => setIsOpen(false)}>
            <motion.div 
              className="user-menu"
              onClick={(e) => e.stopPropagation()}
              initial={{ opacity: 0, scale: 0.9 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.9 }}
              transition={{ duration: 0.2 }}
            >
              <div className="user-info">
                <div className="user-avatar-large">
                  {user?.name?.[0]?.toUpperCase() || 'U'}
                </div>
                <div className="user-details">
                  <div className="user-name-display">
                    {user?.name || 'User'}
                  </div>
                  <div className="user-email">
                    {user?.email}
                  </div>
                </div>
              </div>

              <div className="user-menu-items">
                <a 
                  href={DASHBOARD_URL}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="menu-item"
                >
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <rect x="3" y="3" width="7" height="7"/>
                    <rect x="14" y="3" width="7" height="7"/>
                    <rect x="14" y="14" width="7" height="7"/>
                    <rect x="3" y="14" width="7" height="7"/>
                  </svg>
                  Dashboard
                </a>

                <button 
                  onClick={handleLogout}
                  className="menu-item logout"
                >
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M9 21H5a2 2 0 01-2-2V5a2 2 0 012-2h4"/>
                    <polyline points="16 17 21 12 16 7"/>
                    <line x1="21" y1="12" x2="9" y2="12"/>
                  </svg>
                  Sign Out
                </button>
              </div>
            </motion.div>
          </div>
        )}
      </AnimatePresence>

      <style>{`
        .user-menu-trigger {
          display: flex;
          align-items: center;
          gap: 8px;
          padding: 8px 12px;
          background: var(--bg-secondary);
          border: 1px solid var(--border-light);
          border-radius: 8px;
          cursor: pointer;
          transition: all 0.2s;
        }

        .user-menu-trigger:hover {
          background: var(--bg-tertiary);
          border-color: var(--border-medium);
        }

        .user-avatar {
          width: 28px;
          height: 28px;
          background: var(--primary);
          color: white;
          border-radius: 50%;
          display: flex;
          align-items: center;
          justify-content: center;
          font-weight: 600;
          font-size: 12px;
        }

        .user-name {
          font-size: 14px;
          font-weight: 500;
          color: var(--text-primary);
          white-space: nowrap;
          max-width: 100px;
          overflow: hidden;
          text-overflow: ellipsis;
        }

        .user-menu-overlay {
          position: fixed;
          top: 0;
          left: 0;
          right: 0;
          bottom: 0;
          z-index: 10000;
        }

        .user-menu {
          position: absolute;
          top: calc(100% + 8px);
          right: 0;
          background: var(--bg-primary);
          border: 1px solid var(--border-medium);
          border-radius: 12px;
          box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
          min-width: 200px;
          overflow: hidden;
        }

        .user-info {
          padding: 16px;
          border-bottom: 1px solid var(--border-light);
        }

        .user-avatar-large {
          width: 48px;
          height: 48px;
          background: var(--primary);
          color: white;
          border-radius: 50%;
          display: flex;
          align-items: center;
          justify-content: center;
          font-weight: 600;
          font-size: 18px;
          margin-bottom: 12px;
        }

        .user-details {
          display: flex;
          flex-direction: column;
          gap: 2px;
        }

        .user-name-display {
          font-weight: 600;
          font-size: 14px;
          color: var(--text-primary);
        }

        .user-email {
          font-size: 12px;
          color: var(--text-secondary);
        }

        .user-menu-items {
          padding: 8px;
        }

        .menu-item {
          display: flex;
          align-items: center;
          gap: 12px;
          width: 100%;
          padding: 12px 16px;
          background: none;
          border: none;
          border-radius: 8px;
          font-size: 14px;
          font-weight: 500;
          color: var(--text-primary);
          cursor: pointer;
          transition: all 0.2s;
          text-align: left;
        }

        .menu-item:hover {
          background: var(--bg-secondary);
        }

        .menu-item.logout {
          color: var(--destructive);
        }

        .menu-item.logout:hover {
          background: var(--destructive-bg);
        }

        .menu-item svg {
          flex-shrink: 0;
        }

        @media (max-width: 768px) {
          .user-name {
            display: none;
          }
          
          .user-menu {
            right: -40px;
          }
        }
      `}</style>
    </>
  )
}