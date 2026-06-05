"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.trackClicks = void 0;
var helpers_1 = require("../helpers");
var constants_1 = require("../constants");
function trackClicks(_a) {
    var amplitude = _a.amplitude, allObservables = _a.allObservables, shouldTrackEvent = _a.shouldTrackEvent, evaluateTriggers = _a.evaluateTriggers;
    var clickObservable = allObservables.clickObservable;
    var clickObservableFiltered = clickObservable
        .filter(helpers_1.filterOutNonTrackableEvents)
        .filter(function (click) {
        // Only track clicks on elements that should be tracked,
        return shouldTrackEvent('click', click.closestTrackedAncestor);
    })
        .map(function (click) { return evaluateTriggers(click); });
    var clicks = clickObservableFiltered;
    return clicks.subscribe(function (click) {
        /* istanbul ignore next */
        amplitude === null || amplitude === void 0 ? void 0 : amplitude.track(constants_1.AMPLITUDE_ELEMENT_CLICKED_EVENT, click.targetElementProperties);
    });
}
exports.trackClicks = trackClicks;
//# sourceMappingURL=track-click.js.map