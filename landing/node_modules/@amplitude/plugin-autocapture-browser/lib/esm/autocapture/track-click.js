import { filterOutNonTrackableEvents } from '../helpers';
import { AMPLITUDE_ELEMENT_CLICKED_EVENT } from '../constants';
export function trackClicks(_a) {
    var amplitude = _a.amplitude, allObservables = _a.allObservables, shouldTrackEvent = _a.shouldTrackEvent, evaluateTriggers = _a.evaluateTriggers;
    var clickObservable = allObservables.clickObservable;
    var clickObservableFiltered = clickObservable
        .filter(filterOutNonTrackableEvents)
        .filter(function (click) {
        // Only track clicks on elements that should be tracked,
        return shouldTrackEvent('click', click.closestTrackedAncestor);
    })
        .map(function (click) { return evaluateTriggers(click); });
    var clicks = clickObservableFiltered;
    return clicks.subscribe(function (click) {
        /* istanbul ignore next */
        amplitude === null || amplitude === void 0 ? void 0 : amplitude.track(AMPLITUDE_ELEMENT_CLICKED_EVENT, click.targetElementProperties);
    });
}
//# sourceMappingURL=track-click.js.map