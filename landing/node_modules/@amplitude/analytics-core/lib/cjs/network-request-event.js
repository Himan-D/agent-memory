"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.NetworkRequestEvent = exports.pruneHeaders = exports.PRUNE_STRATEGY = exports.ResponseWrapperXhr = exports.ResponseWrapperFetch = exports.RequestWrapperXhr = exports.RequestWrapperFetch = exports.MAXIMUM_ENTRIES = void 0;
var tslib_1 = require("tslib");
var global_scope_1 = require("./global-scope");
var json_query_1 = require("./utils/json-query");
var constants_1 = require("./types/constants");
var TEXT_READ_TIMEOUT = 500;
exports.MAXIMUM_ENTRIES = 100;
/**
 * This class encapsulates the RequestInit (https://developer.mozilla.org/en-US/docs/Web/API/RequestInit)
 * object so that the consumer can only get access to the headers, method and body size.
 *
 * This is to prevent consumers from directly accessing the Request object
 * and mutating it or running costly operations on it.
 *
 * IMPORTANT:
 *    * Do not make changes to this class without careful consideration
 *      of performance implications, memory usage and potential to mutate the customer's
 *      request.
 *   * NEVER .clone() the RequestInit object. This will 2x's the memory overhead of the request
 *   * NEVER: call .arrayBuffer(), text(), json() or any other method on the body that
 *     consumes the body's stream. This will cause the response to be consumed
 *     meaning the body will be empty when the customer tries to access it.
 *     (ie: if the body is an instanceof https://developer.mozilla.org/en-US/docs/Web/API/ReadableStream
 *      never call any of the methods on it)
 */
