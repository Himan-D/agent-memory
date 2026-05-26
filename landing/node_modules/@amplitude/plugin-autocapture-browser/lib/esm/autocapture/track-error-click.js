import { __assign } from "tslib";
import { filterOutNonTrackableEvents, MouseButton, } from '../helpers';
import { merge } from '@amplitude/analytics-core';
import { AMPLITUDE_ELEMENT_ERROR_CLICKED_EVENT } from '../constants';
var ERROR_CLICK_TIMEOUT = 2000; // 2 seconds to wait for an error to happen
export function trackErrorClicks(_a) {
    var amplitude = _a.amplitude, allObservables = _a.allObservables, shouldTrackErrorClick = _a.shouldTrackErrorClick;
    var clickObservable = allObservables.clickObservable, browserErrorObservable = allObservables.browserErrorObservable;
    var filteredClickObservable = clickObservable.filter(function (click) {
        return (filterOutNonTrackableEvents(click) &&
            shouldTrackErrorClick('click', click.closestTrackedAncestor) &&
            click.event.target instanceof Element &&
            click.event.target.closest('a[target="_blank"]') === null &&
            click.event.button === MouseButton.LEFT_OR_TOUCH_CONTACT);
    });
    var errorClickTimer = null;
    var latestClickEvent = null;
    var clearClickTimer = function () {
        if (errorClickTimer !== null) {
            clearTimeout(errorClickTimer);
            errorClickTimer = null;
        }
        latestClickEvent = null;
    };
    return merge(filteredClickObservable, browserErrorObservable).subscribe(function (event) {
        var _a;
        if (event.type === 'click') {
            clearClickTimer();
            latestClickEvent = event;
            errorClickTimer = setTimeout(clearClickTimer, ERROR_CLICK_TIMEOUT);
            return;
        }
        if (event.type === 'error' && latestClickEvent) {
            amplitude.track(AMPLITUDE_ELEMENT_ERROR_CLICKED_EVENT, __assign((_a = {}, _a['[Amplitude] Kind'] = event.event.kind, _a['[Amplitude] Message'] = event.event.message, _a['[Amplitude] Stack'] = event.event.stack, _a['[Amplitude] Filename'] = event.event.filename, _a['[Amplitude] Line Number'] = event.event.lineNumber, _a['[Amplitude] Column Number'] = event.event.columnNumber, _a), latestClickEvent.targetElementProperties));
            clearClickTimer();
        }
    });
}
//# sourceMappingURL=track-error-click.js.map