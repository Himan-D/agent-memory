"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.networkCapturePlugin = exports.ObservablesEnum = void 0;
var tslib_1 = require("tslib");
/* eslint-disable no-restricted-globals */
var analytics_core_1 = require("@amplitude/analytics-core");
var constants = tslib_1.__importStar(require("./constants"));
var analytics_core_2 = require("@amplitude/analytics-core");
var track_network_event_1 = require("./track-network-event");
var ObservablesEnum;
(function (ObservablesEnum) {
    ObservablesEnum["NetworkObservable"] = "networkObservable";
})(ObservablesEnum = exports.ObservablesEnum || (exports.ObservablesEnum = {}));
var subscription;
var networkCapturePlugin = function (options) {
    if (options === void 0) { options = {}; }
    var name = constants.PLUGIN_NAME;
    var type = 'enrichment';
    var logger;
    var addAdditionalEventProperties = function (event, type) {
        var baseEvent = {
            event: event,
            timestamp: Date.now(),
            type: type,
        };
        return baseEvent;
    };
    // Create observables on events on the window
    var createObservables = function () {
        var _a;
        var networkObservable = new analytics_core_2.Observable(function (observer) {
            var callback = new analytics_core_1.NetworkEventCallback(function (event) {
                var eventWithProperties = addAdditionalEventProperties(event, 'network');
                observer.next(eventWithProperties);
            });
            analytics_core_1.networkObserver.subscribe(callback, logger);
            return function () {
                analytics_core_1.networkObserver.unsubscribe(callback);
            };
        });
        return _a = {},
            _a[ObservablesEnum.NetworkObservable] = networkObservable,
            _a;
    };
    var setup = function (config, amplitude) { return tslib_1.__awaiter(void 0, void 0, void 0, function () {
        var allObservables;
        return tslib_1.__generator(this, function (_a) {
            /* istanbul ignore if */
            if (typeof document === 'undefined') {
                return [2 /*return*/];
            }
            allObservables = createObservables();
            /* istanbul ignore next */
            logger = config === null || config === void 0 ? void 0 : config.loggerProvider;
            subscription = (0, track_network_event_1.trackNetworkEvents)({
                allObservables: allObservables,
                networkTrackingOptions: options,
                amplitude: amplitude,
                loggerProvider: logger,
            });
            /* istanbul ignore next */
            logger === null || logger === void 0 ? void 0 : logger.log("".concat(name, " has been successfully added."));
            return [2 /*return*/];
        });
    }); };
    /* istanbul ignore next */
    var execute = function (event) { return tslib_1.__awaiter(void 0, void 0, void 0, function () {
        return tslib_1.__generator(this, function (_a) {
            return [2 /*return*/, event];
        });
    }); };
    var teardown = function () { return tslib_1.__awaiter(void 0, void 0, void 0, function () {
        return tslib_1.__generator(this, function (_a) {
            subscription.unsubscribe();
            return [2 /*return*/];
        });
    }); };
    return {
        name: name,
        type: type,
        setup: setup,
        execute: execute,
        teardown: teardown,
    };
};
exports.networkCapturePlugin = networkCapturePlugin;
//# sourceMappingURL=network-capture-plugin.js.map