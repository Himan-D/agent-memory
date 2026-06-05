"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.customEnrichmentPlugin = void 0;
var tslib_1 = require("tslib");
var customEnrichmentPlugin = function () {
    var loggerProvider;
    var unsubscribe;
    var enrichEvent;
    function isCustomEnrichmentConfig(config) {
        // 1. Check if it's an object and not null
        if (typeof config !== 'object' || config === null) {
            return false;
        }
        // 2. Validate specific properties exist and are the correct type
        return 'body' in config && typeof config.body === 'string';
    }
    function createEnrichEvent(body) {
        if (body) {
            try {
                // eslint-disable-next-line @typescript-eslint/no-unsafe-call, @typescript-eslint/no-implied-eval
                var fn = new Function('return ' + body)();
                if (typeof fn === 'function') {
                    return fn;
                }
                loggerProvider === null || loggerProvider === void 0 ? void 0 : loggerProvider.error('Custom enrichment body did not evaluate to a function');
            }
            catch (error) {
                loggerProvider === null || loggerProvider === void 0 ? void 0 : loggerProvider.error('Could not create custom enrichment function', error);
            }
        }
        return function (event) { return event; };
    }
    var plugin = {
        name: '@amplitude/plugin-custom-enrichment-browser',
        type: 'enrichment',
        setup: function (config, _) { return tslib_1.__awaiter(void 0, void 0, void 0, function () {
            var subscriptionId_1;
            var _a;
            return tslib_1.__generator(this, function (_b) {
                loggerProvider = config.loggerProvider;
                loggerProvider === null || loggerProvider === void 0 ? void 0 : loggerProvider.log('Installing @amplitude/plugin-custom-enrichment-browser');
                // Fetch remote config for custom enrichment in a non-blocking manner
                if ((_a = config.remoteConfig) === null || _a === void 0 ? void 0 : _a.fetchRemoteConfig) {
                    if (!config.remoteConfigClient) {
                        // TODO(xinyi): Diagnostics.recordEvent
                        loggerProvider === null || loggerProvider === void 0 ? void 0 : loggerProvider.debug('Remote config client is not provided, skipping remote config fetch');
                    }
                    else {
                        subscriptionId_1 = config.remoteConfigClient.subscribe('configs.analyticsSDK.browserSDK.customEnrichment', 'all', function (remoteConfig) {
                            if (remoteConfig && isCustomEnrichmentConfig(remoteConfig)) {
                                enrichEvent = createEnrichEvent(remoteConfig.body || '');
                            }
                            else {
                                // if there is no valid body, clear enrich event
                                enrichEvent = createEnrichEvent('');
                            }
                        });
                        unsubscribe = function () { var _a; return (_a = config.remoteConfigClient) === null || _a === void 0 ? void 0 : _a.unsubscribe(subscriptionId_1); };
                    }
                }
                return [2 /*return*/];
            });
        }); },
        execute: function (event) { return tslib_1.__awaiter(void 0, void 0, void 0, function () {
            var _a;
            return tslib_1.__generator(this, function (_b) {
                if (enrichEvent) {
                    try {
                        return [2 /*return*/, (_a = enrichEvent(event)) !== null && _a !== void 0 ? _a : null];
                    }
                    catch (error) {
                        loggerProvider === null || loggerProvider === void 0 ? void 0 : loggerProvider.error('Could not execute custom enrichment function', error);
                        return [2 /*return*/, event];
                    }
                }
                return [2 /*return*/, event];
            });
        }); },
        teardown: function () { return tslib_1.__awaiter(void 0, void 0, void 0, function () {
            return tslib_1.__generator(this, function (_a) {
                if (unsubscribe) {
                    unsubscribe();
                }
                return [2 /*return*/];
            });
        }); },
    };
    return plugin;
};
exports.customEnrichmentPlugin = customEnrichmentPlugin;
//# sourceMappingURL=custom-enrichment.js.map