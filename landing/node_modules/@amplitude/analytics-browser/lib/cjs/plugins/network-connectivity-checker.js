"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.networkConnectivityCheckerPlugin = void 0;
var tslib_1 = require("tslib");
var analytics_core_1 = require("@amplitude/analytics-core");
var networkConnectivityCheckerPlugin = function () {
    var name = '@amplitude/plugin-network-checker-browser';
    var type = 'before';
    var globalScope = (0, analytics_core_1.getGlobalScope)();
    var eventListeners = [];
    var addNetworkListener = function (type, handler) {
        /* istanbul ignore next */
        if (globalScope === null || globalScope === void 0 ? void 0 : globalScope.addEventListener) {
            globalScope === null || globalScope === void 0 ? void 0 : globalScope.addEventListener(type, handler);
            eventListeners.push({
                type: type,
                handler: handler,
            });
        }
    };
    var removeNetworkListeners = function () {
        eventListeners.forEach(function (_a) {
            var type = _a.type, handler = _a.handler;
            /* istanbul ignore next */
            globalScope === null || globalScope === void 0 ? void 0 : globalScope.removeEventListener(type, handler);
        });
        eventListeners = [];
    };
    var setup = function (config, amplitude) { return tslib_1.__awaiter(void 0, void 0, void 0, function () {
        return tslib_1.__generator(this, function (_a) {
            if (typeof navigator === 'undefined') {
                config.loggerProvider.debug('Network connectivity checker plugin is disabled because navigator is not available.');
                config.offline = false;
                return [2 /*return*/];
            }
            config.offline = !navigator.onLine;
            addNetworkListener('online', function () {
                config.loggerProvider.debug('Network connectivity changed to online.');
                config.offline = false;
                // Flush immediately will cause ERR_NETWORK_CHANGED
                setTimeout(function () {
                    amplitude.flush();
                }, config.flushIntervalMillis);
            });
            addNetworkListener('offline', function () {
                config.loggerProvider.debug('Network connectivity changed to offline.');
                config.offline = true;
            });
            return [2 /*return*/];
        });
    }); };
    var teardown = function () { return tslib_1.__awaiter(void 0, void 0, void 0, function () {
        return tslib_1.__generator(this, function (_a) {
            removeNetworkListeners();
            return [2 /*return*/];
        });
    }); };
    return {
        name: name,
        type: type,
        setup: setup,
        teardown: teardown,
    };
};
exports.networkConnectivityCheckerPlugin = networkConnectivityCheckerPlugin;
//# sourceMappingURL=network-connectivity-checker.js.map