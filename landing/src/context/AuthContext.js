import { createContext, useContext, useState, useEffect } from 'react'
import React from 'react'
import { analytics } from '../utils/analytics'

const API_BASE = (typeof window !== 'undefined' && window.__HYS_API_URL) || 'http://localhost:8080'

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null)
  const [token, setToken] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(function() {
    var storedUser = localStorage.getItem('hystersis_user')
    var storedToken = localStorage.getItem('hystersis_token')
    if (storedUser && storedToken) {
      try {
        var parsed = JSON.parse(storedUser)
        if (parsed && parsed.email === 'demo@hystersis.ai') {
          localStorage.removeItem('hystersis_user')
          localStorage.removeItem('hystersis_token')
        } else {
          setUser(parsed)
          setToken(storedToken)
        }
      } catch (e) {
        localStorage.removeItem('hystersis_user')
        localStorage.removeItem('hystersis_token')
      }
    }
    setLoading(false)
  }, [])

    function isDemoUser(email) {
    return email === 'demo@hystersis.ai'
  }

  function login(email, password) {
    analytics.loginAttempted('password')
    return fetchWithRetry(API_BASE + '/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: email, password: password })
    }).then(function(data) {
      if (data.success && data.user && data.token) {
        setUser(data.user)
        setToken(data.token)
        if (!isDemoUser(data.user.email)) {
          localStorage.setItem('hystersis_user', JSON.stringify(data.user))
          localStorage.setItem('hystersis_token', data.token)
        }
        analytics.identify(data.user.id, { email: data.user.email, name: data.user.name })
        analytics.loginSuccess('password')
      }
      return data
    })
  }

  function register(email, password, name) {
    analytics.signupStarted('password')
    return fetchWithRetry(API_BASE + '/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: email, password: password, name: name })
    }).then(function(data) {
      if (data.success && data.user && data.token) {
        setUser(data.user)
        setToken(data.token)
        localStorage.setItem('hystersis_user', JSON.stringify(data.user))
        localStorage.setItem('hystersis_token', data.token)
        analytics.identify(data.user.id, { email: data.user.email, name: data.user.name })
        analytics.signupSuccess()
      }
      return data
    })
  }

  function logout() {
    var currentUser = user
    setUser(null)
    setToken(null)
    localStorage.removeItem('hystersis_user')
    localStorage.removeItem('hystersis_token')
    if (currentUser) {
      analytics.amplitudeReset()
    }
    fetchWithRetry(API_BASE + '/auth/logout', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    }).catch(function() {})
  }

  function refreshToken() {
    return fetchWithRetry(API_BASE + '/auth/refresh', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer ' + token
      }
    }).then(function(data) {
      if (data.success && data.user && data.token) {
        setUser(data.user)
        setToken(data.token)
        if (!isDemoUser(data.user.email)) {
          localStorage.setItem('hystersis_user', JSON.stringify(data.user))
          localStorage.setItem('hystersis_token', data.token)
        }
      }
      return data
    })
  }

  var value = {
    user: user,
    token: token,
    login: login,
    register: register,
    logout: logout,
    refreshToken: refreshToken,
    loading: loading
  }

  return React.createElement(AuthContext.Provider, { value: value }, children)
}

function fetchWithRetry(url, options, retries) {
  if (retries === undefined) retries = 1
  return fetch(url, options).then(function(response) {
    if (!response.ok && retries > 0) {
      return new Promise(function(resolve) {
        setTimeout(function() {
          resolve(fetchWithRetry(url, options, retries - 1))
        }, 500)
      })
    }
    return response.json().then(function(data) {
      if (!response.ok) {
        var err = new Error(data.error || 'Request failed')
        err.status = response.status
        throw err
      }
      return data
    })
  }).catch(function(err) {
    if (err.status === 401) {
      return { success: false, error: 'Invalid email or password' }
    }
    if (retries > 0) {
      return new Promise(function(resolve) {
        setTimeout(function() {
          resolve(fetchWithRetry(url, options, retries - 1))
        }, 1000)
      })
    }
    return { success: false, error: err.message || 'Network error. Please try again.' }
  })
}

export function useAuth() {
  return useContext(AuthContext)
}