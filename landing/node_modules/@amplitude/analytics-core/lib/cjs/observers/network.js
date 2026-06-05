"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.networkObserver = exports.NetworkObserver = exports.NetworkEventCallback = void 0;
var tslib_1 = require("tslib");
var global_scope_1 = require("../global-scope");
var uuid_1 = require("../utils/uuid");
var network_request_event_1 = require("../network-request-event");
/**
 * Typeguard function checks if an input is a Request object.
 */
function isRequest(requestInfo) {
    return typeof requestInfo === 'object' && requestInfo !== null && 'url' in requestInfo && 'method' in requestInfo;
}
var NetworkEventCallback = /** @class */ (function () {
    function NetworkEventCallback(callback, id) {
        if (id === void 0) { id = (0, uuid_1.UUID)(); }
        this.callback = callback;
        this.id = id;
    }
    return NetworkEventCallback;
}());
exports.NetworkEventCallback = NetworkEventCallback;
function safeInvoke(fn) {
    try {
        fn();
    }
    catch (err) {
        // swallow the error
    }
}
var NetworkObserver = /** @class */ (function () {
    function NetworkObserver(logger) {
        this.eventCallbacks = new Map();
        this.isObserving = false;
        this.logger = logger;
        var globalScope = (0, global_scope_1.getGlobalScope)();
        if (!NetworkObserver.isSupported()) {
            /* istanbul ignore next */
            return;
        }
        this.globalScope = globalScope;
    }
    NetworkObserver.isSupported = function () {
        var globalScope = (0, global_scope_1.getGlobalScope)();
        return !!globalScope && !!globalScope.fetch;
    };
    NetworkObserver.prototype.subscribe = function (eventCallback, logger) {
        var _a, _b, _c, _d, _e, _f, _g, _h, _j, _k;
        if (!this.logger) {
            this.logger = logger;
        }
        this.eventCallbacks.set(eventCallback.id, eventCallback);
        if (!this.isObserving) {
            /* istanbul ignore next */
            // eslint-disable-next-line @typescript-eslint/unbound-method
            var originalXhrOpen = (_c = (_b = (_a = this.globalScope) === null || _a === void 0 ? void 0 : _a.XMLHttpRequest) === null || _b === void 0 ? void 0 : _b.prototype) === null || _c === void 0 ? void 0 : _c.open;
            /* istanbul ignore next */
            // eslint-disable-next-line @typescript-eslint/unbound-method
            var originalXhrSend = (_f = (_e = (_d = this.globalScope) === null || _d === void 0 ? void 0 : _d.XMLHttpRequest) === null || _e === void 0 ? void 0 : _e.prototype) === null || _f === void 0 ? void 0 : _f.send;
            /* istanbul ignore next */
            // eslint-disable-next-line @typescript-eslint/unbound-method
            var originalXhrSetRequestHeader = (_j = (_h = (_g = this.globalScope) === null || _g === void 0 ? void 0 : _g.XMLHttpRequest) === null || _h === void 0 ? void 0 : _h.prototype) === null || _j === void 0 ? void 0 : _j.setRequestHeader;
            if (originalXhrOpen && originalXhrSend && originalXhrSetRequestHeader) {
                this.observeXhr(originalXhrOpen, originalXhrSend, originalXhrSetRequestHeader);
            }
            /* istanbul ignore next */
            var originalFetch = (_k = this.globalScope) === null || _k === void 0 ? void 0 : _k.fetch;
            /* istanbul ignore next */
            if (originalFetch) {
                this.observeFetch(originalFetch);
            }
            /* istanbul ignore next */
            this.isObserving = true;
        }
    };
    NetworkObserver.prototype.unsubscribe = function (eventCallback) {
        this.eventCallbacks.delete(eventCallback.id);
    };
    NetworkObserver.prototype.triggerEventCallbacks = function (event) {
        var _this = this;
        this.eventCallbacks.forEach(function (callback) {
            try {
                callback.callback(event);
            }
            catch (err) {
                // if the callback throws an error, we should catch it
                // to avoid breaking the fetch promise chain
                safeInvoke(function () {
                    var _a;
                    /* istanbul ignore next */
                    (_a = _this.logger) === null || _a === void 0 ? void 0 : _a.debug('an unexpected error occurred while triggering event callbacks', err);
                });
            }
        });
    };
    NetworkObserver.prototype.handleNetworkRequestEvent = function (requestType, requestInfo, requestWrapper, responseWrapper, typedError, startTime, durationStart) {
        var _a;
        /* istanbul ignore next */
        if (startTime === undefined || durationStart === undefined) {
            // if we reach this point, it means that the performance API is not supported
            // so we can't construct a NetworkRequestEvent
            return;
        }
        // parse the URL and Method
        var url;
        var method = 'GET';
        if (isRequest(requestInfo)) {
            url = requestInfo['url'];
            method = requestInfo['method'];
        }
        else {
            url = (_a = requestInfo === null || requestInfo === void 0 ? void 0 : requestInfo.toString) === null || _a === void 0 ? void 0 : _a.call(requestInfo);
        }
        // strip basic auth from the URL
        if (url) {
            try {
                var parsedUrl = new URL(url);
                // reconstruct the URL without the basic auth
                url = "".concat(parsedUrl.protocol, "//").concat(parsedUrl.host).concat(parsedUrl.pathname).concat(parsedUrl.search).concat(parsedUrl.hash);
                // eslint-disable-next-line no-empty
            }
            catch (err) { }
        }
        method = (requestWrapper === null || requestWrapper === void 0 ? void 0 : requestWrapper.method) || method;
        var status, error;
        if (responseWrapper) {
            status = responseWrapper.status;
        }
        if (typedError) {
            error = {
                name: typedError.name || 'UnknownError',
                message: typedError.message || 'An unknown error occurred',
            };
            status = 0;
        }
        var duration = Math.floor(performance.now() - durationStart);
        var endTime = Math.floor(startTime + duration);
        var requestEvent = new network_request_event_1.NetworkRequestEvent(requestType, method, startTime, // timestamp and startTime are aliases
        startTime, url, requestWrapper, status, duration, responseWrapper, error, endTime);
        this.triggerEventCallbacks(requestEvent);
    };
    NetworkObserver.prototype.getTimestamps = function () {
        var _a, _b;
        /* istanbul ignore next */
        return {
            startTime: (_a = Date.now) === null || _a === void 0 ? void 0 : _a.call(Date),
            durationStart: (_b = performance === null || performance === void 0 ? void 0 : performance.now) === null || _b === void 0 ? void 0 : _b.call(performance),
        };
    };
    NetworkObserver.prototype.observeFetch = function (originalFetch) {
        var _this = this;
        /* istanbul ignore next */
        if (!this.globalScope || !originalFetch) {
            return;
        }
        /**
         * IMPORTANT: This overrides window.fetch in browsers.
         * You probably never need to make changes to this function.
         * If you do, please be careful to preserve the original functionality of fetch
         * and make sure another developer who is an expert reviews this change throughly
         */
        this.globalScope.fetch = function (requestInfo, requestInit) { return tslib_1.__awaiter(_this, void 0, void 0, function () {
            var timestamps, originalResponse, originalError, err_1;
            var _this = this;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        try {
                            timestamps = this.getTimestamps();
                        }
                        catch (error) {
                            /* istanbul ignore next */
                            safeInvoke(function () { var _a; return (_a = _this.logger) === null || _a === void 0 ? void 0 : _a.debug('an unexpected error occurred while retrieving timestamps', error); });
                        }
                        _a.label = 1;
                    case 1:
                        _a.trys.push([1, 3, , 4]);
                        return [4 /*yield*/, originalFetch(requestInfo, requestInit)];
                    case 2:
                        originalResponse = _a.sent();
                        return [3 /*break*/, 4];
                    case 3:
                        err_1 = _a.sent();
                        // Capture error information
                        originalError = err_1;
                        return [3 /*break*/, 4];
                    case 4:
                        // 3. call the handler after the fetch call is done
                        try {
                            this.handleNetworkRequestEvent('fetch', requestInfo, requestInit ? new network_request_event_1.RequestWrapperFetch(requestInit) : undefined, originalResponse ? new network_request_event_1.ResponseWrapperFetch(originalResponse) : undefined, originalError, 
                            /* istanbul ignore next */
                            timestamps === null || timestamps === void 0 ? void 0 : timestamps.startTime, 
                            /* istanbul ignore next */
                            timestamps === null || timestamps === void 0 ? void 0 : timestamps.durationStart);
                        }
                        catch (err) {
                            // this catch shouldn't be reachable, but keep it here for safety
                            // because we're overriding the fetch function and better to be safe than sorry
                            /* istanbul ignore next */
                            safeInvoke(function () { var _a; return (_a = _this.logger) === null || _a === void 0 ? void 0 : _a.debug('an unexpected error occurred while handling fetch', err); });
                        }
                        // 4. return the original response or throw the original error
                        if (originalResponse) {
                            // if the response is not undefined, return it
                            return [2 /*return*/, originalResponse];
                        }
                        else {
                            throw originalError;
                        }
                        return [2 /*return*/];
                }
            });
        }); };
    };
    /**
     * Creates a function that parses the response of an XMLHttpRequest as JSON.
     *
     * Returns function instead of JSON object to avoid unnecessary parsing if the
     * body is not being captured.
     *
     * @param xhrSafe - The XMLHttpRequest object.
     * @param context - The NetworkObserver instance.
     * @returns A function that parses the response of an XMLHttpRequest as JSON.
     */
    NetworkObserver.createXhrJsonParser = function (xhrUnsafe, context) {
        return function () {
            var _a;
            try {
                if (xhrUnsafe.responseType === 'json') {
                    // if response is a JS object, clone it so that subscribers can't mutate it
                    if ((_a = context.globalScope) === null || _a === void 0 ? void 0 : _a.structuredClone) {
                        /* eslint-disable-next-line @typescript-eslint/no-unsafe-return */
                        return context.globalScope.structuredClone(xhrUnsafe.response);
                    }
                }
                else if (['text', ''].includes(xhrUnsafe.responseType)) {
                    // if response is a string, parse it as JSON
                    /* eslint-disable-next-line @typescript-eslint/no-unsafe-return */
                    return JSON.parse(xhrUnsafe.responseText);
                }
            }
            catch (err) {
                /* istanbul ignore if */
                if (err instanceof Error && err.name === 'InvalidStateError') {
                    // https://developer.mozilla.org/en-US/docs/Web/API/XMLHttpRequest/responseText#exceptions
                    // if we reach here, it means we don't handle responseType correctly
                    safeInvoke(function () {
                        var _a;
                        return (_a = context.logger) === null || _a === void 0 ? void 0 : _a.debug("unexpected error when retrieving responseText. responseType='".concat(xhrUnsafe.responseType, "'"));
                    });
                }
                // the other possible error is Json Parse error which we fail silently
                return null;
            }
            return null;
        };
    };
    NetworkObserver.prototype.observeXhr = function (originalXhrOpen, originalXhrSend, originalXhrSetRequestHeader) {
        /* istanbul ignore next */
        if (!this.globalScope || !originalXhrOpen || !originalXhrSend) {
            return;
        }
        var xhrProto = this.globalScope.XMLHttpRequest.prototype;
        var networkObserverContext = this;
        /**
         * IMPORTANT: This overrides window.XMLHttpRequest.prototype.open
         * You probably never need to make changes to this function.
         * If you do, please be careful to preserve the original functionality of xhr.open
         * and make sure another developer who is an expert reviews this change throughly
         */
        xhrProto.open = function () {
            var _a;
            var args = [];
            for (var _i = 0; _i < arguments.length; _i++) {
                args[_i] = arguments[_i];
            }
            var xhrSafe = this;
            var _b = tslib_1.__read(args, 2), method = _b[0], url = _b[1];
            try {
                /* istanbul ignore next */
                xhrSafe.$$AmplitudeAnalyticsEvent = tslib_1.__assign({ method: method, url: (_a = url === null || url === void 0 ? void 0 : url.toString) === null || _a === void 0 ? void 0 : _a.call(url), headers: {} }, networkObserverContext.getTimestamps());
            }
            catch (err) {
                /* istanbul ignore next */
                safeInvoke(function () { var _a; return (_a = networkObserverContext.logger) === null || _a === void 0 ? void 0 : _a.debug('an unexpected error occurred while calling xhr open', err); });
            }
            // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
            return originalXhrOpen.apply(xhrSafe, args);
        };
        /**
         * IMPORTANT: This overrides window.XMLHttpRequest.prototype.send
         * You probably never need to make changes to this function.
         * If you do, please be careful to preserve the original functionality of xhr.send
         * and make sure another developer who is an expert reviews this change throughly
         */
        // allow "any" type for args to reflect how it's used in the browser
        /* eslint-disable-next-line @typescript-eslint/no-unsafe-argument */
        xhrProto.send = function () {
            var args = [];
            for (var _i = 0; _i < arguments.length; _i++) {
                args[_i] = arguments[_i];
            }
            // eslint-disable-next-line @typescript-eslint/no-this-alias
            var xhrUnsafe = this;
            var xhrSafe = xhrUnsafe;
            var getJson = NetworkObserver.createXhrJsonParser(xhrUnsafe, networkObserverContext);
            var body = args[0];
            var requestEvent = xhrSafe.$$AmplitudeAnalyticsEvent;
            // if xhrSafe.$$AmplitudeAnalyticsEvent is not set, it means that
            // the xhr.open method was called before we monkey-patched XHR and
            // the event is missed
            if (xhrSafe.$$AmplitudeAnalyticsEvent) {
                xhrSafe.addEventListener('loadend', function () {
                    try {
                        var responseHeaders = xhrSafe.getAllResponseHeaders();
                        var responseBodySize = xhrSafe.getResponseHeader('content-length');
                        var responseWrapper = new network_request_event_1.ResponseWrapperXhr(xhrSafe.status, responseHeaders, 
                        /* istanbul ignore next */
                        responseBodySize ? parseInt(responseBodySize, 10) : undefined, getJson);
                        var requestHeaders = xhrSafe.$$AmplitudeAnalyticsEvent.headers;
                        var requestWrapper = new network_request_event_1.RequestWrapperXhr(body, requestHeaders);
                        requestEvent.status = xhrSafe.status;
                        networkObserverContext.handleNetworkRequestEvent('xhr', { url: requestEvent.url, method: requestEvent.method }, requestWrapper, responseWrapper, undefined, requestEvent.startTime, requestEvent.durationStart);
                    }
                    catch (err) {
                        /* istanbul ignore next */
                        safeInvoke(function () { var _a; return (_a = networkObserverContext.logger) === null || _a === void 0 ? void 0 : _a.debug('an unexpected error occurred while handling xhr send', err); });
                    }
                });
            }
            /* eslint-disable-next-line @typescript-eslint/no-unsafe-argument */
            return originalXhrSend.apply(xhrSafe, args);
        };
        /**
         * IMPORTANT: This overrides window.XMLHttpRequest.prototype.setRequestHeader
         * You probably never need to make changes to this function.
         * If you do, please be careful to preserve the original functionality of xhr.setRequestHeader
         * and make sure another developer who is an expert reviews this change throughly
         */
        // allow "any" type for args to reflect how it's used in the browser
        /* eslint-disable-next-line @typescript-eslint/no-unsafe-argument */
        xhrProto.setRequestHeader = function (headerName, headerValue) {
            var xhrSafe = this;
            try {
                var analyticsEvent = xhrSafe.$$AmplitudeAnalyticsEvent;
                if (analyticsEvent) {
                    /* eslint-disable-next-line @typescript-eslint/no-unsafe-assignment */
                    analyticsEvent.headers[headerName] = headerValue;
                }
            }
            catch (err) {
                /* istanbul ignore next */
                safeInvoke(function () { var _a; return (_a = networkObserverContext.logger) === null || _a === void 0 ? void 0 : _a.debug('an unexpected error occurred while calling xhr setRequestHeader', err); });
            }
            /* eslint-disable-next-line @typescript-eslint/no-unsafe-argument */
            originalXhrSetRequestHeader.apply(xhrSafe, [headerName, headerValue]);
        };
    };
    return NetworkObserver;
}());
exports.NetworkObserver = NetworkObserver;
// singleton instance of NetworkObserver
exports.networkObserver = new NetworkObserver();
//# sourceMappingURL=network.js.map