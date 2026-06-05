"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.webVitalsPlugin = void 0;
var tslib_1 = require("tslib");
/* eslint-disable no-restricted-globals */
var analytics_core_1 = require("@amplitude/analytics-core");
var constants_1 = require("./constants");
var web_vitals_1 = require("web-vitals");
function getMetricStartTime(metric) {
    var _a;
    /* istanbul ignore next */
    var startTime = ((_a = metric.entries[0]) === null || _a === void 0 ? void 0 : _a.startTime) || 0;
    return performance.timeOrigin + startTime;
}
function processMetric(metric) {
    return {
        value: metric.value,
        rating: metric.rating,
        delta: metric.delta,
        navigationType: metric.navigationType,
        id: metric.id,
        timestamp: Math.floor(getMetricStartTime(metric)),
        navigationStart: Math.floor(performance.timeOrigin),
    };
}
var webVitalsPlugin = function () {
    var visibilityListener = null;
    var globalScope = (0, analytics_core_1.getGlobalScope)();
    var doc = globalScope === null || globalScope === void 0 ? void 0 : globalScope.document;
    var location = globalScope === null || globalScope === void 0 ? void 0 : globalScope.location;
    var setup = function (config, amplitude) { return tslib_1.__awaiter(void 0, void 0, void 0, function () {
        var locationHref, webVitalsPayload;
        return tslib_1.__generator(this, function (_a) {
            if (doc === undefined) {
                return [2 /*return*/];
            }
            locationHref = (0, analytics_core_1.getDecodeURI)(/* istanbul ignore next */ (location === null || location === void 0 ? void 0 : location.href) || '', config.loggerProvider);
            webVitalsPayload = {
                '[Amplitude] Page Domain': /* istanbul ignore next */ (location === null || location === void 0 ? void 0 : location.hostname) || '',
                '[Amplitude] Page Location': locationHref,
                '[Amplitude] Page Path': (0, analytics_core_1.getDecodeURI)(/* istanbul ignore next */ (location === null || location === void 0 ? void 0 : location.pathname) || '', config.loggerProvider),
                '[Amplitude] Page Title': /* istanbul ignore next */ (typeof document !== 'undefined' && document.title) || '',
                '[Amplitude] Page URL': (0, analytics_core_1.getDecodeURI)(locationHref.split('?')[0], config.loggerProvider),
            };
            (0, web_vitals_1.onLCP)(function (metric) {
                webVitalsPayload['[Amplitude] LCP'] = processMetric(metric);
            });
            (0, web_vitals_1.onFCP)(function (metric) {
                webVitalsPayload['[Amplitude] FCP'] = processMetric(metric);
            });
            (0, web_vitals_1.onINP)(function (metric) {
                webVitalsPayload['[Amplitude] INP'] = processMetric(metric);
            });
            (0, web_vitals_1.onCLS)(function (metric) {
                webVitalsPayload['[Amplitude] CLS'] = processMetric(metric);
            });
            (0, web_vitals_1.onTTFB)(function (metric) {
                webVitalsPayload['[Amplitude] TTFB'] = processMetric(metric);
            });
            visibilityListener = function () {
                if (doc.visibilityState === 'hidden' && visibilityListener) {
                    amplitude.track(constants_1.WEB_VITALS_EVENT_NAME, webVitalsPayload);
                    doc.removeEventListener('visibilitychange', visibilityListener);
                    visibilityListener = null;
                }
            };
            doc.addEventListener('visibilitychange', visibilityListener);
            return [2 /*return*/];
        });
    }); };
    var execute = function (event) { return tslib_1.__awaiter(void 0, void 0, void 0, function () {
        return tslib_1.__generator(this, function (_a) {
            return [2 /*return*/, event];
        });
    }); };
    var teardown = function () { return tslib_1.__awaiter(void 0, void 0, void 0, function () {
        return tslib_1.__generator(this, function (_a) {
            if (visibilityListener) {
                /* istanbul ignore next */
                doc === null || doc === void 0 ? void 0 : doc.removeEventListener('visibilitychange', visibilityListener);
            }
            return [2 /*return*/];
        });
    }); };
    return {
        name: constants_1.PLUGIN_NAME,
        type: 'enrichment',
        setup: setup,
        execute: execute,
        teardown: teardown,
    };
};
exports.webVitalsPlugin = webVitalsPlugin;
//# sourceMappingURL=web-vitals-plugin.js.map