import { __awaiter, __generator } from "tslib";
/* eslint-disable no-restricted-globals */
import { getGlobalScope, getDecodeURI, } from '@amplitude/analytics-core';
import { PLUGIN_NAME, WEB_VITALS_EVENT_NAME } from './constants';
import { onLCP, onINP, onCLS, onFCP, onTTFB } from 'web-vitals';
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
export var webVitalsPlugin = function () {
    var visibilityListener = null;
    var globalScope = getGlobalScope();
    var doc = globalScope === null || globalScope === void 0 ? void 0 : globalScope.document;
    var location = globalScope === null || globalScope === void 0 ? void 0 : globalScope.location;
    var setup = function (config, amplitude) { return __awaiter(void 0, void 0, void 0, function () {
        var locationHref, webVitalsPayload;
        return __generator(this, function (_a) {
            if (doc === undefined) {
                return [2 /*return*/];
            }
            locationHref = getDecodeURI(/* istanbul ignore next */ (location === null || location === void 0 ? void 0 : location.href) || '', config.loggerProvider);
            webVitalsPayload = {
                '[Amplitude] Page Domain': /* istanbul ignore next */ (location === null || location === void 0 ? void 0 : location.hostname) || '',
                '[Amplitude] Page Location': locationHref,
                '[Amplitude] Page Path': getDecodeURI(/* istanbul ignore next */ (location === null || location === void 0 ? void 0 : location.pathname) || '', config.loggerProvider),
                '[Amplitude] Page Title': /* istanbul ignore next */ (typeof document !== 'undefined' && document.title) || '',
                '[Amplitude] Page URL': getDecodeURI(locationHref.split('?')[0], config.loggerProvider),
            };
            onLCP(function (metric) {
                webVitalsPayload['[Amplitude] LCP'] = processMetric(metric);
            });
            onFCP(function (metric) {
                webVitalsPayload['[Amplitude] FCP'] = processMetric(metric);
            });
            onINP(function (metric) {
                webVitalsPayload['[Amplitude] INP'] = processMetric(metric);
            });
            onCLS(function (metric) {
                webVitalsPayload['[Amplitude] CLS'] = processMetric(metric);
            });
            onTTFB(function (metric) {
                webVitalsPayload['[Amplitude] TTFB'] = processMetric(metric);
            });
            visibilityListener = function () {
                if (doc.visibilityState === 'hidden' && visibilityListener) {
                    amplitude.track(WEB_VITALS_EVENT_NAME, webVitalsPayload);
                    doc.removeEventListener('visibilitychange', visibilityListener);
                    visibilityListener = null;
                }
            };
            doc.addEventListener('visibilitychange', visibilityListener);
            return [2 /*return*/];
        });
    }); };
    var execute = function (event) { return __awaiter(void 0, void 0, void 0, function () {
        return __generator(this, function (_a) {
            return [2 /*return*/, event];
        });
    }); };
    var teardown = function () { return __awaiter(void 0, void 0, void 0, function () {
        return __generator(this, function (_a) {
            if (visibilityListener) {
                /* istanbul ignore next */
                doc === null || doc === void 0 ? void 0 : doc.removeEventListener('visibilitychange', visibilityListener);
            }
            return [2 /*return*/];
        });
    }); };
    return {
        name: PLUGIN_NAME,
        type: 'enrichment',
        setup: setup,
        execute: execute,
        teardown: teardown,
    };
};
//# sourceMappingURL=web-vitals-plugin.js.map