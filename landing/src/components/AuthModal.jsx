import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useAuth } from '../context/AuthContext'

export function AuthModal() {
  const [isOpen, setIsOpen] = useState(false)
  const [mode, setMode] = useState('signin') // 'signin' or 'signup'
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  
  const { login, register } = useAuth()

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      var result = mode === 'signin' 
        ? await login(email, password)
        : await register(email, password, name)

      if (result.success) {
        setIsOpen(false)
        setEmail('')
        setPassword('')
        setName('')
      } else {
        setError(result.error || 'Authentication failed')
      }
    } catch (err) {
      setError('Network error. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
      <button 
        onClick={() => setIsOpen(true)}
        className="nav-cta"
      >
        Sign In
      </button>

      <AnimatePresence>
        {isOpen && (
          <div className="auth-modal-overlay" onClick={() => setIsOpen(false)}>
            <motion.div 
              className="auth-modal"
              onClick={(e) => e.stopPropagation()}
              initial={{ opacity: 0, scale: 0.9 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.9 }}
              transition={{ duration: 0.2 }}
            >
              <div className="auth-header">
                <div className="auth-logo">
                  <div className="logo-icon">
                    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/>
                    </svg>
                  </div>
                  <h1>Hystersis</h1>
                </div>
                <button 
                  onClick={() => setIsOpen(false)}
                  className="close-btn"
                  aria-label="Close"
                >
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <line x1="18" y1="6" x2="6" y2="18"/>
                    <line x1="6" y1="6" x2="18" y2="18"/>
                  </svg>
                </button>
              </div>

              <div className="auth-tabs">
                <button 
                  className={`tab ${mode === 'signin' ? 'active' : ''}`}
                  onClick={() => {
                    setMode('signin')
                    setError('')
                  }}
                >
                  Sign In
                </button>
                <button 
                  className={`tab ${mode === 'signup' ? 'active' : ''}`}
                  onClick={() => {
                    setMode('signup')
                    setError('')
                  }}
                >
                  Sign Up
                </button>
              </div>

              <form onSubmit={handleSubmit} className="auth-form">
                {error && (
                  <div className="error-message">
                    {error}
                  </div>
                )}

                {mode === 'signup' && (
                  <div className="form-group">
                    <label htmlFor="name">Name</label>
                    <input
                      id="name"
                      type="text"
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      placeholder="John Doe"
                      required
                    />
                  </div>
                )}

                <div className="form-group">
                  <label htmlFor="email">Email</label>
                  <input
                    id="email"
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    placeholder="you@example.com"
                    required
                  />
                </div>

                <div className="form-group">
                  <label htmlFor="password">Password</label>
                  <input
                    id="password"
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="••••••••"
                    required
                  />
                </div>

                <button 
                  type="submit" 
                  className="auth-submit"
                  disabled={loading}
                >
                  {loading ? 'Loading...' : mode === 'signin' ? 'Sign In' : 'Sign Up'}
                </button>
              </form>

              {mode === 'signin' && (
                <div className="demo-credentials">
                  <p className="demo-label">Demo Credentials</p>
                  <div className="demo-info">
                    <code>demo@hystersis.ai</code>
                    <span>/</span>
                    <code>demo123</code>
                  </div>
                </div>
              )}
            </motion.div>
          </div>
        )}
      </AnimatePresence>

      <style>{`
        .auth-modal-overlay {
          position: fixed;
          top: 0;
          left: 0;
          right: 0;
          bottom: 0;
          background: rgba(0, 0, 0, 0.5);
          display: flex;
          align-items: center;
          justify-content: center;
          z-index: 10000;
          padding: 20px;
        }

        .auth-modal {
          background: var(--bg-primary);
          border: 1px solid var(--border-medium);
          border-radius: 16px;
          width: 100%;
          max-width: 400px;
          box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
        }

        .auth-header {
          padding: 24px 24px 16px;
          border-bottom: 1px solid var(--border-light);
          display: flex;
          justify-content: space-between;
          align-items: center;
        }

        .auth-logo {
          display: flex;
          align-items: center;
          gap: 12px;
        }

        .logo-icon {
          width: 40px;
          height: 40px;
          background: var(--primary);
          border-radius: 10px;
          display: flex;
          align-items: center;
          justify-content: center;
          color: white;
        }

        .auth-logo h1 {
          font-size: 20px;
          font-weight: 700;
          color: var(--text-primary);
          margin: 0;
        }

        .close-btn {
          background: none;
          border: none;
          color: var(--text-secondary);
          cursor: pointer;
          padding: 8px;
          border-radius: 6px;
          transition: all 0.2s;
        }

        .close-btn:hover {
          background: var(--bg-secondary);
          color: var(--text-primary);
        }

        .auth-tabs {
          display: flex;
          background: var(--bg-secondary);
          border-radius: 8px;
          margin: 20px 24px;
          padding: 4px;
        }

        .tab {
          flex: 1;
          padding: 12px;
          background: none;
          border: none;
          border-radius: 6px;
          font-weight: 500;
          color: var(--text-secondary);
          cursor: pointer;
          transition: all 0.2s;
        }

        .tab.active {
          background: var(--bg-primary);
          color: var(--text-primary);
          box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
        }

        .auth-form {
          padding: 24px;
        }

        .form-group {
          margin-bottom: 20px;
        }

        .form-group label {
          display: block;
          margin-bottom: 8px;
          font-weight: 500;
          color: var(--text-primary);
          font-size: 14px;
        }

        .form-group input {
          width: 100%;
          padding: 12px 16px;
          border: 1px solid var(--border-light);
          border-radius: 8px;
          background: var(--bg-primary);
          color: var(--text-primary);
          font-size: 14px;
          transition: all 0.2s;
        }

        .form-group input:focus {
          outline: none;
          border-color: var(--primary);
          box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
        }

        .error-message {
          background: var(--destructive-bg);
          color: var(--destructive);
          padding: 12px 16px;
          border-radius: 8px;
          margin-bottom: 20px;
          font-size: 14px;
          text-align: center;
        }

        .auth-submit {
          width: 100%;
          padding: 12px 24px;
          background: var(--primary);
          color: white;
          border: none;
          border-radius: 8px;
          font-weight: 600;
          font-size: 14px;
          cursor: pointer;
          transition: all 0.2s;
        }

        .auth-submit:hover:not(:disabled) {
          opacity: 0.9;
        }

        .auth-submit:disabled {
          opacity: 0.6;
          cursor: not-allowed;
        }

        .demo-credentials {
          padding: 20px 24px;
          border-top: 1px solid var(--border-light);
          background: var(--bg-secondary);
          border-radius: 0 0 16px 16px;
        }

        .demo-label {
          font-size: 12px;
          font-weight: 500;
          color: var(--text-secondary);
          margin-bottom: 8px;
          text-align: center;
        }

        .demo-info {
          display: flex;
          align-items: center;
          justify-content: center;
          gap: 4px;
          font-size: 12px;
        }

        .demo-info code {
          background: var(--bg-primary);
          padding: 4px 8px;
          border-radius: 4px;
          border: 1px solid var(--border-light);
          font-family: 'Monaco', 'Menlo', monospace;
        }

        @media (max-width: 480px) {
          .auth-modal-overlay {
            padding: 10px;
          }
          
          .auth-modal {
            max-width: 100%;
          }
        }
      `}</style>
    </>
  )
}