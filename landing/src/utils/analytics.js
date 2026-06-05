import { posthog } from 'posthog-js'
import * as amplitude from '@amplitude/analytics-browser'

const AMPLITUDE_API_KEY = import.meta.env.VITE_AMPLITUDE_API_KEY || ''

// Initialize Amplitude
if (AMPLITUDE_API_KEY) {
  amplitude.init(AMPLITUDE_API_KEY, {
    defaultTracking: {
      sessions: true,
      pageViews: true,
      formInteractions: true,
      fileDownloads: true,
    },
  })
}

export const analytics = {
  // PostHog methods
  track: (event, properties = {}) => {
    posthog.capture(event, properties)
    if (AMPLITUDE_API_KEY) {
      amplitude.track(event, properties)
    }
  },

  identify: (userId, traits = {}) => {
    posthog.identify(userId, traits)
    if (AMPLITUDE_API_KEY) {
      amplitude.setUserId(userId)
      const identify = new amplitude.Identify()
      Object.entries(traits).forEach(([key, value]) => {
        if (typeof value === 'string') identify.set(key, value)
        else if (typeof value === 'number') identify.set(key, value)
        else if (typeof value === 'boolean') identify.set(key, value)
      })
      amplitude.identify(identify)
    }
  },

  pageView: (name, properties = {}) => {
    posthog.capture('$pageview', { name, ...properties })
    if (AMPLITUDE_API_KEY) {
      amplitude.track('Page Viewed', { page_name: name, ...properties })
    }
  },

  featureUsed: (feature, properties = {}) => {
    posthog.capture('feature_used', { feature, ...properties })
    if (AMPLITUDE_API_KEY) {
      amplitude.track('Feature Used', { feature, ...properties })
    }
  },

  ctaClicked: (ctaName, location, properties = {}) => {
    posthog.capture('cta_clicked', { cta_name: ctaName, location, ...properties })
    if (AMPLITUDE_API_KEY) {
      amplitude.track('CTA Clicked', { cta_name: ctaName, location, ...properties })
    }
  },

  pricingViewed: (plan, properties = {}) => {
    posthog.capture('pricing_viewed', { plan, ...properties })
    if (AMPLITUDE_API_KEY) {
      amplitude.track('Pricing Viewed', { plan, ...properties })
    }
  },

  signupStarted: (method, properties = {}) => {
    posthog.capture('signup_started', { method, ...properties })
    if (AMPLITUDE_API_KEY) {
      amplitude.track('Signup Started', { method, ...properties })
    }
  },

  loginAttempted: (method, properties = {}) => {
    posthog.capture('login_attempted', { method, ...properties })
    if (AMPLITUDE_API_KEY) {
      amplitude.track('Login Attempted', { method, ...properties })
    }
  },

  loginSuccess: (method, properties = {}) => {
    posthog.capture('login_success', { method, ...properties })
    if (AMPLITUDE_API_KEY) {
      amplitude.track('Login Success', { method, ...properties })
    }
  },

  signupSuccess: (properties = {}) => {
    posthog.capture('signup_success', properties)
    if (AMPLITUDE_API_KEY) {
      amplitude.track('Signup Success', properties)
    }
  },

  sdkInstalled: (sdk, language, properties = {}) => {
    posthog.capture('sdk_installed', { sdk, language, ...properties })
    if (AMPLITUDE_API_KEY) {
      amplitude.track('SDK Installed', { sdk, language, ...properties })
    }
  },

  docViewed: (docName, properties = {}) => {
    posthog.capture('doc_viewed', { doc_name: docName, ...properties })
    if (AMPLITUDE_API_KEY) {
      amplitude.track('Document Viewed', { doc_name: docName, ...properties })
    }
  },

  sessionStarted: (properties = {}) => {
    posthog.capture('session_started', properties)
    if (AMPLITUDE_API_KEY) {
      amplitude.track('Session Started', properties)
    }
  },

  // Amplitude-specific methods
  amplitudeTrack: (eventName, properties = {}) => {
    if (AMPLITUDE_API_KEY) {
      amplitude.track(eventName, properties)
    }
  },

  amplitudeSetUserProperties: (properties = {}) => {
    if (AMPLITUDE_API_KEY) {
      const identify = new amplitude.Identify()
      Object.entries(properties).forEach(([key, value]) => {
        if (typeof value === 'string') identify.set(key, value)
        else if (typeof value === 'number') identify.set(key, value)
        else if (typeof value === 'boolean') identify.set(key, value)
      })
      amplitude.identify(identify)
    }
  },

  amplitudeSetUserId: (userId) => {
    if (AMPLITUDE_API_KEY) {
      amplitude.setUserId(userId)
    }
  },

  amplitudeReset: () => {
    if (AMPLITUDE_API_KEY) {
      amplitude.reset()
    }
  },
}

export const initAnalytics = () => {
  // PostHog is initialized in main.jsx
  // Amplitude is initialized here if key is available
  if (AMPLITUDE_API_KEY) {
    console.log('Amplitude analytics initialized')
  }
}

export default analytics