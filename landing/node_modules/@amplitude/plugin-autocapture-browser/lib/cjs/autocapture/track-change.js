"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.trackChange = void 0;
var helpers_1 = require("../helpers");
var constants_1 = require("../constants");
function trackChange(_a) {
    var amplitude = _a.amplitude, allObservables = _a.allObservables, getEventProperties = _a.getEventProperties, shouldTrackEvent = _a.shouldTrackEvent, evaluateTriggers = _a.evaluateTriggers;
    var changeObservable = allObservables.changeObservable;
    var filteredChangeObservable = changeObservable
        .filter(helpers_1.filterOutNonTrackableEvents)
        .filter(function (changeEvent) {
        // Only track change on elements that should be tracked,
        return shouldTrackEvent('change', changeEvent.closestTrackedAncestor);
    })
        .map(function (changeEvent) { return evaluateTriggers(changeEvent); });
    return filteredChangeObservable.subscribe(function (changeEvent) {
        /* istanbul ignore next */
        amplitude === null || amplitude === void 0 ? void 0 : amplitude.track(constants_1.AMPLITUDE_ELEMENT_CHANGED_EVENT, getEventProperties('change', changeEvent.closestTrackedAncestor));
    });
}
exports.trackChange = trackChange;
//# sourceMappingURL=track-change.js.map