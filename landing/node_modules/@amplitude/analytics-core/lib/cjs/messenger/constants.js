"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.AMPLITUDE_BACKGROUND_CAPTURE_SCRIPT_URL = exports.AMPLITUDE_ORIGINS_MAP = exports.AMPLITUDE_ORIGIN_STAGING = exports.AMPLITUDE_ORIGIN_EU = exports.AMPLITUDE_ORIGIN = void 0;
// Shared origin constants for Amplitude cross-window communication
exports.AMPLITUDE_ORIGIN = 'https://app.amplitude.com';
exports.AMPLITUDE_ORIGIN_EU = 'https://app.eu.amplitude.com';
exports.AMPLITUDE_ORIGIN_STAGING = 'https://apps.stag2.amplitude.com';
exports.AMPLITUDE_ORIGINS_MAP = {
    US: exports.AMPLITUDE_ORIGIN,
    EU: exports.AMPLITUDE_ORIGIN_EU,
    STAGING: exports.AMPLITUDE_ORIGIN_STAGING,
};
// Background capture script URL (shared between autocapture and session-replay)
exports.AMPLITUDE_BACKGROUND_CAPTURE_SCRIPT_URL = 'https://cdn.amplitude.com/libs/background-capture-1.0.0-alpha.3.js.gz';
//# sourceMappingURL=constants.js.map