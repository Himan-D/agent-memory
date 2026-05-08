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