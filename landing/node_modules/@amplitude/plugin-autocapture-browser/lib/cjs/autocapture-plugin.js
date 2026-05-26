"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.autocapturePlugin = exports.ObservablesEnum = void 0;
var tslib_1 = require("tslib");
/* eslint-disable no-restricted-globals */
var analytics_core_1 = require("@amplitude/analytics-core");
var version_1 = require("./version");
var constants = tslib_1.__importStar(require("./constants"));
var helpers_1 = require("./helpers");
var messenger_1 = require("./libs/messenger");
var track_click_1 = require("./autocapture/track-click");
var track_change_1 = require("./autocapture/track-change");
var track_action_click_1 = require("./autocapture/track-action-click");
var track_scroll_1 = require("./autocapture/track-scroll");
var observables_1 = require("./observables");
var triggers_1 = require("./pageActions/triggers");
var data_extractor_1 = require("./data-extractor");
var analytics_core_2 = require("@amplitude/analytics-core");
var track_exposure_1 = require("./autocapture/track-exposure");
var track_viewport_content_updated_1 = require("./autocapture/track-viewport-content-updated");
var ObservablesEnum;
(function (ObservablesEnum) {
    ObservablesEnum["ClickObservable"] = "clickObservable";
    ObservablesEnum["ChangeObservable"] = "changeObservable";
    ObservablesEnum["NavigateObservable"] = "navigateObservable";
    ObservablesEnum["MutationObservable"] = "mutationObservable";
    ObservablesEnum["ScrollObservable"] = "scrollObservable";
    ObservablesEnum["ExposureObservable"] = "exposureObservable";
    ObservablesEnum["BrowserErrorObservable"] = "browserErrorObservable";
    ObservablesEnum["SelectionObservable"] = "selectionObservable";
    ObservablesEnum["MouseMoveObservable"] = "mouseMoveObservable";
})(ObservablesEnum = exports.ObservablesEnum || (exports.ObservablesEnum = {}));
var autocapturePlugin = function (options, context) {
    var _a, _b, _c, _d, _e, _f, _g, _h, _j, _k, _l, _m;
    if (options === void 0) { options = {}; }
    // Set the plugin version tag
    // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
    context === null || context === void 0 ? void 0 : context.diagnosticsClient.setTag('plugin.autocapture.version', version_1.VERSION);
    var _o = options.dataAttributePrefix, dataAttributePrefix = _o === void 0 ? analytics_core_1.DEFAULT_DATA_ATTRIBUTE_PREFIX : _o, _p = options.visualTaggingOptions, visualTaggingOptions = _p === void 0 ? {
        enabled: true,
    } : _p;
    options.cssSelectorAllowlist = (_a = options.cssSelectorAllowlist) !== null && _a !== void 0 ? _a : analytics_core_1.DEFAULT_CSS_SELECTOR_ALLOWLIST;
    options.actionClickAllowlist = (_b = options.actionClickAllowlist) !== null && _b !== void 0 ? _b : analytics_core_1.DEFAULT_ACTION_CLICK_ALLOWLIST;
    options.debounceTime = (_c = options.debounceTime) !== null && _c !== void 0 ? _c : 0;
    var isViewportContentUpdatedEnabled = ((_d = options.viewportContentUpdated) === null || _d === void 0 ? void 0 : _d.enabled) !== false;
    var resolvedExposureDuration = (_g = (_f = (_e = options.viewportContentUpdated) === null || _e === void 0 ? void 0 : _e.exposureDuration) !== null && _f !== void 0 ? _f : options.exposureDuration) !== null && _g !== void 0 ? _g : analytics_core_1.DEFAULT_EXPOSURE_DURATION;
    options.viewportContentUpdated = tslib_1.__assign(tslib_1.__assign({}, options.viewportContentUpdated), { exposureDuration: resolvedExposureDuration });
    options.pageUrlExcludelist = (_h = options.pageUrlExcludelist) === null || _h === void 0 ? void 0 : _h.reduce(function (acc, excludePattern) {
        if (typeof excludePattern === 'string') {
            acc.push(excludePattern);
        }
        if (excludePattern instanceof RegExp) {
            acc.push(excludePattern);
        }
        if (typeof excludePattern === 'object' && excludePattern !== null && 'pattern' in excludePattern) {
            try {
                acc.push(new RegExp(excludePattern.pattern));
            }
            catch (regexError) {
                console.warn("Invalid regex pattern: ".concat(excludePattern.pattern), regexError);
                return acc;
            }
        }
        return acc;
    }, []);
    var name = constants.PLUGIN_NAME;
    var type = 'enrichment';
    var subscriptions = [];
    var dataExtractor = new data_extractor_1.DataExtractor(options, context);
    // Page-level state shared across trackers, emitted in a single Page View End event on beforeunload
    // elementExposedForPage holds the total set of elements seen during the entire page view lifetime
    var elementExposedForPage = new Set();
    // currentElementExposed only holds the set of elements that will be flushed during the next [Amplitude] Viewport Content Updated event
    var currentElementExposed = new Set();
    var beforeUnloadCleanup;
    var createObservables = function () {
        var _a;
        var clickObservable = (0, analytics_core_1.multicast)((0, observables_1.createClickObservable)().map(function (click) {
            return dataExtractor.addAdditionalEventProperties(click, 'click', options.cssSelectorAllowlist, dataAttributePrefix);
        }));
        var changeObservable = (0, analytics_core_1.multicast)(new analytics_core_2.Observable(function (observer) {
            var _a;
            var handler = function (changeEvent) {
                var enrichedChangeEvent = dataExtractor.addAdditionalEventProperties(changeEvent, 'change', options.cssSelectorAllowlist, dataAttributePrefix);
                observer.next(enrichedChangeEvent);
            };
            /* istanbul ignore next */
            (_a = (0, analytics_core_1.getGlobalScope)()) === null || _a === void 0 ? void 0 : _a.document.addEventListener('change', handler, { capture: true });
            /* istanbul ignore next */
            return function () { var _a; return (_a = (0, analytics_core_1.getGlobalScope)()) === null || _a === void 0 ? void 0 : _a.document.removeEventListener('change', handler); };
        }));
        // Create observable for URL changes
        var navigateObservable;
        /* istanbul ignore next */
        if (window.navigation) {
            navigateObservable = (0, analytics_core_1.multicast)(new analytics_core_2.Observable(function (observer) {
                var handler = function (navigateEvent) {
                    var enrichedNavigateEvent = dataExtractor.addAdditionalEventProperties(navigateEvent, 'navigate', options.cssSelectorAllowlist, dataAttributePrefix);
                    observer.next(enrichedNavigateEvent);
                };
                window.navigation.addEventListener('navigate', handler);
                return function () {
                    window.navigation.removeEventListener('navigate', handler);
                };
            }));
        }
        var mutationObservable = (0, analytics_core_1.multicast)((0, observables_1.createMutationObservable)().map(function (mutation) {
            return dataExtractor.addAdditionalEventProperties(mutation, 'mutation', options.cssSelectorAllowlist, dataAttributePrefix);
        }));
        var scrollObservable = (0, observables_1.createScrollObservable)();
        var exposureObservable = (0, observables_1.createExposureObservable)(mutationObservable, options.cssSelectorAllowlist);
        return _a = {},
            _a[ObservablesEnum.ChangeObservable] = changeObservable,
            // [ObservablesEnum.ErrorObservable]: errorObservable,
            _a[ObservablesEnum.ClickObservable] = clickObservable,
            _a[ObservablesEnum.MutationObservable] = mutationObservable,
            _a[ObservablesEnum.NavigateObservable] = navigateObservable,
            _a[ObservablesEnum.ScrollObservable] = scrollObservable,
            _a[ObservablesEnum.ExposureObservable] = exposureObservable,
            _a;
    };
    // Group labeled events by event type (eg. click, change)
    var groupedLabeledEvents = (0, triggers_1.groupLabeledEventIdsByEventType)(Object.values((_k = (_j = options.pageActions) === null || _j === void 0 ? void 0 : _j.labeledEvents) !== null && _k !== void 0 ? _k : {}));
    var labeledEventToTriggerMap = (0, triggers_1.createLabeledEventToTriggerMap)((_m = (_l = options.pageActions) === null || _l === void 0 ? void 0 : _l.triggers) !== null && _m !== void 0 ? _m : []);
    // Evaluate triggers for the given event by running the actions associated with the matching triggers
    var evaluateTriggers = (0, triggers_1.createTriggerEvaluator)(groupedLabeledEvents, labeledEventToTriggerMap, dataExtractor, options);
    // Function to recalculate internal variables when remote config is updated
    var recomputePageActionsData = function (remotePageActions) {
        var _a, _b;
        if (remotePageActions) {
            // Merge remote config with local options
            options.pageActions = tslib_1.__assign(tslib_1.__assign({}, options.pageActions), remotePageActions);
            // Recalculate internal variables
            groupedLabeledEvents = (0, triggers_1.groupLabeledEventIdsByEventType)(Object.values((_a = options.pageActions.labeledEvents) !== null && _a !== void 0 ? _a : {}));
            labeledEventToTriggerMap = (0, triggers_1.createLabeledEventToTriggerMap)((_b = options.pageActions.triggers) !== null && _b !== void 0 ? _b : []);
            // Update evaluateTriggers function
            evaluateTriggers.update(groupedLabeledEvents, labeledEventToTriggerMap, options);
        }
    };
    var setup = function (config, amplitude) { return tslib_1.__awaiter(void 0, void 0, void 0, function () {
        var pageViewEndFired, lastScroll, shouldTrackEvent, shouldTrackActionClick, allObservables, clickTrackingSubscription, changeSubscription, actionClickSubscription, scrollTracker, trackers, globalScope, handleViewportContentUpdated, handleExposure, beforeUnloadHandler_1, navigateObservable, popstateHandler_1, originalPushState_1, allowlist, actionClickAllowlist, messenger;
        var _a;
        return tslib_1.__generator(this, function (_b) {
            /* istanbul ignore if */
            if (typeof document === 'undefined') {
                return [2 /*return*/];
            }
            pageViewEndFired = false;
            lastScroll = { maxX: undefined, maxY: undefined };
            // Fetch remote config for pageActions in a non-blocking manner
            if (config.fetchRemoteConfig) {
                if (!config.remoteConfigClient) {
                    // TODO(xinyi): Diagnostics.recordEvent
                    config.loggerProvider.debug('Remote config client is not provided, skipping remote config fetch');
                }
                else {
                    config.remoteConfigClient.subscribe('configs.analyticsSDK.pageActions', 'all', function (remoteConfig) {
                        recomputePageActionsData(remoteConfig);
                    });
                }
            }
            shouldTrackEvent = (0, helpers_1.createShouldTrackEvent)(options, options.cssSelectorAllowlist);
            shouldTrackActionClick = (0, helpers_1.createShouldTrackEvent)(options, options.actionClickAllowlist);
            allObservables = createObservables();
            clickTrackingSubscription = (0, track_click_1.trackClicks)({
                allObservables: allObservables,
                amplitude: amplitude,
                shouldTrackEvent: shouldTrackEvent,
                evaluateTriggers: evaluateTriggers.evaluate.bind(evaluateTriggers),
            });
            subscriptions.push(clickTrackingSubscription);
            changeSubscription = (0, track_change_1.trackChange)({
                allObservables: allObservables,
                getEventProperties: function () {
                    var args = [];
                    for (var _i = 0; _i < arguments.length; _i++) {
                        args[_i] = arguments[_i];
                    }
                    return dataExtractor.getEventProperties.apply(dataExtractor, tslib_1.__spreadArray(tslib_1.__spreadArray([], tslib_1.__read(args), false), [dataAttributePrefix], false));
                },
                amplitude: amplitude,
                shouldTrackEvent: shouldTrackEvent,
                evaluateTriggers: evaluateTriggers.evaluate.bind(evaluateTriggers),
            });
            subscriptions.push(changeSubscription);
            actionClickSubscription = (0, track_action_click_1.trackActionClick)({
                allObservables: allObservables,
                options: options,
                getEventProperties: function () {
                    var args = [];
                    for (var _i = 0; _i < arguments.length; _i++) {
                        args[_i] = arguments[_i];
                    }
                    return dataExtractor.getEventProperties.apply(dataExtractor, tslib_1.__spreadArray(tslib_1.__spreadArray([], tslib_1.__read(args), false), [dataAttributePrefix], false));
                },
                amplitude: amplitude,
                shouldTrackEvent: shouldTrackEvent,
                shouldTrackActionClick: shouldTrackActionClick,
            });
            if (actionClickSubscription) {
                subscriptions.push(actionClickSubscription);
            }
            scrollTracker = (0, track_scroll_1.trackScroll)({
                allObservables: allObservables,
                amplitude: amplitude,
            });
            subscriptions.push(scrollTracker);
            trackers = {};
            globalScope = (0, analytics_core_1.getGlobalScope)();
            handleViewportContentUpdated = function (isPageEnd) {
                if (isPageEnd && pageViewEndFired) {
                    return;
                }
                setTimeout(function () {
                    pageViewEndFired = false;
                }, 100);
                pageViewEndFired = true;
                (0, track_viewport_content_updated_1.fireViewportContentUpdated)({
                    amplitude: amplitude,
                    scrollTracker: scrollTracker,
                    currentElementExposed: currentElementExposed,
                    elementExposedForPage: elementExposedForPage,
                    exposureTracker: trackers.exposure,
                    isPageEnd: isPageEnd,
                    lastScroll: lastScroll,
                });
            };
            handleExposure = function (elementPath) {
                (0, track_viewport_content_updated_1.onExposure)(elementPath, elementExposedForPage, currentElementExposed, handleViewportContentUpdated);
            };
            if (isViewportContentUpdatedEnabled) {
                trackers.exposure = (0, track_exposure_1.trackExposure)({
                    allObservables: allObservables,
                    onExposure: handleExposure,
                    dataExtractor: dataExtractor,
                    exposureDuration: resolvedExposureDuration,
                });
                if (trackers.exposure) {
                    subscriptions.push(trackers.exposure);
                }
                beforeUnloadHandler_1 = function () {
                    handleViewportContentUpdated(true);
                };
                /* istanbul ignore next */
                globalScope === null || globalScope === void 0 ? void 0 : globalScope.addEventListener('beforeunload', beforeUnloadHandler_1);
                beforeUnloadCleanup = function () {
                    /* istanbul ignore next */
                    globalScope === null || globalScope === void 0 ? void 0 : globalScope.removeEventListener('beforeunload', beforeUnloadHandler_1);
                };
                // Ensure cleanup on teardown as well
                subscriptions.push({ unsubscribe: function () { return beforeUnloadCleanup(); } });
                navigateObservable = allObservables[ObservablesEnum.NavigateObservable];
                if (navigateObservable) {
                    subscriptions.push(navigateObservable.subscribe(function () {
                        handleViewportContentUpdated(true);
                    }));
                }
                else if (globalScope) {
                    popstateHandler_1 = function () {
                        handleViewportContentUpdated(true);
                    };
                    /* istanbul ignore next */
                    // Fallback for SPA tracking when Navigation API is not available
                    globalScope.addEventListener('popstate', popstateHandler_1);
                    originalPushState_1 = globalScope.history.pushState;
                    if (globalScope.history && originalPushState_1) {
                        // eslint-disable-next-line @typescript-eslint/unbound-method
                        globalScope.history.pushState = new Proxy(originalPushState_1, {
                            apply: function (target, thisArg, _a) {
                                var _b = tslib_1.__read(_a, 3), state = _b[0], unused = _b[1], url = _b[2];
                                target.apply(thisArg, [state, unused, url]);
                                handleViewportContentUpdated(true);
                            },
                        });
                    }
                    subscriptions.push({
                        unsubscribe: function () {
                            /* istanbul ignore next */
                            globalScope.removeEventListener('popstate', popstateHandler_1);
                            /* istanbul ignore next */
                            if (globalScope.history && originalPushState_1) {
                                globalScope.history.pushState = originalPushState_1;
                            }
                        },
                    });
                }
            }
            /* istanbul ignore next */
            (_a = config === null || config === void 0 ? void 0 : config.loggerProvider) === null || _a === void 0 ? void 0 : _a.log("".concat(name, " has been successfully added."));
            // Setup visual tagging and background capture on the shared messenger singleton.
            // Using the singleton ensures a single message listener per page, even when
            // session-replay is also loaded.
            if (window.opener && visualTaggingOptions.enabled) {
                allowlist = options.cssSelectorAllowlist;
                actionClickAllowlist = options.actionClickAllowlist;
                messenger = (0, analytics_core_1.getOrCreateWindowMessenger)();
                (0, messenger_1.enableVisualTagging)(messenger, {
                    dataExtractor: dataExtractor,
                    isElementSelectable: (0, helpers_1.createShouldTrackEvent)(options, tslib_1.__spreadArray(tslib_1.__spreadArray([], tslib_1.__read(allowlist), false), tslib_1.__read(actionClickAllowlist), false)),
                    cssSelectorAllowlist: allowlist,
                    actionClickAllowlist: actionClickAllowlist,
                });
                (0, analytics_core_1.enableBackgroundCapture)(messenger);
                /* istanbul ignore next */
                messenger.setup(tslib_1.__assign({ logger: config === null || config === void 0 ? void 0 : config.loggerProvider }, ((config === null || config === void 0 ? void 0 : config.serverZone) && { endpoint: constants.AMPLITUDE_ORIGINS_MAP[config.serverZone] })));
            }
            return [2 /*return*/];
        });
    }); };
    var execute = function (event) { return tslib_1.__awaiter(void 0, void 0, void 0, function () {
        return tslib_1.__generator(this, function (_a) {
            return [2 /*return*/, event];
        });
    }); };
    var teardown = function () { return tslib_1.__awaiter(void 0, void 0, void 0, function () {
        var subscriptions_1, subscriptions_1_1, subscription;
        var e_1, _a;
        return tslib_1.__generator(this, function (_b) {
            try {
                for (subscriptions_1 = tslib_1.__values(subscriptions), subscriptions_1_1 = subscriptions_1.next(); !subscriptions_1_1.done; subscriptions_1_1 = subscriptions_1.next()) {
                    subscription = subscriptions_1_1.value;
                    subscription.unsubscribe();
                }
            }
            catch (e_1_1) { e_1 = { error: e_1_1 }; }
            finally {
                try {
                    if (subscriptions_1_1 && !subscriptions_1_1.done && (_a = subscriptions_1.return)) _a.call(subscriptions_1);
                }
                finally { if (e_1) throw e_1.error; }
            }
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
exports.autocapturePlugin = autocapturePlugin;
//# sourceMappingURL=autocapture-plugin.js.map