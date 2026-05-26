"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.frustrationPlugin = void 0;
var tslib_1 = require("tslib");
/* eslint-disable no-restricted-globals */
var analytics_core_1 = require("@amplitude/analytics-core");
var constants = tslib_1.__importStar(require("./constants"));
var helpers_1 = require("./helpers");
var track_dead_click_1 = require("./autocapture/track-dead-click");
var track_rage_click_1 = require("./autocapture/track-rage-click");
var autocapture_plugin_1 = require("./autocapture-plugin");
var observables_1 = require("./observables");
var data_extractor_1 = require("./data-extractor");
var track_error_click_1 = require("./autocapture/track-error-click");
var track_thrashed_cursor_1 = require("./autocapture/track-thrashed-cursor");
/**
 * Helper function to extract the css selector allowlist
 * from the frustration interactions options for a specific
 * autocapture feature.
 */
function getCssSelectorAllowlist(options, attribute, defaultAllowlist, enabled) {
    if (!enabled) {
        return [];
    }
    var config = options[attribute];
    if (typeof config === 'object' &&
        config !== null &&
        'cssSelectorAllowlist' in config &&
        Array.isArray(config.cssSelectorAllowlist)) {
        return config.cssSelectorAllowlist;
    }
    return defaultAllowlist;
}
var MINIMUM_THRASHED_CURSOR_DIRECTION_CHANGES = 5;
var MAXIMUM_THRASHED_CURSOR_THRESHOLD = 4000;
var frustrationPlugin = function (options) {
    var _a;
    if (options === void 0) { options = {}; }
    var name = constants.FRUSTRATION_PLUGIN_NAME;
    var type = 'enrichment';
    var subscriptions = [];
    var isErrorClicksEnabled = options.errorClicks !== false;
    // if errorClicks is not defined, disable it
    // change this once it moves out of @experimental
    if (!options.errorClicks) {
        isErrorClicksEnabled = false;
    }
    // Check if each feature is enabled
    var deadClicksEnabled = options.deadClicks !== false && options.deadClicks !== null;
    var rageClicksEnabled = options.rageClicks !== false && options.rageClicks !== null;
    var thrashedCursorEnabled = options.thrashedCursor !== false && options.thrashedCursor !== null;
    if (!options.thrashedCursor) {
        thrashedCursorEnabled = false;
    }
    // Get CSS selectors for enabled features
    var rageCssSelectors = getCssSelectorAllowlist(options, 'rageClicks', analytics_core_1.DEFAULT_RAGE_CLICK_ALLOWLIST, rageClicksEnabled);
    var deadCssSelectors = getCssSelectorAllowlist(options, 'deadClicks', analytics_core_1.DEFAULT_DEAD_CLICK_ALLOWLIST, deadClicksEnabled);
    var errorCssSelectors = getCssSelectorAllowlist(options, 'errorClicks', analytics_core_1.DEFAULT_ERROR_CLICK_ALLOWLIST, isErrorClicksEnabled);
    var dataAttributePrefix = (_a = options.dataAttributePrefix) !== null && _a !== void 0 ? _a : analytics_core_1.DEFAULT_DATA_ATTRIBUTE_PREFIX;
    var dataExtractor = new data_extractor_1.DataExtractor(options);
    // combine the selector lists from enabled features to determine which clicked elements should be filtered
    var combinedCssSelectors = tslib_1.__spreadArray([], tslib_1.__read(new Set(tslib_1.__spreadArray(tslib_1.__spreadArray(tslib_1.__spreadArray([], tslib_1.__read(rageCssSelectors), false), tslib_1.__read(deadCssSelectors), false), tslib_1.__read(errorCssSelectors), false))), false);
    // Create observables on events on the window
    var createObservables = function () {
        var _a;
        var clickObservable = (0, analytics_core_1.multicast)((0, observables_1.createClickObservable)('pointerdown').map(function (click) {
            return dataExtractor.addAdditionalEventProperties(click, 'click', combinedCssSelectors, dataAttributePrefix, true);
        }));
        var browserErrorObservables = (0, analytics_core_1.multicast)((0, observables_1.createErrorObservable)().map(function (error) {
            return dataExtractor.addTypeAndTimestamp(error, 'error');
        }));
        var enrichedMutationObservable = (0, analytics_core_1.multicast)((0, observables_1.createMutationObservable)().map(function (mutation) {
            return dataExtractor.addAdditionalEventProperties(mutation, 'mutation', combinedCssSelectors, dataAttributePrefix);
        }));
        var enrichedNavigateObservable;
        if (window.navigation) {
            var navigateObservable = new analytics_core_1.Observable(function (observer) {
                var handler = function (event) {
                    observer.next(tslib_1.__assign(tslib_1.__assign({}, event), { type: 'navigate' }));
                };
                window.navigation.addEventListener('navigate', handler);
                return function () {
                    window.navigation.removeEventListener('navigate', handler);
                };
            });
            enrichedNavigateObservable = (0, analytics_core_1.multicast)(navigateObservable.map(function (navigate) {
                return dataExtractor.addAdditionalEventProperties(navigate, 'navigate', combinedCssSelectors, dataAttributePrefix);
            }));
        }
        var selectionObservable = (0, analytics_core_1.multicast)(new analytics_core_1.Observable(function (observer) {
            var handler = function () {
                var el = document.activeElement;
                // handle input and textarea
                // if the selectionStart and selectionEnd are the same, it means
                // nothing is selected (collapsed) and the cursor position is one point
                if (el && (el.tagName === 'TEXTAREA' || el.tagName === 'INPUT')) {
                    var start = void 0;
                    var end = void 0;
                    try {
                        start = el.selectionStart;
                        end = el.selectionEnd;
                        if (start === end)
                            return; // collapsed
                    }
                    catch (error) {
                        // input that doesn't support selectionStart/selectionEnd (like checkbox)
                        // do nothing here
                        return;
                    }
                    return observer.next();
                }
                // handle non-input elements
                // non-input elements have an attribute called "isCollapsed" which
                // if true, indicates there "is currently not any text selected"
                // (see https://developer.mozilla.org/en-US/docs/Web/API/Selection/isCollapsed)
                var selection = window.getSelection();
                if (!selection || selection.isCollapsed)
                    return;
                return observer.next();
            };
            window.document.addEventListener('selectionchange', handler);
            return function () {
                window.document.removeEventListener('selectionchange', handler);
            };
        }));
        var mouseMoveObservable = (0, analytics_core_1.multicast)((0, observables_1.createMouseMoveObservable)());
        return _a = {},
            _a[autocapture_plugin_1.ObservablesEnum.ClickObservable] = clickObservable,
            _a[autocapture_plugin_1.ObservablesEnum.MutationObservable] = enrichedMutationObservable,
            _a[autocapture_plugin_1.ObservablesEnum.NavigateObservable] = enrichedNavigateObservable,
            _a[autocapture_plugin_1.ObservablesEnum.BrowserErrorObservable] = browserErrorObservables,
            _a[autocapture_plugin_1.ObservablesEnum.SelectionObservable] = selectionObservable,
            _a[autocapture_plugin_1.ObservablesEnum.MouseMoveObservable] = mouseMoveObservable,
            _a;
    };
    var setup = function (config, amplitude) { return tslib_1.__awaiter(void 0, void 0, void 0, function () {
        var allObservables, shouldTrackRageClick, rageClickSubscription, shouldTrackDeadClick, deadClickSubscription, shouldTrackErrorClick, errorClickSubscription, directionChanges, thresholdMs, thrashedCursorSubscription;
        var _a;
        return tslib_1.__generator(this, function (_b) {
            /* istanbul ignore if */
            if (typeof document === 'undefined') {
                return [2 /*return*/];
            }
            allObservables = createObservables();
            // Create subscriptions only for enabled features
            if (rageClicksEnabled) {
                shouldTrackRageClick = (0, helpers_1.createShouldTrackEvent)(options, rageCssSelectors);
                rageClickSubscription = (0, track_rage_click_1.trackRageClicks)({
                    allObservables: allObservables,
                    amplitude: amplitude,
                    shouldTrackRageClick: shouldTrackRageClick,
                });
                subscriptions.push(rageClickSubscription);
            }
            if (deadClicksEnabled) {
                shouldTrackDeadClick = (0, helpers_1.createShouldTrackEvent)(options, deadCssSelectors);
                deadClickSubscription = (0, track_dead_click_1.trackDeadClick)({
                    amplitude: amplitude,
                    allObservables: allObservables,
                    getEventProperties: function (actionType, element) {
                        return dataExtractor.getEventProperties(actionType, element, dataAttributePrefix);
                    },
                    shouldTrackDeadClick: shouldTrackDeadClick,
                });
                subscriptions.push(deadClickSubscription);
            }
            if (isErrorClicksEnabled) {
                shouldTrackErrorClick = (0, helpers_1.createShouldTrackEvent)(options, errorCssSelectors);
                errorClickSubscription = (0, track_error_click_1.trackErrorClicks)({
                    amplitude: amplitude,
                    allObservables: allObservables,
                    shouldTrackErrorClick: shouldTrackErrorClick,
                });
                subscriptions.push(errorClickSubscription);
            }
            if (thrashedCursorEnabled) {
                directionChanges = void 0, thresholdMs = void 0;
                if (typeof options.thrashedCursor === 'object') {
                    directionChanges = options.thrashedCursor.directionChanges;
                    thresholdMs = options.thrashedCursor.threshold;
                    if (directionChanges && directionChanges < MINIMUM_THRASHED_CURSOR_DIRECTION_CHANGES) {
                        config.loggerProvider.warn("'thrashedCursor.directionChanges' of ".concat(directionChanges, " is below the minimum of ").concat(MINIMUM_THRASHED_CURSOR_DIRECTION_CHANGES, ", setting to ").concat(MINIMUM_THRASHED_CURSOR_DIRECTION_CHANGES));
                        directionChanges = MINIMUM_THRASHED_CURSOR_DIRECTION_CHANGES;
                    }
                    if (thresholdMs && thresholdMs > MAXIMUM_THRASHED_CURSOR_THRESHOLD) {
                        config.loggerProvider.warn("'thrashedCursor.threshold' of ".concat(thresholdMs, " is above the maximum of ").concat(MAXIMUM_THRASHED_CURSOR_THRESHOLD, ", setting to ").concat(MAXIMUM_THRASHED_CURSOR_THRESHOLD));
                        thresholdMs = MAXIMUM_THRASHED_CURSOR_THRESHOLD;
                    }
                }
                thrashedCursorSubscription = (0, track_thrashed_cursor_1.trackThrashedCursor)({
                    amplitude: amplitude,
                    options: options,
                    allObservables: allObservables,
                    directionChanges: directionChanges,
                    thresholdMs: thresholdMs,
                });
                subscriptions.push(thrashedCursorSubscription);
            }
            /* istanbul ignore next */
            (_a = config === null || config === void 0 ? void 0 : config.loggerProvider) === null || _a === void 0 ? void 0 : _a.log("".concat(name, " has been successfully added."));
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
exports.frustrationPlugin = frustrationPlugin;
//# sourceMappingURL=frustration-plugin.js.map