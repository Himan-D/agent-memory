import { __assign } from "tslib";
import { merge, asyncMap } from '@amplitude/analytics-core';
import { filterOutNonTrackableEvents, MouseButton } from '../helpers';
import { AMPLITUDE_ELEMENT_DEAD_CLICKED_EVENT } from '../constants';
var DEAD_CLICK_TIMEOUT = 3000; // 3 seconds to wait for an activity to happen
var CHANGE_EVENTS = ['mutation', 'navigate'];
export function trackDeadClick(_a) {
    var amplitude = _a.amplitude, allObservables = _a.allObservables, getEventProperties = _a.getEventProperties, shouldTrackDeadClick = _a.shouldTrackDeadClick;
    var clickObservable = allObservables.clickObservable, mutationObservable = allObservables.mutationObservable, navigateObservable = allObservables.navigateObservable;
    var filteredClickObservable = clickObservable.filter(function (click) {
        return (filterOutNonTrackableEvents(click) &&
            shouldTrackDeadClick('click', click.closestTrackedAncestor) &&
            click.event.target instanceof Element &&
            click.event.target.closest('a[target="_blank"]') === null &&
            click.event.button === MouseButton.LEFT_OR_TOUCH_CONTACT);
    });
    /* istanbul ignore next */
    var changeObservables = navigateObservable ? merge(mutationObservable, navigateObservable) : mutationObservable;
    var clicksAndChangeObservable = merge(filteredClickObservable, changeObservables);
    var deadClickTimer = null;
    var deadClickObservable = asyncMap(clicksAndChangeObservable, function (event) {
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
        amplitude.track(AMPLITUDE_ELEMENT_DEAD_CLICKED_EVENT, __assign(__assign({}, getEventProperties('click', actionClick.closestTrackedAncestor)), deadClickEvent), { time: actionClick.timestamp });
    });
}
//# sourceMappingURL=track-dead-click.js.map