import { posthog } from 'posthog-js'

export const analytics = {
  track: (event, properties = {}) => {
    posthog.capture(event, properties)
  },

  identify: (userId, traits = {}) => {
    posthog.identify(userId, traits)
  },

  pageView: (name, properties = {}) => {
    posthog.capture('$pageview', { name, ...properties })
  },

  featureUsed: (feature, properties = {}) => {
    posthog.capture('feature_used', { feature, ...properties })
  },

  ctaClicked: (ctaName, location, properties = {}) => {
    posthog.capture('cta_clicked', { cta_name: ctaName, location, ...properties })
  },

  pricingViewed: (plan, properties = {}) => {
    posthog.capture('pricing_viewed', { plan, ...properties })
  },

  signupStarted: (method, properties = {}) => {
    posthog.capture('signup_started', { method, ...properties })
  },

  sdkInstalled: (sdk, language, properties = {}) => {
    posthog.capture('sdk_installed', { sdk, language, ...properties })
  },

  docViewed: (docName, properties = {}) => {
    posthog.capture('doc_viewed', { doc_name: docName, ...properties })
  },

  sessionStarted: (properties = {}) => {
    posthog.capture('session_started', properties)
  },
}

export default analytics
