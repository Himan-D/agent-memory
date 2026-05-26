import { filterOutNonTrackableEvents, } from '../helpers';
import { AMPLITUDE_ELEMENT_CHANGED_EVENT } from '../constants';
export function trackChange(_a) {
    var amplitude = _a.amplitude, allObservables = _a.allObservables, getEventProperties = _a.getEventProperties, shouldTrackEvent = _a.shouldTrackEvent, evaluateTriggers = _a.evaluateTriggers;
    var changeObservable = allObservables.changeObservable;
    var filteredChangeObservable = changeObservable
        .filter(filterOutNonTrackableEvents)
        .filter(function (changeEvent) {
        // Only track change on elements that should be tracked,
        return shouldTrackEvent('change', changeEvent.closestTrackedAncestor);
    })
        .map(function (changeEvent) { return evaluateTriggers(changeEvent); });
    return filteredChangeObservable.subscribe(function (changeEvent) {
        /* istanbul ignore next */
        amplitude === null || amplitude === void 0 ? void 0 : amplitude.track(AMPLITUDE_ELEMENT_CHANGED_EVENT, getEventProperties('change', changeEvent.closestTrackedAncestor));
    });
}
//# sourceMappingURL=track-change.js.map