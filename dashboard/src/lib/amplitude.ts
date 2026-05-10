"use client";

import * as amplitude from "@amplitude/analytics-browser";

const AMPLITUDE_API_KEY = process.env.NEXT_PUBLIC_AMPLITUDE_API_KEY || "";

let initialized = false;

export function initAmplitude() {
  if (!AMPLITUDE_API_KEY || initialized) return;
  amplitude.init(AMPLITUDE_API_KEY, {
    defaultTracking: {
      sessions: true,
      pageViews: true,
      formInteractions: true,
      fileDownloads: true,
    },
  });
  initialized = true;
}

export function trackEvent(eventName: string, eventProperties?: Record<string, string | number | boolean>) {
  if (!AMPLITUDE_API_KEY) return;
  amplitude.track(eventName, eventProperties);
}

// Auth tracking events
export function trackSignInAttempt(email: string) {
  trackEvent("sign_in_attempt", { email });
}

export function trackSignInSuccess(email: string) {
  trackEvent("sign_in_success", { email });
  amplitude.setUserId(email);
}

export function trackSignInError(email: string, error: string) {
  trackEvent("sign_in_error", { email, error });
}

export function trackSignUpAttempt(email: string) {
  trackEvent("sign_up_attempt", { email });
}

export function trackSignUpSuccess(email: string) {
  trackEvent("sign_up_success", { email });
}

export function trackSignUpError(email: string, error: string) {
  trackEvent("sign_up_error", { email, error });
}

export function trackLogout() {
  trackEvent("user_logout", {});
  amplitude.reset();
}

export function trackPageView(pageName: string) {
  trackEvent("page_view", { page_name: pageName });
}

export function trackFeatureUsage(feature: string) {
  trackEvent("feature_used", { feature });
}

export function setUserId(userId: string) {
  if (!AMPLITUDE_API_KEY) return;
  amplitude.setUserId(userId);
}

export function setUserProperties(properties: Record<string, string | number | boolean>) {
  if (!AMPLITUDE_API_KEY) return;
  const identify = new amplitude.Identify();
  Object.entries(properties).forEach(([key, value]) => {
    if (typeof value === "string") identify.set(key, value);
    else if (typeof value === "number") identify.set(key, value);
    else if (typeof value === "boolean") identify.set(key, value);
  });
  amplitude.identify(identify);
}

export function resetAmplitude() {
  if (!AMPLITUDE_API_KEY) return;
  amplitude.reset();
}

export function identifyUser(userId: string, traits: Record<string, string | number | boolean> = {}) {
  if (!AMPLITUDE_API_KEY) return;
  amplitude.setUserId(userId);
  const identify = new amplitude.Identify();
  Object.entries(traits).forEach(([key, value]) => {
    if (typeof value === "string") identify.set(key, value);
    else if (typeof value === "number") identify.set(key, value);
    else if (typeof value === "boolean") identify.set(key, value);
  });
  amplitude.identify(identify);
}