var RequestWrapperFetch = /** @class */ (function () {
    function RequestWrapperFetch(request) {
        this.request = request;
    }
    RequestWrapperFetch.prototype.headers = function (allow) {
        var e_1, _a;
        if (allow === void 0) { allow = []; }
        var headersUnsafe = this.request.headers;
        // copy the headers into a new object
        var headersSafeCopy = {};
        if (Array.isArray(headersUnsafe)) {
            headersUnsafe.forEach(function (_a) {
                var _b = tslib_1.__read(_a, 2), headerName = _b[0], headerValue = _b[1];
                headersSafeCopy[headerName] = headerValue;
            });
        }
        else if (headersUnsafe instanceof Headers) {
            headersUnsafe.forEach(function (value, key) {
                headersSafeCopy[key] = value;
            });
        }
        else if (typeof headersUnsafe === 'object' && headersUnsafe !== null) {
            try {
                for (var _b = tslib_1.__values(Object.entries(headersUnsafe)), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var _d = tslib_1.__read(_c.value, 2), key = _d[0], value = _d[1];
                    headersSafeCopy[key] = value;
                }
            }
            catch (e_1_1) { e_1 = { error: e_1_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_1) throw e_1.error; }
            }
        }
        return (0, exports.pruneHeaders)(headersSafeCopy, { allow: allow });
    };
    Object.defineProperty(RequestWrapperFetch.prototype, "bodySize", {
        get: function () {
            if (typeof this._bodySize === 'number')
                return this._bodySize;
            var global = (0, global_scope_1.getGlobalScope)();
            /* istanbul ignore if */
            if (!(global === null || global === void 0 ? void 0 : global.TextEncoder)) {
                return;
            }
            var body = this.request.body;
            this._bodySize = getBodySize(body, exports.MAXIMUM_ENTRIES);
            return this._bodySize;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(RequestWrapperFetch.prototype, "method", {
        get: function () {
            return this.request.method;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(RequestWrapperFetch.prototype, "body", {
        get: function () {
            if (typeof this.request.body === 'string') {
                return this.request.body;
            }
            return null;
        },
        enumerable: false,
        configurable: true
    });
    RequestWrapperFetch.prototype.json = function (allow, exclude) {
        if (allow === void 0) { allow = []; }
        if (exclude === void 0) { exclude = []; }
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var text;
            return tslib_1.__generator(this, function (_a) {
                if (allow.length === 0) {
                    return [2 /*return*/, null];
                }
                text = this.body;
                return [2 /*return*/, safeParseAndPruneBody(text, allow, exclude)];
            });
        });
    };
    return RequestWrapperFetch;
}());
exports.RequestWrapperFetch = RequestWrapperFetch;
var RequestWrapperXhr = /** @class */ (function () {
    function RequestWrapperXhr(bodyRaw, requestHeaders) {
        this.bodyRaw = bodyRaw;
        this.requestHeaders = requestHeaders;
    }
    RequestWrapperXhr.prototype.headers = function (allow) {
        if (allow === void 0) { allow = []; }
        return (0, exports.pruneHeaders)(this.requestHeaders, { allow: allow });
    };
    Object.defineProperty(RequestWrapperXhr.prototype, "bodySize", {
        get: function () {
            return getBodySize(this.bodyRaw, exports.MAXIMUM_ENTRIES);
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(RequestWrapperXhr.prototype, "body", {
        get: function () {
            if (typeof this.bodyRaw === 'string') {
                return this.bodyRaw;
            }
            return null;
        },
        enumerable: false,
        configurable: true
    });
    RequestWrapperXhr.prototype.json = function (allow, exclude) {
        if (allow === void 0) { allow = []; }
        if (exclude === void 0) { exclude = []; }
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var text;
            return tslib_1.__generator(this, function (_a) {
                if (allow.length === 0) {
                    return [2 /*return*/, null];
                }
                text = this.body;
                return [2 /*return*/, safeParseAndPruneBody(text, allow, exclude)];
            });
        });
    };
    return RequestWrapperXhr;
}());
exports.RequestWrapperXhr = RequestWrapperXhr;
function getBodySize(bodyUnsafe, maxEntries) {
    var e_2, _a;
    var bodySize;
    var global = (0, global_scope_1.getGlobalScope)();
    /* istanbul ignore next */
    var TextEncoder = global === null || global === void 0 ? void 0 : global.TextEncoder;
    /* istanbul ignore next */
    if (!TextEncoder) {
        return;
    }
    var bodySafe;
    if (typeof bodyUnsafe === 'string') {
        bodySafe = bodyUnsafe;
        bodySize = new TextEncoder().encode(bodySafe).length;
    }
    else if (bodyUnsafe instanceof Blob) {
        bodySafe = bodyUnsafe;
        bodySize = bodySafe.size;
    }
    else if (bodyUnsafe instanceof URLSearchParams) {
        bodySafe = bodyUnsafe;
        bodySize = new TextEncoder().encode(bodySafe.toString()).length;
    }
    else if (ArrayBuffer.isView(bodyUnsafe)) {
        bodySafe = bodyUnsafe;
        bodySize = bodySafe.byteLength;
    }
    else if (bodyUnsafe instanceof ArrayBuffer) {
        bodySafe = bodyUnsafe;
        bodySize = bodySafe.byteLength;
    }
    else if (bodyUnsafe instanceof FormData) {
        // Estimating only for text parts; not accurate for files
        var formData = bodyUnsafe;
        var total = 0;
        var count = 0;
        try {
            for (var _b = tslib_1.__values(formData.entries()), _c = _b.next(); !_c.done; _c = _b.next()) {
                var _d = tslib_1.__read(_c.value, 2), key = _d[0], value = _d[1];
                total += key.length;
                if (typeof value === 'string') {
                    total += new TextEncoder().encode(value).length;
                }
                else if (value instanceof Blob) {
                    total += value.size;
                }
                else {
                    // encountered an unknown type
                    // we can't estimate the size of this entry
                    return;
                }
                // terminate if we reach the maximum number of entries
                // to avoid performance issues in case of very large FormData
                if (++count >= maxEntries) {
                    return;
                }
            }
        }
        catch (e_2_1) { e_2 = { error: e_2_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_2) throw e_2.error; }
        }
        bodySize = total;
    }
    else if (bodyUnsafe instanceof ReadableStream) {
        // If bodyUnsafe is an instanceof ReadableStream, we can't determine the size,
        // without consuming it, so we return undefined.
        // Never ever consume ReadableStream! DO NOT DO IT!!!
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        bodySafe = bodyUnsafe;
        return;
    }
    return bodySize;
}
/**
 * This class encapsulates the Fetch API Response object
 * (https://developer.mozilla.org/en-US/docs/Web/API/Response) so that the consumer can
 * only get access to the headers and body size.
 *
 * This is to prevent consumers from directly accessing the Response object
 * and mutating it or running costly operations on it.
 *
 * IMPORTANT:
 *   * Do not make changes to this class without careful consideration
 *     of performance implications, memory usage and potential to mutate the customer's
 *     response.
 *   * Do not .clone() the Response object unless you need to access the body.
 *     Cloning will 2x the memory overhead of the response.
 *   * NEVER consume the body's stream. This will cause the response to be consumed
 *     meaning the body will be empty when the customer tries to access it.
 *     (ie: if the body is an instanceof https://developer.mozilla.org/en-US/docs/Web/API/ReadableStream
 *      never call any of the methods on it)
 */
var ResponseWrapperFetch = /** @class */ (function () {
    function ResponseWrapperFetch(response) {
        this.response = response;
    }
    ResponseWrapperFetch.prototype.headers = function (allow) {
        var _a;
        if (allow === void 0) { allow = []; }
        if (this.response.headers instanceof Headers) {
            var headersSafe = this.response.headers;
            var headersOut_1 = {};
            /* istanbul ignore next */
            (_a = headersSafe === null || headersSafe === void 0 ? void 0 : headersSafe.forEach) === null || _a === void 0 ? void 0 : _a.call(headersSafe, function (value, key) {
                headersOut_1[key] = value;
            });
            return (0, exports.pruneHeaders)(headersOut_1, { allow: allow });
        }
        return;
    };
    Object.defineProperty(ResponseWrapperFetch.prototype, "bodySize", {
        get: function () {
            var _a, _b;
            if (this._bodySize !== undefined)
                return this._bodySize;
            /* istanbul ignore next */
            var contentLength = (_b = (_a = this.response.headers) === null || _a === void 0 ? void 0 : _a.get) === null || _b === void 0 ? void 0 : _b.call(_a, 'content-length');
            var bodySize = contentLength ? parseInt(contentLength, 10) : undefined;
            this._bodySize = bodySize;
            return bodySize;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(ResponseWrapperFetch.prototype, "status", {
        get: function () {
            return this.response.status;
        },
        enumerable: false,
        configurable: true
    });
    ResponseWrapperFetch.prototype.text = function () {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var textPromise, timer, text, error_1;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        // !!!IMPORTANT: we clone the response to avoid mutating the original response
                        // never call .text(), .json(), etc.. on the original response always clone it first
                        if (!this.clonedResponse) {
                            this.clonedResponse = this.response.clone();
                        }
                        _a.label = 1;
                    case 1:
                        _a.trys.push([1, 3, , 4]);
                        textPromise = this.clonedResponse.text();
                        timer = new Promise(function (resolve) {
                            return setTimeout(
                            /* istanbul ignore next */
                            function () { return resolve(null); }, TEXT_READ_TIMEOUT);
                        });
                        return [4 /*yield*/, Promise.race([textPromise, timer])];
                    case 2:
                        text = _a.sent();
                        return [2 /*return*/, text];
                    case 3:
                        error_1 = _a.sent();
                        return [2 /*return*/, null];
                    case 4: return [2 /*return*/];
                }
            });
        });
    };
    ResponseWrapperFetch.prototype.json = function (allow, exclude) {
        if (allow === void 0) { allow = []; }
        if (exclude === void 0) { exclude = []; }
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var text;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (allow.length === 0) {
                            return [2 /*return*/, null];
                        }
                        return [4 /*yield*/, this.text()];
                    case 1:
                        text = _a.sent();
                        return [2 /*return*/, safeParseAndPruneBody(text, allow, exclude)];
                }
            });
        });
    };
    return ResponseWrapperFetch;
}());
exports.ResponseWrapperFetch = ResponseWrapperFetch;
var ResponseWrapperXhr = /** @class */ (function () {
    function ResponseWrapperXhr(statusCode, headersString, size, getJson) {
        this.statusCode = statusCode;
        this.headersString = headersString;
        this.size = size;
        this.getJson = getJson;
    }
    Object.defineProperty(ResponseWrapperXhr.prototype, "bodySize", {
        get: function () {
            return this.size;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(ResponseWrapperXhr.prototype, "status", {
        get: function () {
            return this.statusCode;
        },
        enumerable: false,
        configurable: true
    });
    ResponseWrapperXhr.prototype.headers = function (allow) {
        var e_3, _a;
        if (allow === void 0) { allow = []; }
        if (!this.headersString) {
            return {};
        }
        var headers = {};
        var headerLines = this.headersString.split('\r\n');
        try {
            for (var headerLines_1 = tslib_1.__values(headerLines), headerLines_1_1 = headerLines_1.next(); !headerLines_1_1.done; headerLines_1_1 = headerLines_1.next()) {
                var line = headerLines_1_1.value;
                var _b = tslib_1.__read(line.split(': '), 2), key = _b[0], value = _b[1];
                if (key && value) {
                    headers[key] = value;
                }
            }
        }
        catch (e_3_1) { e_3 = { error: e_3_1 }; }
        finally {
            try {
                if (headerLines_1_1 && !headerLines_1_1.done && (_a = headerLines_1.return)) _a.call(headerLines_1);
            }
            finally { if (e_3) throw e_3.error; }
        }
        return (0, exports.pruneHeaders)(headers, { allow: allow });
    };
    ResponseWrapperXhr.prototype.json = function (allow, exclude) {
        if (allow === void 0) { allow = []; }
        if (exclude === void 0) { exclude = []; }
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var jsonBody;
            return tslib_1.__generator(this, function (_a) {
                if (allow.length === 0) {
                    return [2 /*return*/, null];
                }
                jsonBody = this.getJson();
                if (jsonBody) {
                    (0, json_query_1.pruneJson)(jsonBody, allow, exclude);
                    return [2 /*return*/, jsonBody];
                }
                return [2 /*return*/, null];
            });
        });
    };
    return ResponseWrapperXhr;
}());
exports.ResponseWrapperXhr = ResponseWrapperXhr;
function safeParseAndPruneBody(text, allow, exclude) {
    if (!text)
        return null;
    try {
        var json = JSON.parse(text);
        (0, json_query_1.pruneJson)(json, allow, exclude);
        return json;
    }
    catch (error) {
        return null;
    }
}
var PRUNE_STRATEGY;
(function (PRUNE_STRATEGY) {
    PRUNE_STRATEGY["REDACT"] = "redact";
    PRUNE_STRATEGY["REMOVE"] = "remove";
})(PRUNE_STRATEGY = exports.PRUNE_STRATEGY || (exports.PRUNE_STRATEGY = {}));
var REDACTED_VALUE = '[REDACTED]';
/**
 * Prune headers from a headers record object.
 * @param headers - The headers to prune.
 * @param options - The options to prune the headers.
 * @param options.exclude - List of headers to delete from headers
 * @param options.include - List of headers to keep in headers, if not provided, all headers are kept by default
 * @returns The pruned headers.
 */
var pruneHeaders = function (headers, options) {
    var e_4, _a;
    var _b = options.allow, allow = _b === void 0 ? [] : _b, _c = options.strategy, strategy = _c === void 0 ? PRUNE_STRATEGY.REMOVE : _c;
    var exclude = tslib_1.__spreadArray([], tslib_1.__read(constants_1.FORBIDDEN_HEADERS), false);
    var headersPruned = {};
    var _loop_1 = function (key) {
        var lowerKey = key.toLowerCase();
        if (exclude.find(function (e) { return e.toLowerCase() === lowerKey; })) {
            if (strategy === PRUNE_STRATEGY.REDACT) {
                headersPruned[key] = REDACTED_VALUE;
            }
        }
        else if (!allow.find(function (i) { return i.toLowerCase() === lowerKey; })) {
            if (strategy === PRUNE_STRATEGY.REDACT) {
                headersPruned[key] = REDACTED_VALUE;
            }
        }
        else {
            headersPruned[key] = headers[key];
        }
    };
    try {
        for (var _d = tslib_1.__values(Object.keys(headers)), _e = _d.next(); !_e.done; _e = _d.next()) {
            var key = _e.value;
            _loop_1(key);
        }
    }
    catch (e_4_1) { e_4 = { error: e_4_1 }; }
    finally {
        try {
            if (_e && !_e.done && (_a = _d.return)) _a.call(_d);
        }
        finally { if (e_4) throw e_4.error; }
    }
    return headersPruned;
};
exports.pruneHeaders = pruneHeaders;
var NetworkRequestEvent = /** @class */ (function () {
    function NetworkRequestEvent(type, method, timestamp, startTime, url, requestWrapper, status, duration, responseWrapper, error, endTime) {
        if (status === void 0) { status = 0; }
        this.type = type;
        this.method = method;
        this.timestamp = timestamp;
        this.startTime = startTime;
        this.url = url;
        this.requestWrapper = requestWrapper;
        this.status = status;
        this.duration = duration;
        this.responseWrapper = responseWrapper;
        this.error = error;
        this.endTime = endTime;
    }
    NetworkRequestEvent.prototype.toSerializable = function () {
        var _a, _b, _c, _d;
        var serialized = {
            type: this.type,
            method: this.method,
            url: this.url,
            timestamp: this.timestamp,
            status: this.status,
            duration: this.duration,
            error: this.error,
            startTime: this.startTime,
            endTime: this.endTime,
            requestHeaders: (_a = this.requestWrapper) === null || _a === void 0 ? void 0 : _a.headers(tslib_1.__spreadArray([], tslib_1.__read(constants_1.SAFE_HEADERS), false)),
            requestBodySize: (_b = this.requestWrapper) === null || _b === void 0 ? void 0 : _b.bodySize,
            responseHeaders: (_c = this.responseWrapper) === null || _c === void 0 ? void 0 : _c.headers(tslib_1.__spreadArray([], tslib_1.__read(constants_1.SAFE_HEADERS), false)),
            responseBodySize: (_d = this.responseWrapper) === null || _d === void 0 ? void 0 : _d.bodySize,
        };
        return Object.fromEntries(Object.entries(serialized).filter(function (_a) {
            var _b = tslib_1.__read(_a, 2), _ = _b[0], v = _b[1];
            return v !== undefined;
        }));
    };
    return NetworkRequestEvent;
}());
exports.NetworkRequestEvent = NetworkRequestEvent;
//# sourceMappingURL=network-request-event.js.map