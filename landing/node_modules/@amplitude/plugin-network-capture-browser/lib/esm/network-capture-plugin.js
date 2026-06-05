import { __awaiter, __generator } from "tslib";
/* eslint-disable no-restricted-globals */
import { networkObserver, NetworkEventCallback, } from '@amplitude/analytics-core';
import * as constants from './constants';
import { Observable } from '@amplitude/analytics-core';
import { trackNetworkEvents } from './track-network-event';
export var ObservablesEnum;
(function (ObservablesEnum) {
    ObservablesEnum["NetworkObservable"] = "networkObservable";
})(ObservablesEnum || (ObservablesEnum = {}));
var subscription;
export var networkCapturePlugin = function (options) {
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
        var networkObservable = new Observable(function (observer) {
            var callback = new NetworkEventCallback(function (event) {
                var eventWithProperties = addAdditionalEventProperties(event, 'network');
                observer.next(eventWithProperties);
            });
            networkObserver.subscribe(callback, logger);
            return function () {
                networkObserver.unsubscribe(callback);
            };
        });
        return _a = {},
            _a[ObservablesEnum.NetworkObservable] = networkObservable,
            _a;
    };
    var setup = function (config, amplitude) { return __awaiter(void 0, void 0, void 0, function () {
        var allObservables;
        return __generator(this, function (_a) {
            /* istanbul ignore if */
            if (typeof document === 'undefined') {
                return [2 /*return*/];
            }
            allObservables = createObservables();
            /* istanbul ignore next */
            logger = config === null || config === void 0 ? void 0 : config.loggerProvider;
            subscription = trackNetworkEvents({
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
    var execute = function (event) { return __awaiter(void 0, void 0, void 0, function () {
        return __generator(this, function (_a) {
            return [2 /*return*/, event];
        });
    }); };
    var teardown = function () { return __awaiter(void 0, void 0, void 0, function () {
        return __generator(this, function (_a) {
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
//# sourceMappingURL=network-capture-plugin.js.map