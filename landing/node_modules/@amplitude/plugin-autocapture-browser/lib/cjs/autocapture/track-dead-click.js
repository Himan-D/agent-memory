"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.trackDeadClick = void 0;
var tslib_1 = require("tslib");
var analytics_core_1 = require("@amplitude/analytics-core");
var helpers_1 = require("../helpers");
var constants_1 = require("../constants");
var DEAD_CLICK_TIMEOUT = 3000; // 3 seconds to wait for an activity to happen
var CHANGE_EVENTS = ['mutation', 'navigate'];
function trackDeadClick(_a) {
    var amplitude = _a.amplitude, allObservables = _a.allObservables, getEventProperties = _a.getEventProperties, shouldTrackDeadClick = _a.shouldTrackDeadClick;
    var clickObservable = allObservables.clickObservable, mutationObservable = allObservables.mutationObservable, navigateObservable = allObservables.navigateObservable;
    var filteredClickObservable = clickObservable.filter(function (click) {
        return ((0, helpers_1.filterOutNonTrackableEvents)(click) &&
            shouldTrackDeadClick('click', click.closestTrackedAncestor) &&
            click.event.target instanceof Element &&
            click.event.target.closest('a[target="_blank"]') === null &&
            click.event.button === helpers_1.MouseButton.LEFT_OR_TOUCH_CONTACT);
    });
    /* istanbul ignore next */
    var changeObservables = navigateObservable ? (0, analytics_core_1.merge)(mutationObservable, navigateObservable) : mutationObservable;
    var clicksAndChangeObservable = (0, analytics_core_1.merge)(filteredClickObservable, changeObservables);
    var deadClickTimer = null;
    var deadClickObservable = (0, analytics_core_1.asyncMap)(clicksAndChangeObservable, function (event) {
        if (deadClickTimer && CHANGE_EVENTS.includes(event.type)) {
            // a mutation or navigation means it's not a dead click, so clear the timer
            clearTimeout(deadClickTimer);
            deadClickTimer = null;
            return Promise.resolve(null);
        }
        else if (event.type === 'click') {
            // if a dead click is already on-deck, return null.
            // this throttles dead clicks events so no more than one dead click event
            // is tracked per every DEAD_CLICK_TIMEOUT ms.
            if (deadClickTimer) {
                return Promise.resolve(null);
            }
            return new Promise(function (resolve) {
                deadClickTimer = setTimeout(function () {
                    resolve(event);
                    deadClickTimer = null;
                }, DEAD_CLICK_TIMEOUT);
            });
        }
        // unreachable code, but needed to satisfy the type checker
        return Promise.resolve(null);
    });
    return deadClickObservable.subscribe(function (actionClick) {
        if (!actionClick)
            return;
        var deadClickEvent = {
            '[Amplitude] X': actionClick.event.clientX,
            '[Amplitude] Y': actionClick.event.clientY,
        };
        amplitude.track(constants_1.AMPLITUDE_ELEMENT_DEAD_CLICKED_EVENT, tslib_1.__assign(tslib_1.__assign({}, getEventProperties('click', actionClick.closestTrackedAncestor)), deadClickEvent), { time: actionClick.timestamp });
    });
}
exports.trackDeadClick = trackDeadClick;
//# sourceMappingURL=track-dead-click.js.map