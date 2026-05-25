"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.trackActionClick = void 0;
var analytics_core_1 = require("@amplitude/analytics-core");
var helpers_1 = require("../helpers");
var constants_1 = require("../constants");
function trackActionClick(_a) {
    var amplitude = _a.amplitude, allObservables = _a.allObservables, options = _a.options, getEventProperties = _a.getEventProperties, shouldTrackEvent = _a.shouldTrackEvent, shouldTrackActionClick = _a.shouldTrackActionClick;
    var clickObservable = allObservables.clickObservable, mutationObservable = allObservables.mutationObservable, navigateObservable = allObservables.navigateObservable;
    var filteredClickObservable = clickObservable
        .filter(function (click) {
        return !shouldTrackEvent('click', click.closestTrackedAncestor);
    })
        .map(function (click) {
        // overwrite the closestTrackedAncestor with the closest element that is on the actionClickAllowlist
        var closestActionClickEl = (0, helpers_1.getClosestElement)(click.event.target, options.actionClickAllowlist);
        click.closestTrackedAncestor = closestActionClickEl;
        // overwrite the targetElementProperties with the properties of the closestActionClickEl
        if (click.closestTrackedAncestor !== null) {
            click.targetElementProperties = getEventProperties(click.type, click.closestTrackedAncestor);
        }
        return click;
    })
        .filter(helpers_1.filterOutNonTrackableEvents)
        .filter(function (clickEvent) {
        // Only track change on elements that should be tracked
        return shouldTrackActionClick('click', clickEvent.closestTrackedAncestor);
    });
    var mutationOrNavigate = navigateObservable ? (0, analytics_core_1.merge)(mutationObservable, navigateObservable) : mutationObservable;
    var clickMutationNavigateObservable = (0, analytics_core_1.merge)(filteredClickObservable, mutationOrNavigate);
    var actionClickTimer = null;
    var lastClickEvent = null;
    var actionClickObservable = (0, analytics_core_1.asyncMap)(clickMutationNavigateObservable, function (event) {
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
        amplitude === null || amplitude === void 0 ? void 0 : amplitude.track(constants_1.AMPLITUDE_ELEMENT_CLICKED_EVENT, getEventProperties('click', actionClick.closestTrackedAncestor));
    });
}
exports.trackActionClick = trackActionClick;
//# sourceMappingURL=track-action-click.js.map