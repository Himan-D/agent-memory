"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.trackExposure = void 0;
/* eslint-disable no-restricted-globals */
var analytics_core_1 = require("@amplitude/analytics-core");
function trackExposure(_a) {
    var allObservables = _a.allObservables, onExposure = _a.onExposure, dataExtractor = _a.dataExtractor, _b = _a.exposureDuration, exposureDuration = _b === void 0 ? analytics_core_1.DEFAULT_EXPOSURE_DURATION : _b;
    // Track which elements have been marked as exposed (per-element state)
    var exposureMap = new Map();
    // Track active timers for elements that are currently visible but not yet exposed
    var exposureTimerMap = new Map();
    var exposureObservable = allObservables.exposureObservable;
    var exposureSubscription = exposureObservable.subscribe(function (event) {
        var entry = event;
        var element = entry.target;
        if (entry.isIntersecting) {
            // Element became visible - start exposure timer if not already exposed
            if (!exposureMap.get(element)) {
                var timer = setTimeout(function () {
                    // Element has been visible for exposureDuration - mark as exposed
                    exposureMap.set(element, true);
                    // Record the CSS selector path in the shared exposure state
                    var elementPath = dataExtractor.getElementPath(element);
                    onExposure(elementPath);
                    // Clear the timer reference
                    exposureTimerMap.set(element, null);
                }, exposureDuration);
                exposureTimerMap.set(element, timer);
            }
        }
        else if (!entry.isIntersecting && entry.intersectionRatio < 1.0) {
            // Element left viewport - cancel exposure timer if one exists
            var timer = exposureTimerMap.get(element);
            if (timer) {
                clearTimeout(timer);
                exposureTimerMap.set(element, null);
            }
        }
    });
    return {
        unsubscribe: function () {
            exposureSubscription.unsubscribe();
        },
        reset: function () {
            exposureTimerMap.forEach(function (timer) {
                if (timer) {
                    clearTimeout(timer);
                }
            });
            exposureTimerMap.clear();
            exposureMap.clear();
        },
    };
}
exports.trackExposure = trackExposure;
//# sourceMappingURL=track-exposure.js.map