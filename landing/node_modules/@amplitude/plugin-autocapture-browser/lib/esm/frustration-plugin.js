import { __assign, __awaiter, __generator, __read, __spreadArray, __values } from "tslib";
/* eslint-disable no-restricted-globals */
import { DEFAULT_DATA_ATTRIBUTE_PREFIX, multicast, Observable, DEFAULT_RAGE_CLICK_ALLOWLIST, DEFAULT_DEAD_CLICK_ALLOWLIST, DEFAULT_ERROR_CLICK_ALLOWLIST, } from '@amplitude/analytics-core';
import * as constants from './constants';
import { createShouldTrackEvent } from './helpers';
import { trackDeadClick } from './autocapture/track-dead-click';
import { trackRageClicks } from './autocapture/track-rage-click';
import { ObservablesEnum } from './autocapture-plugin';
import { createClickObservable, createErrorObservable, createMutationObservable, createMouseMoveObservable, } from './observables';
import { DataExtractor } from './data-extractor';
import { trackErrorClicks } from './autocapture/track-error-click';
import { trackThrashedCursor } from './autocapture/track-thrashed-cursor';
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
export var frustrationPlugin = function (options) {
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
    var rageCssSelectors = getCssSelectorAllowlist(options, 'rageClicks', DEFAULT_RAGE_CLICK_ALLOWLIST, rageClicksEnabled);
    var deadCssSelectors = getCssSelectorAllowlist(options, 'deadClicks', DEFAULT_DEAD_CLICK_ALLOWLIST, deadClicksEnabled);
    var errorCssSelectors = getCssSelectorAllowlist(options, 'errorClicks', DEFAULT_ERROR_CLICK_ALLOWLIST, isErrorClicksEnabled);
    var dataAttributePrefix = (_a = options.dataAttributePrefix) !== null && _a !== void 0 ? _a : DEFAULT_DATA_ATTRIBUTE_PREFIX;
    var dataExtractor = new DataExtractor(options);
    // combine the selector lists from enabled features to determine which clicked elements should be filtered
    var combinedCssSelectors = __spreadArray([], __read(new Set(__spreadArray(__spreadArray(__spreadArray([], __read(rageCssSelectors), false), __read(deadCssSelectors), false), __read(errorCssSelectors), false))), false);
    // Create observables on events on the window
    var createObservables = function () {
        var _a;
        var clickObservable = multicast(createClickObservable('pointerdown').map(function (click) {
            return dataExtractor.addAdditionalEventProperties(click, 'click', combinedCssSelectors, dataAttributePrefix, true);
        }));
        var browserErrorObservables = multicast(createErrorObservable().map(function (error) {
            return dataExtractor.addTypeAndTimestamp(error, 'error');
        }));
        var enrichedMutationObservable = multicast(createMutationObservable().map(function (mutation) {
            return dataExtractor.addAdditionalEventProperties(mutation, 'mutation', combinedCssSelectors, dataAttributePrefix);
        }));
        var enrichedNavigateObservable;
        if (window.navigation) {
            var navigateObservable = new Observable(function (observer) {
                var handler = function (event) {
                    observer.next(__assign(__assign({}, event), { type: 'navigate' }));
                };
                window.navigation.addEventListener('navigate', handler);
                return function () {
                    window.navigation.removeEventListener('navigate', handler);
                };
            });
            enrichedNavigateObservable = multicast(navigateObservable.map(function (navigate) {
                return dataExtractor.addAdditionalEventProperties(navigate, 'navigate', combinedCssSelectors, dataAttributePrefix);
            }));
        }
        var selectionObservable = multicast(new Observable(function (observer) {
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
        var mouseMoveObservable = multicast(createMouseMoveObservable());
        return _a = {},
            _a[ObservablesEnum.ClickObservable] = clickObservable,
            _a[ObservablesEnum.MutationObservable] = enrichedMutationObservable,
            _a[ObservablesEnum.NavigateObservable] = enrichedNavigateObservable,
            _a[ObservablesEnum.BrowserErrorObservable] = browserErrorObservables,
            _a[ObservablesEnum.SelectionObservable] = selectionObservable,
            _a[ObservablesEnum.MouseMoveObservable] = mouseMoveObservable,
            _a;
    };
    var setup = function (config, amplitude) { return __awaiter(void 0, void 0, void 0, function () {
        var allObservables, shouldTrackRageClick, rageClickSubscription, shouldTrackDeadClick, deadClickSubscription, shouldTrackErrorClick, errorClickSubscription, directionChanges, thresholdMs, thrashedCursorSubscription;
        var _a;
        return __generator(this, function (_b) {
            /* istanbul ignore if */
            if (typeof document === 'undefined') {
                return [2 /*return*/];
            }
            allObservables = createObservables();
            // Create subscriptions only for enabled features
            if (rageClicksEnabled) {
                shouldTrackRageClick = createShouldTrackEvent(options, rageCssSelectors);
                rageClickSubscription = trackRageClicks({
                    allObservables: allObservables,
                    amplitude: amplitude,
                    shouldTrackRageClick: shouldTrackRageClick,
                });
                subscriptions.push(rageClickSubscription);
            }
            if (deadClicksEnabled) {
                shouldTrackDeadClick = createShouldTrackEvent(options, deadCssSelectors);
                deadClickSubscription = trackDeadClick({
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
                shouldTrackErrorClick = createShouldTrackEvent(options, errorCssSelectors);
                errorClickSubscription = trackErrorClicks({
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
                thrashedCursorSubscription = trackThrashedCursor({
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
    var execute = function (event) { return __awaiter(void 0, void 0, void 0, function () {
        return __generator(this, function (_a) {
            return [2 /*return*/, event];
        });
    }); };
    var teardown = function () { return __awaiter(void 0, void 0, void 0, function () {
        var subscriptions_1, subscriptions_1_1, subscription;
        var e_1, _a;
        return __generator(this, function (_b) {
            try {
                for (subscriptions_1 = __values(subscriptions), subscriptions_1_1 = subscriptions_1.next(); !subscriptions_1_1.done; subscriptions_1_1 = subscriptions_1.next()) {
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
//# sourceMappingURL=frustration-plugin.js.map