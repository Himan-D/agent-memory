import { __assign, __awaiter, __generator } from "tslib";
import { asyncMap, DEFAULT_RAGE_CLICK_THRESHOLD, DEFAULT_RAGE_CLICK_WINDOW_MS, DEFAULT_RAGE_CLICK_OUT_OF_BOUNDS_THRESHOLD, } from '@amplitude/analytics-core';
import { AMPLITUDE_ELEMENT_RAGE_CLICKED_EVENT } from '../constants';
var RAGE_CLICK_THRESHOLD = DEFAULT_RAGE_CLICK_THRESHOLD;
var RAGE_CLICK_WINDOW_MS = DEFAULT_RAGE_CLICK_WINDOW_MS;
var RAGE_CLICK_OUT_OF_BOUNDS_THRESHOLD = DEFAULT_RAGE_CLICK_OUT_OF_BOUNDS_THRESHOLD;
function addCoordinates(regionBox, click) {
    var _a, _b, _c, _d;
    var _e = click.event, pageX = _e.pageX, pageY = _e.pageY;
    regionBox.yMin = Math.min((_a = regionBox.yMin) !== null && _a !== void 0 ? _a : pageY, pageY);
    regionBox.yMax = Math.max((_b = regionBox.yMax) !== null && _b !== void 0 ? _b : pageY, pageY);
    regionBox.xMin = Math.min((_c = regionBox.xMin) !== null && _c !== void 0 ? _c : pageX, pageX);
    regionBox.xMax = Math.max((_d = regionBox.xMax) !== null && _d !== void 0 ? _d : pageX, pageX);
    regionBox.isOutOfBounds =
        regionBox.yMax - regionBox.yMin > RAGE_CLICK_OUT_OF_BOUNDS_THRESHOLD ||
            regionBox.xMax - regionBox.xMin > RAGE_CLICK_OUT_OF_BOUNDS_THRESHOLD;
}
function getRageClickAnalyticsEvent(clickWindow) {
    /* istanbul ignore if */
    if (clickWindow.length === 0) {
        return null;
    }
    var firstClick = clickWindow[0];
    var lastClick = clickWindow[clickWindow.length - 1];
    var rageClickEvent = __assign({ '[Amplitude] Begin Time': new Date(firstClick.timestamp).toISOString(), '[Amplitude] End Time': new Date(lastClick.timestamp).toISOString(), '[Amplitude] Duration': lastClick.timestamp - firstClick.timestamp, '[Amplitude] Clicks': clickWindow.map(function (click) { return ({
            X: click.event.pageX,
            Y: click.event.pageY,
            Time: click.timestamp,
        }); }), '[Amplitude] Click Count': clickWindow.length }, firstClick.targetElementProperties);
    return { rageClickEvent: rageClickEvent, time: firstClick.timestamp };
}
function isClickOutsideRageClickWindow(clickWindow, click) {
    var firstIndex = Math.max(0, clickWindow.length - RAGE_CLICK_THRESHOLD + 1);
    var firstClick = clickWindow[firstIndex];
    return click.timestamp - firstClick.timestamp >= RAGE_CLICK_WINDOW_MS;
}
function isNewElement(clickWindow, click) {
    return (clickWindow.length > 0 &&
        clickWindow[clickWindow.length - 1].closestTrackedAncestor !== click.closestTrackedAncestor);
}
export function trackRageClicks(_a) {
    var _this = this;
    var amplitude = _a.amplitude, allObservables = _a.allObservables, shouldTrackRageClick = _a.shouldTrackRageClick;
    var clickObservable = allObservables.clickObservable, selectionObservable = allObservables.selectionObservable;
    // Keep track of all clicks within the sliding window
    var clickWindow = [];
    // Keep track of the region box for all clicks, to determine when a rage click is out of bounds
    var clickBoundingBox = {};
    var pendingRageClick = null;
    // helper function to reset the click window and region box
    function resetClickWindow(click) {
        clickWindow = [];
        clickBoundingBox = {};
        if (click) {
            addCoordinates(clickBoundingBox, click);
            clickWindow.push(click);
        }
    }
    var rageClickObservable = asyncMap(clickObservable.filter(function (click) { return shouldTrackRageClick('click', click.closestTrackedAncestor); }), function (click) { return __awaiter(_this, void 0, void 0, function () {
        var resolutionValue;
        return __generator(this, function (_a) {
            // add this click's coordinates to the bounding box
            addCoordinates(clickBoundingBox, click);
            resolutionValue = null;
            // if current click is:
            //  1. first click in the window
            //  2. on a new element
            //  3. outside the rage click time window
            //  4. out of bounds
            // then start a new click window
            if (clickWindow.length === 0 ||
                isNewElement(clickWindow, click) ||
                isClickOutsideRageClickWindow(clickWindow, click) ||
                clickBoundingBox.isOutOfBounds) {
                // if there was a previous Rage Click Event on deck, then send it
                if (pendingRageClick) {
                    resolutionValue = getRageClickAnalyticsEvent(clickWindow);
                }
                resetClickWindow(click);
            }
            else {
                clickWindow.push(click);
            }
            // if there was a previous Rage Click Event on deck, then resolve it
            if (pendingRageClick) {
                // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
                clearTimeout(pendingRageClick.timerId);
                pendingRageClick.resolve(resolutionValue);
                pendingRageClick = null;
            }
            // if we have enough clicks to be a rage click, set a timout to trigger the rage
            // click event after the time threshold is reached.
            // This will be cancelled if a new click is tracked within the time threshold.
            if (clickWindow.length >= RAGE_CLICK_THRESHOLD) {
                return [2 /*return*/, new Promise(function (resolve) {
                        pendingRageClick = {
                            resolve: resolve,
                            timerId: setTimeout(function () {
                                resolve(getRageClickAnalyticsEvent(clickWindow));
                            }, RAGE_CLICK_WINDOW_MS),
                        };
                    })];
            }
            return [2 /*return*/, null];
        });
    }); });
    // reset the click window when a selection change occurs
    /* istanbul ignore next */
    var selectionSubscription = selectionObservable === null || selectionObservable === void 0 ? void 0 : selectionObservable.subscribe(function () {
        resetClickWindow();
    });
    var rageClickSubscription = rageClickObservable.subscribe(function (data) {
        /* istanbul ignore if */
        if (data === null) {
            return;
        }
        amplitude.track(AMPLITUDE_ELEMENT_RAGE_CLICKED_EVENT, data.rageClickEvent, { time: data.time });
    });
    return {
        unsubscribe: function () {
            rageClickSubscription.unsubscribe();
            /* istanbul ignore next */
            selectionSubscription === null || selectionSubscription === void 0 ? void 0 : selectionSubscription.unsubscribe();
        },
    };
}
//# sourceMappingURL=track-rage-click.js.map