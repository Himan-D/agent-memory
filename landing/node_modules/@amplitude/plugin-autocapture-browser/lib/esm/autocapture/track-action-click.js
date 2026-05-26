import { merge, asyncMap } from '@amplitude/analytics-core';
import { filterOutNonTrackableEvents, getClosestElement, } from '../helpers';
import { AMPLITUDE_ELEMENT_CLICKED_EVENT } from '../constants';
export function trackActionClick(_a) {
    var amplitude = _a.amplitude, allObservables = _a.allObservables, options = _a.options, getEventProperties = _a.getEventProperties, shouldTrackEvent = _a.shouldTrackEvent, shouldTrackActionClick = _a.shouldTrackActionClick;
    var clickObservable = allObservables.clickObservable, mutationObservable = allObservables.mutationObservable, navigateObservable = allObservables.navigateObservable;
    var filteredClickObservable = clickObservable
        .filter(function (click) {
        return !shouldTrackEvent('click', click.closestTrackedAncestor);
    })
        .map(function (click) {
        // overwrite the closestTrackedAncestor with the closest element that is on the actionClickAllowlist
        var closestActionClickEl = getClosestElement(click.event.target, options.actionClickAllowlist);
        click.closestTrackedAncestor = closestActionClickEl;
        // overwrite the targetElementProperties with the properties of the closestActionClickEl
        if (click.closestTrackedAncestor !== null) {
            click.targetElementProperties = getEventProperties(click.type, click.closestTrackedAncestor);
        }
        return click;
    })
        .filter(filterOutNonTrackableEvents)
        .filter(function (clickEvent) {
        // Only track change on elements that should be tracked
        return shouldTrackActionClick('click', clickEvent.closestTrackedAncestor);
    });
    var mutationOrNavigate = navigateObservable ? merge(mutationObservable, navigateObservable) : mutationObservable;
    var clickMutationNavigateObservable = merge(filteredClickObservable, mutationOrNavigate);
    var actionClickTimer = null;
    var lastClickEvent = null;
    var actionClickObservable = asyncMap(clickMutationNavigateObservable, function (event) {
        // clear any previous timer
        if (actionClickTimer) {
            clearTimeout(actionClickTimer);
            actionClickTimer = null;
        }
        if (event.type === 'click') {
            // mark the 'last click event'
            lastClickEvent = event;
            // set a timer to clear last click event if no mutation event between now and 500ms
            actionClickTimer = setTimeout(function () {
                actionClickTimer = null;
                lastClickEvent = null;
            }, 500);
            return Promise.resolve(null);
        }
        else {
            // if mutation/navigation + last click event, then it's an action click
            if (lastClickEvent) {
                var event_1 = lastClickEvent;
                lastClickEvent = null;
                return Promise.resolve(event_1);
            }
        }
        return Promise.resolve(null);
    });
    return actionClickObservable.subscribe(function (actionClick) {
        if (!actionClick)
            return;
        /* istanbul ignore next */
        amplitude === null || amplitude === void 0 ? void 0 : amplitude.track(AMPLITUDE_ELEMENT_CLICKED_EVENT, getEventProperties('click', actionClick.closestTrackedAncestor));
    });
}
//# sourceMappingURL=track-action-click.js.map