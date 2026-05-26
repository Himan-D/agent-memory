"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.BaseTransport = void 0;
var status_1 = require("../types/status");
var status_code_1 = require("../utils/status-code");
var BaseTransport = /** @class */ (function () {
    function BaseTransport() {
    }
    BaseTransport.prototype.send = function (_serverUrl, _payload, _enableRequestBodyCompression) {
        return Promise.resolve(null);
    };
    BaseTransport.prototype.buildResponse = function (responseJSON) {
        var _a, _b, _c, _d, _e, _f, _g, _h, _j, _k, _l, _m, _o, _p, _q, _r, _s, _t, _u, _v, _w, _x;
        if (typeof responseJSON !== 'object') {
            return null;
        }
        var statusCode = responseJSON.code || 0;
        var status = this.buildStatus(statusCode);
        switch (status) {
            case status_1.Status.Success:
                return {
                    status: status,
                    statusCode: statusCode,
                    body: {
                        eventsIngested: (_a = responseJSON.events_ingested) !== null && _a !== void 0 ? _a : 0,
                        payloadSizeBytes: (_b = responseJSON.payload_size_bytes) !== null && _b !== void 0 ? _b : 0,
                        serverUploadTime: (_c = responseJSON.server_upload_time) !== null && _c !== void 0 ? _c : 0,
                    },
                };
            case status_1.Status.Invalid:
                return {
                    status: status,
                    statusCode: statusCode,
                    body: {
                        error: (_d = responseJSON.error) !== null && _d !== void 0 ? _d : '',
                        missingField: (_e = responseJSON.missing_field) !== null && _e !== void 0 ? _e : '',
                        eventsWithInvalidFields: (_f = responseJSON.events_with_invalid_fields) !== null && _f !== void 0 ? _f : {},
                        eventsWithMissingFields: (_g = responseJSON.events_with_missing_fields) !== null && _g !== void 0 ? _g : {},
                        eventsWithInvalidIdLengths: (_h = responseJSON.events_with_invalid_id_lengths) !== null && _h !== void 0 ? _h : {},
                        epsThreshold: (_j = responseJSON.eps_threshold) !== null && _j !== void 0 ? _j : 0,
                        exceededDailyQuotaDevices: (_k = responseJSON.exceeded_daily_quota_devices) !== null && _k !== void 0 ? _k : {},
                        silencedDevices: (_l = responseJSON.silenced_devices) !== null && _l !== void 0 ? _l : [],
                        silencedEvents: (_m = responseJSON.silenced_events) !== null && _m !== void 0 ? _m : [],
                        throttledDevices: (_o = responseJSON.throttled_devices) !== null && _o !== void 0 ? _o : {},
                        throttledEvents: (_p = responseJSON.throttled_events) !== null && _p !== void 0 ? _p : [],
                    },
                };
            case status_1.Status.PayloadTooLarge:
                return {
                    status: status,
                    statusCode: statusCode,
                    body: {
                        error: (_q = responseJSON.error) !== null && _q !== void 0 ? _q : '',
                    },
                };
            case status_1.Status.RateLimit:
                return {
                    status: status,
                    statusCode: statusCode,
                    body: {
                        error: (_r = responseJSON.error) !== null && _r !== void 0 ? _r : '',
                        epsThreshold: (_s = responseJSON.eps_threshold) !== null && _s !== void 0 ? _s : 0,
                        throttledDevices: (_t = responseJSON.throttled_devices) !== null && _t !== void 0 ? _t : {},
                        throttledUsers: (_u = responseJSON.throttled_users) !== null && _u !== void 0 ? _u : {},
                        exceededDailyQuotaDevices: (_v = responseJSON.exceeded_daily_quota_devices) !== null && _v !== void 0 ? _v : {},
                        exceededDailyQuotaUsers: (_w = responseJSON.exceeded_daily_quota_users) !== null && _w !== void 0 ? _w : {},
                        throttledEvents: (_x = responseJSON.throttled_events) !== null && _x !== void 0 ? _x : [],
                    },
                };
            case status_1.Status.Timeout:
            default:
                return {
                    status: status,
                    statusCode: statusCode,
                };
        }
    };
    BaseTransport.prototype.buildStatus = function (code) {
        if ((0, status_code_1.isSuccessStatusCode)(code)) {
            return status_1.Status.Success;
        }
        if (code === 429) {
            return status_1.Status.RateLimit;
        }
        if (code === 413) {
            return status_1.Status.PayloadTooLarge;
        }
        if (code === 408) {
            return status_1.Status.Timeout;
        }
        if (code >= 400 && code < 500) {
            return status_1.Status.Invalid;
        }
        if (code >= 500) {
            return status_1.Status.Failed;
        }
        return status_1.Status.Unknown;
    };
    return BaseTransport;
}());
exports.BaseTransport = BaseTransport;
//# sourceMappingURL=base.js.map