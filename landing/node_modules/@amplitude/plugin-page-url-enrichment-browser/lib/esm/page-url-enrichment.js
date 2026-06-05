import { __assign, __awaiter, __generator, __read } from "tslib";
import { BrowserStorage, getDecodeURI, getGlobalScope, getPageTitle, replaceSensitiveString, SpecialEventType, } from '@amplitude/analytics-core';
export var CURRENT_PAGE_STORAGE_KEY = 'AMP_CURRENT_PAGE';
export var PREVIOUS_PAGE_STORAGE_KEY = 'AMP_PREVIOUS_PAGE';
export var URL_INFO_STORAGE_KEY = 'AMP_URL_INFO';
var PreviousPageType;
(function (PreviousPageType) {
    PreviousPageType["Direct"] = "direct";
    PreviousPageType["Internal"] = "internal";
    PreviousPageType["External"] = "external";
})(PreviousPageType || (PreviousPageType = {}));
export var EXCLUDED_DEFAULT_EVENT_TYPES = new Set([
    SpecialEventType.IDENTIFY,
    SpecialEventType.GROUP_IDENTIFY,
    SpecialEventType.REVENUE,
]);
export var isPageUrlEnrichmentEnabled = function (option) {
    if (typeof option === 'boolean') {
        return option;
    }
    if (typeof option === 'object' && option !== null && 'pageUrlEnrichment' in option) {
        return Boolean(option.pageUrlEnrichment);
    }
    return false;
};
export var pageUrlEnrichmentPlugin = function (_a) {
    var _b = _a === void 0 ? {} : _a, _c = _b.internalDomains, internalDomains = _c === void 0 ? [] : _c;
    var globalScope = getGlobalScope();
    var sessionStorage = undefined;
    var isStorageEnabled = false;
    var loggerProvider = undefined;
    var isProxied = false;
    var isTracking = false;
    var getHostname = function (url) {
        var hostname;
        try {
            var decodedUrl = getDecodeURI(url, loggerProvider);
            hostname = new URL(decodedUrl).hostname;
        }
        catch (e) {
            /* istanbul ignore next */
            loggerProvider === null || loggerProvider === void 0 ? void 0 : loggerProvider.error('Could not parse URL: ', e);
        }
        return hostname;
    };
    var getPrevPageType = function (previousPage) {
        var currentDomain = (typeof location !== 'undefined' && location.hostname) || '';
        var previousPageDomain = previousPage ? getHostname(previousPage) : undefined;
        if (!previousPageDomain) {
            return PreviousPageType.Direct;
        }
        var isCurrentInternal = internalDomains.some(function (domain) { return currentDomain.indexOf(domain) !== -1; });
        var isPrevInternal = internalDomains.some(function (domain) { return previousPageDomain.indexOf(domain) !== -1; });
        if (currentDomain === previousPageDomain || (isPrevInternal && isCurrentInternal)) {
            return PreviousPageType.Internal;
        }
        return PreviousPageType.External;
    };
    var saveURLInfo = function () { return __awaiter(void 0, void 0, void 0, function () {
        var URLInfo, currentURL, storedCurrentURL, previousURL;
        var _a;
        return __generator(this, function (_b) {
            switch (_b.label) {
                case 0:
                    if (!(sessionStorage && isStorageEnabled)) return [3 /*break*/, 3];
                    return [4 /*yield*/, sessionStorage.get(URL_INFO_STORAGE_KEY)];
                case 1:
                    URLInfo = _b.sent();
                    currentURL = getDecodeURI((typeof location !== 'undefined' && location.href) || '');
                    storedCurrentURL = (URLInfo === null || URLInfo === void 0 ? void 0 : URLInfo[CURRENT_PAGE_STORAGE_KEY]) || '';
                    previousURL = void 0;
                    if (currentURL === storedCurrentURL) {
                        previousURL = (URLInfo === null || URLInfo === void 0 ? void 0 : URLInfo[PREVIOUS_PAGE_STORAGE_KEY]) || '';
                    }
                    else if (storedCurrentURL) {
                        previousURL = storedCurrentURL;
                    }
                    else {
                        previousURL = document.referrer || '';
                    }
                    return [4 /*yield*/, sessionStorage.set(URL_INFO_STORAGE_KEY, (_a = {},
                            _a[CURRENT_PAGE_STORAGE_KEY] = currentURL,
                            _a[PREVIOUS_PAGE_STORAGE_KEY] = previousURL,
                            _a))];
                case 2:
                    _b.sent();
                    _b.label = 3;
                case 3: return [2 /*return*/];
            }
        });
    }); };
    var saveUrlInfoWrapper = function () {
        void saveURLInfo();
    };
    var plugin = {
        name: '@amplitude/plugin-page-url-enrichment-browser',
        type: 'enrichment',
        setup: function (config, _) { return __awaiter(void 0, void 0, void 0, function () {
            var _a;
            return __generator(this, function (_b) {
                switch (_b.label) {
                    case 0:
                        loggerProvider = config.loggerProvider;
                        loggerProvider.log('Installing @amplitude/plugin-page-url-enrichment-browser');
                        isTracking = true;
                        if (!globalScope) return [3 /*break*/, 2];
                        try {
                            sessionStorage = new BrowserStorage(globalScope.sessionStorage);
                        }
                        catch (error) {
                            /* istanbul ignore next */
                            loggerProvider === null || loggerProvider === void 0 ? void 0 : loggerProvider.debug('sessionStorage is not available in this environment.');
                        }
                        return [4 /*yield*/, (sessionStorage === null || sessionStorage === void 0 ? void 0 : sessionStorage.isEnabled())];
                    case 1:
                        isStorageEnabled = (_a = (_b.sent())) !== null && _a !== void 0 ? _a : false;
                        globalScope.addEventListener('popstate', saveUrlInfoWrapper);
                        if (!isProxied) {
                            /* istanbul ignore next */
                            // There is no global browser listener for changes to history, so we have
                            // to modify pushState and replaceState directly.
                            // https://stackoverflow.com/a/64927639
                            // eslint-disable-next-line @typescript-eslint/unbound-method
                            globalScope.history.pushState = new Proxy(globalScope.history.pushState, {
                                apply: function (target, thisArg, _a) {
                                    var _b = __read(_a, 3), state = _b[0], unused = _b[1], url = _b[2];
                                    target.apply(thisArg, [state, unused, url]);
                                    if (isTracking) {
                                        saveUrlInfoWrapper();
                                    }
                                },
                            });
                            // eslint-disable-next-line @typescript-eslint/unbound-method
                            globalScope.history.replaceState = new Proxy(globalScope.history.replaceState, {
                                apply: function (target, thisArg, _a) {
                                    var _b = __read(_a, 3), state = _b[0], unused = _b[1], url = _b[2];
                                    target.apply(thisArg, [state, unused, url]);
                                    if (isTracking) {
                                        saveUrlInfoWrapper();
                                    }
                                },
                            });
                            isProxied = true;
                        }
                        _b.label = 2;
                    case 2: return [2 /*return*/];
                }
            });
        }); },
        execute: function (event) { return __awaiter(void 0, void 0, void 0, function () {
            var locationHREF, URLInfo, updatedURLInfo, previousPage;
            var _a;
            return __generator(this, function (_b) {
                switch (_b.label) {
                    case 0:
                        locationHREF = getDecodeURI((typeof location !== 'undefined' && location.href) || '');
                        if (!(sessionStorage && isStorageEnabled)) return [3 /*break*/, 5];
                        return [4 /*yield*/, sessionStorage.get(URL_INFO_STORAGE_KEY)];
                    case 1:
                        URLInfo = _b.sent();
                        if (!!(URLInfo === null || URLInfo === void 0 ? void 0 : URLInfo[CURRENT_PAGE_STORAGE_KEY])) return [3 /*break*/, 3];
                        return [4 /*yield*/, sessionStorage.set(URL_INFO_STORAGE_KEY, (_a = {},
                                _a[CURRENT_PAGE_STORAGE_KEY] = locationHREF,
                                _a[PREVIOUS_PAGE_STORAGE_KEY] = document.referrer || '',
                                _a))];
                    case 2:
                        _b.sent();
                        _b.label = 3;
                    case 3: return [4 /*yield*/, sessionStorage.get(URL_INFO_STORAGE_KEY)];
                    case 4:
                        updatedURLInfo = _b.sent();
                        previousPage = '';
                        if (updatedURLInfo) {
                            previousPage = updatedURLInfo[PREVIOUS_PAGE_STORAGE_KEY] || '';
                        }
                        // no need to proceed to add additional properties if the event is one of the default event types to be excluded
                        if (EXCLUDED_DEFAULT_EVENT_TYPES.has(event.event_type)) {
                            return [2 /*return*/, event];
                        }
                        event.event_properties = __assign(__assign({}, (event.event_properties || {})), { '[Amplitude] Page Domain': addIfNotExist(event, '[Amplitude] Page Domain', (typeof location !== 'undefined' && location.hostname) || ''), '[Amplitude] Page Location': addIfNotExist(event, '[Amplitude] Page Location', locationHREF), '[Amplitude] Page Path': addIfNotExist(event, '[Amplitude] Page Path', (typeof location !== 'undefined' && getDecodeURI(location.pathname)) || ''), '[Amplitude] Page Title': addIfNotExist(event, '[Amplitude] Page Title', getPageTitle(replaceSensitiveString)), '[Amplitude] Page URL': addIfNotExist(event, '[Amplitude] Page URL', locationHREF.split('?')[0]), '[Amplitude] Previous Page Location': previousPage, '[Amplitude] Previous Page Type': getPrevPageType(previousPage) });
                        _b.label = 5;
                    case 5: return [2 /*return*/, event];
                }
            });
        }); },
        teardown: function () { return __awaiter(void 0, void 0, void 0, function () {
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (globalScope) {
                            globalScope.removeEventListener('popstate', saveUrlInfoWrapper);
                            isTracking = false;
                        }
                        if (!(sessionStorage && isStorageEnabled)) return [3 /*break*/, 2];
                        return [4 /*yield*/, sessionStorage.set(URL_INFO_STORAGE_KEY, {})];
                    case 1:
                        _a.sent();
                        _a.label = 2;
                    case 2: return [2 /*return*/];
                }
            });
        }); },
    };
    return plugin;
};
function addIfNotExist(event, key, value) {
    if (!event.event_properties) {
        event.event_properties = {};
    }
    if (event.event_properties[key] === undefined) {
        return value;
    }
    return event.event_properties[key];
}
//# sourceMappingURL=page-url-enrichment.js.map