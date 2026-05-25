"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.enableBackgroundCapture = void 0;
var constants_1 = require("./constants");
/**
 * Brand key set on the messenger instance to track whether background capture
 * has been enabled.
 */
var BG_CAPTURE_BRAND = '__AMPLITUDE_BACKGROUND_CAPTURE__';
/**
 * Enable background capture on a messenger instance.
 * Plugins can call this on a shared messenger instance.
 * The first call registers the handlers; subsequent calls are no-ops.
 *
 * @param messenger - The messenger to enable background capture on
 * @param options.scriptUrl - Override the background capture script URL (optional)
 */
function enableBackgroundCapture(messenger, options) {
    var _a;
    // Check the brand on the messenger object itself — works across bundle boundaries
    var branded = messenger;
    if (branded[BG_CAPTURE_BRAND] === true) {
        return;
    }
    branded[BG_CAPTURE_BRAND] = true;
    var scriptUrl = (_a = options === null || options === void 0 ? void 0 : options.scriptUrl) !== null && _a !== void 0 ? _a : constants_1.AMPLITUDE_BACKGROUND_CAPTURE_SCRIPT_URL;
    var backgroundCaptureInstance = null;
    var onBackgroundCapture = function (type, backgroundCaptureData) {
        var _a, _b;
        if (type === 'background-capture-complete') {
            (_b = (_a = messenger.logger) === null || _a === void 0 ? void 0 : _a.debug) === null || _b === void 0 ? void 0 : _b.call(_a, 'Background capture complete');
            messenger.notify({ action: 'background-capture-complete', data: backgroundCaptureData });
        }
    };
    messenger.registerActionHandler('initialize-background-capture', function () {
        var _a, _b;
        (_b = (_a = messenger.logger) === null || _a === void 0 ? void 0 : _a.debug) === null || _b === void 0 ? void 0 : _b.call(_a, 'Initializing background capture (external script)');
        var resolvedUrl = new URL(scriptUrl, messenger.endpoint).toString();
        messenger
            .loadScriptOnce(resolvedUrl)
            .then(function () {
            var _a, _b, _c;
            (_b = (_a = messenger.logger) === null || _a === void 0 ? void 0 : _a.debug) === null || _b === void 0 ? void 0 : _b.call(_a, 'Background capture script loaded (external)');
            // eslint-disable-next-line
            backgroundCaptureInstance = /* istanbul ignore next -- window is always defined in browser */ (_c = window === null || window === void 0 ? void 0 : window.amplitudeBackgroundCapture) === null || _c === void 0 ? void 0 : _c.call(window, {
                messenger: messenger,
                onBackgroundCapture: onBackgroundCapture,
            });
            messenger.notify({ action: 'background-capture-loaded' });
        })
            .catch(function () {
            var _a;
            (_a = messenger.logger) === null || _a === void 0 ? void 0 : _a.warn('Failed to initialize background capture');
        });
    });
    messenger.registerActionHandler('close-background-capture', function () {
        var _a;
        // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-call
        (_a = backgroundCaptureInstance === null || backgroundCaptureInstance === void 0 ? void 0 : backgroundCaptureInstance.close) === null || _a === void 0 ? void 0 : _a.call(backgroundCaptureInstance);
        backgroundCaptureInstance = null;
    });
}
exports.enableBackgroundCapture = enableBackgroundCapture;
//# sourceMappingURL=background-capture.js.map