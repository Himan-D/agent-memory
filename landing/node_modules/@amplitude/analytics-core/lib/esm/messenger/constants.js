// Shared origin constants for Amplitude cross-window communication
export var AMPLITUDE_ORIGIN = 'https://app.amplitude.com';
export var AMPLITUDE_ORIGIN_EU = 'https://app.eu.amplitude.com';
export var AMPLITUDE_ORIGIN_STAGING = 'https://apps.stag2.amplitude.com';
export var AMPLITUDE_ORIGINS_MAP = {
    US: AMPLITUDE_ORIGIN,
    EU: AMPLITUDE_ORIGIN_EU,
    STAGING: AMPLITUDE_ORIGIN_STAGING,
};
// Background capture script URL (shared between autocapture and session-replay)
export var AMPLITUDE_BACKGROUND_CAPTURE_SCRIPT_URL = 'https://cdn.amplitude.com/libs/background-capture-1.0.0-alpha.3.js.gz';
//# sourceMappingURL=constants.js.map