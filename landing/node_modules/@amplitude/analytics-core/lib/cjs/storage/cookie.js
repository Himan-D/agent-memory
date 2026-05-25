"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.isDomainEqual = exports.decodeCookieValue = exports.CookieStorage = void 0;
var tslib_1 = require("tslib");
var global_scope_1 = require("../global-scope");
/* istanbul ignore next */
var getLocks = function () {
    var _a;
    var globalScope = (0, global_scope_1.getGlobalScope)();
    return (_a = globalScope === null || globalScope === void 0 ? void 0 : globalScope.navigator) === null || _a === void 0 ? void 0 : _a.locks;
};
var CookieStorage = /** @class */ (function () {
    function CookieStorage(options, config) {
        if (config === void 0) { config = {}; }
        this.options = tslib_1.__assign({}, options);
        this.config = config;
    }
    CookieStorage.prototype.isEnabled = function () {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var testKey, testCookieOptions, testStorage, testValue;
            var _this = this;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        testKey = 'AMP_TEST';
                        testCookieOptions = tslib_1.__assign({}, this.options);
                        testStorage = new CookieStorage(testCookieOptions);
                        testValue = String(Date.now());
                        return [4 /*yield*/, testStorage.transaction(testKey, function (storage) {
                                var _a, _b;
                                try {
                                    storage.set(testValue);
                                    var value = storage.get();
                                    var result = value === testValue;
                                    /* istanbul ignore next */
                                    if (!result && _this.config.diagnosticsClient) {
                                        (_a = _this.config.diagnosticsClient) === null || _a === void 0 ? void 0 : _a.recordEvent('cookies.isEnabled.failure', {
                                            reason: 'Test Value mismatch',
                                            testKey: testKey,
                                            testValue: testValue,
                                            sync: true,
                                        });
                                    }
                                    return result;
                                }
                                catch (e) {
                                    /* istanbul ignore next */
                                    if (_this.config.diagnosticsClient) {
                                        var errMessage = e instanceof Error ? e.message : String(e);
                                        (_b = _this.config.diagnosticsClient) === null || _b === void 0 ? void 0 : _b.recordEvent('cookies.isEnabled.failure', {
                                            reason: 'Cookie getter/setter failed',
                                            testKey: testKey,
                                            testValue: testValue,
                                            error: errMessage,
                                            sync: true,
                                        });
                                    }
                                    return false;
                                }
                                finally {
                                    // clean-up the AMP_TEST cookie behind us
                                    storage.set(null);
                                }
                            })];
                    case 1: return [2 /*return*/, _a.sent()];
                }
            });
        });
    };
    CookieStorage.prototype.get = function (key) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var value;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.getRaw(key)];
                    case 1:
                        value = _a.sent();
                        return [2 /*return*/, this.decodeCookieValue(key, value)];
                }
            });
        });
    };
    CookieStorage.prototype.decodeCookieValue = function (key, value) {
        if (!value) {
            return undefined;
        }
        try {
            var decodedValue = (0, exports.decodeCookieValue)(value);
            if (decodedValue === undefined) {
                console.error("Amplitude Logger [Error]: Failed to decode cookie value for key: ".concat(key, ", value: ").concat(value));
                return undefined;
            }
            // eslint-disable-next-line @typescript-eslint/no-unsafe-return
            return JSON.parse(decodedValue);
        }
        catch (_a) {
            console.error("Amplitude Logger [Error]: Failed to parse cookie value for key: ".concat(key, ", value: ").concat(value));
            return undefined;
        }
    };
    CookieStorage.prototype.getSync = function (key) {
        var value = this.getRawSync(key);
        return this.decodeCookieValue(key, value);
    };
    CookieStorage.prototype.getRaw = function (key) {
        var _a, _b;
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var globalScope, globalScopeWithCookiesStore, cookieStore, cookies, cookies_1, cookies_1_1, cookie, ignoreError_1;
            var e_1, _c;
            return tslib_1.__generator(this, function (_d) {
                switch (_d.label) {
                    case 0:
                        globalScope = (0, global_scope_1.getGlobalScope)();
                        globalScopeWithCookiesStore = globalScope;
                        _d.label = 1;
                    case 1:
                        _d.trys.push([1, 4, , 5]);
                        cookieStore = globalScopeWithCookiesStore === null || globalScopeWithCookiesStore === void 0 ? void 0 : globalScopeWithCookiesStore.cookieStore;
                        if (!cookieStore) return [3 /*break*/, 3];
                        return [4 /*yield*/, cookieStore.getAll(key)];
                    case 2:
                        cookies = _d.sent();
                        if (cookies) {
                            /* istanbul ignore if */
                            if (cookies.length > 1) {
                                (_a = this.config.diagnosticsClient) === null || _a === void 0 ? void 0 : _a.recordEvent('cookies.duplicate', {
                                    cookies: cookies.map(function (cookie) { return cookie.domain; }),
                                });
                                (_b = this.config.diagnosticsClient) === null || _b === void 0 ? void 0 : _b.increment('cookies.duplicate.occurrence.cookieStore');
                            }
                            try {
                                for (cookies_1 = tslib_1.__values(cookies), cookies_1_1 = cookies_1.next(); !cookies_1_1.done; cookies_1_1 = cookies_1.next()) {
                                    cookie = cookies_1_1.value;
                                    if ((0, exports.isDomainEqual)(cookie.domain, this.options.domain)) {
                                        return [2 /*return*/, cookie.value];
                                    }
                                }
                            }
                            catch (e_1_1) { e_1 = { error: e_1_1 }; }
                            finally {
                                try {
                                    if (cookies_1_1 && !cookies_1_1.done && (_c = cookies_1.return)) _c.call(cookies_1);
                                }
                                finally { if (e_1) throw e_1.error; }
                            }
                        }
                        _d.label = 3;
                    case 3: return [3 /*break*/, 5];
                    case 4:
                        ignoreError_1 = _d.sent();
                        return [3 /*break*/, 5];
                    case 5: return [2 /*return*/, this.getRawSync(key)];
                }
            });
        });
    };
    CookieStorage.prototype.getRawSync = function (key) {
        var _this = this;
        var _a, _b;
        var globalScope = (0, global_scope_1.getGlobalScope)();
        var cookies = ((_b = (_a = globalScope === null || globalScope === void 0 ? void 0 : globalScope.document) === null || _a === void 0 ? void 0 : _a.cookie.split('; ')) !== null && _b !== void 0 ? _b : []).filter(function (c) { return c.indexOf(key + '=') === 0; });
        var match = undefined;
        // if matcher function is provided, use it to de-duplicate when there's more than one cookie
        /* istanbul ignore if */
        var duplicateResolverFn = this.config.duplicateResolverFn;
        if (typeof duplicateResolverFn === 'function' && cookies.length > 1) {
            match = cookies.find(function (c) {
                var _a;
                try {
                    var res = duplicateResolverFn(c.substring(key.length + 1));
                    if (!res) {
                        (_a = _this.config.diagnosticsClient) === null || _a === void 0 ? void 0 : _a.increment('cookies.duplicate.occurrence.document.cookie');
                    }
                    return res;
                }
                catch (ignoreError) {
                    /* istanbul ignore next */
                    return false;
                }
            });
        }
        // if match was not found, just get the first one that matches the key
        if (!match) {
            match = cookies[0];
        }
        if (!match) {
            return undefined;
        }
        return match.substring(key.length + 1);
    };
    CookieStorage.prototype.set = function (key, value) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            return tslib_1.__generator(this, function (_a) {
                this.setSync(key, value);
                return [2 /*return*/];
            });
        });
    };
    CookieStorage.prototype.setSync = function (key, value) {
        var _a;
        try {
            var expirationDays = (_a = this.options.expirationDays) !== null && _a !== void 0 ? _a : 0;
            var expires = value !== null ? expirationDays : -1;
            var expireDate = undefined;
            if (expires) {
                var date = new Date();
                date.setTime(date.getTime() + expires * 24 * 60 * 60 * 1000);
                expireDate = date;
            }
            var str = "".concat(key, "=").concat(btoa(encodeURIComponent(JSON.stringify(value))));
            if (expireDate) {
                str += "; expires=".concat(expireDate.toUTCString());
            }
            str += '; path=/';
            if (this.options.domain) {
                str += "; domain=".concat(this.options.domain);
            }
            if (this.options.secure) {
                str += '; Secure';
            }
            if (this.options.sameSite) {
                str += "; SameSite=".concat(this.options.sameSite);
            }
            var globalScope = (0, global_scope_1.getGlobalScope)();
            if (globalScope === null || globalScope === void 0 ? void 0 : globalScope.document) {
                globalScope.document.cookie = str;
            }
        }
        catch (error) {
            var errorMessage = error instanceof Error ? error.message : String(error);
            console.error("Amplitude Logger [Error]: Failed to set cookie for key: ".concat(key, ". Error: ").concat(errorMessage));
        }
    };
    CookieStorage.prototype.remove = function (key) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.set(key, null)];
                    case 1:
                        _a.sent();
                        return [2 /*return*/];
                }
            });
        });
    };
    CookieStorage.prototype.reset = function () {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            return tslib_1.__generator(this, function (_a) {
                return [2 /*return*/];
            });
        });
    };
    CookieStorage.isDomainWritable = function (domain) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var options, storageKey, storage, res, error_1;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (CookieStorage.cachedTlds[domain]) {
                            return [2 /*return*/, true];
                        }
                        options = {
                            domain: '.' + domain,
                        };
                        storageKey = 'AMP_TLDTEST';
                        storage = new CookieStorage(options);
                        _a.label = 1;
                    case 1:
                        _a.trys.push([1, 3, , 4]);
                        return [4 /*yield*/, storage.transaction(storageKey, function (storageSync) {
                                if (CookieStorage.cachedTlds[domain]) {
                                    return true;
                                }
                                try {
                                    storageSync.set(1);
                                    var result = !!storageSync.get();
                                    if (result) {
                                        CookieStorage.cachedTlds[domain] = true;
                                    }
                                    return result;
                                }
                                finally {
                                    storageSync.set(null);
                                }
                            })];
                    case 2:
                        res = _a.sent();
                        return [2 /*return*/, !!res];
                    case 3:
                        error_1 = _a.sent();
                        return [2 /*return*/, false];
                    case 4: return [2 /*return*/];
                }
            });
        });
    };
    CookieStorage.prototype.transaction = function (key, callback) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var locks, callbackWrapper, error_2;
            var _this = this;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        locks = getLocks();
                        callbackWrapper = function () {
                            // construct a sync storage object that is scoped to
                            // Cookie with name <key>
                            var storageSync = {
                                get: function () { return _this.getSync(key); },
                                set: function (value) { return _this.setSync(key, value); },
                            };
                            return callback(storageSync);
                        };
                        // if 'locks' is missing, it is a legacy browser, just call the callback directly
                        // and settle for a transaction that isn't isolated across tabs
                        if (!locks) {
                            return [2 /*return*/, callbackWrapper()];
                        }
                        _a.label = 1;
                    case 1:
                        _a.trys.push([1, 3, , 4]);
                        return [4 /*yield*/, locks.request("com.amplitude:cookie-lock:".concat(key), callbackWrapper)];
                    case 2: return [2 /*return*/, (_a.sent())];
                    case 3:
                        error_2 = _a.sent();
                        return [2 /*return*/, callbackWrapper()];
                    case 4: return [2 /*return*/];
                }
            });
        });
    };
    CookieStorage.cachedTlds = {};
    return CookieStorage;
}());
exports.CookieStorage = CookieStorage;
var decodeCookiesAsDefault = function (value) {
    try {
        return decodeURIComponent(atob(value));
    }
    catch (_a) {
        return undefined;
    }
};
var decodeCookiesWithDoubleUrlEncoding = function (value) {
    // Modern Ruby (v7+) automatically encodes cookies with URL encoding by
    // https://api.rubyonrails.org/classes/ActionDispatch/Cookies.html
    try {
        return decodeURIComponent(atob(decodeURIComponent(value)));
    }
    catch (_a) {
        return undefined;
    }
};
/**
 * Decodes a cookie value that was encoded with btoa(encodeURIComponent(...)).
 * Handles both standard encoding and double URL encoding (used by Ruby Rails v7+).
 */
var decodeCookieValue = function (value) {
    var _a;
    return (_a = decodeCookiesAsDefault(value)) !== null && _a !== void 0 ? _a : decodeCookiesWithDoubleUrlEncoding(value);
};
exports.decodeCookieValue = decodeCookieValue;
/**
 * Compares two domain strings for equality, ignoring leading dots.
 * This is useful for comparing cookie domains since ".example.com" and "example.com"
 * are effectively equivalent for cookie scoping.
 */
var isDomainEqual = function (domain1, domain2) {
    if (domain1 === '' && domain2 === '') {
        return true;
    }
    if (!domain1 || !domain2) {
        return false;
    }
    var normalized1 = domain1.startsWith('.') ? domain1.substring(1) : domain1;
    var normalized2 = domain2.startsWith('.') ? domain2.substring(1) : domain2;
    return normalized1.toLowerCase() === normalized2.toLowerCase();
};
exports.isDomainEqual = isDomainEqual;
//# sourceMappingURL=cookie.js.map