import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.jsx'
import './App.css'
import { posthog } from 'posthog-js'

posthog.init(import.meta.env.VITE_POSTHOG_KEY || 'phc_placeholder', {
  api_host: import.meta.env.VITE_POSTHOG_HOST || 'https://app.posthog.com',
  person_profiles: 'identified_only',
  capture_pageview: true,
  capture_pageleave: true,
})

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)