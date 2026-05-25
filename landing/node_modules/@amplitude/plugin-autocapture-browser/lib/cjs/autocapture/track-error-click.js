"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.trackErrorClicks = void 0;
var tslib_1 = require("tslib");
var helpers_1 = require("../helpers");
var analytics_core_1 = require("@amplitude/analytics-core");
var constants_1 = require("../constants");
var ERROR_CLICK_TIMEOUT = 2000; // 2 seconds to wait for an error to happen
function trackErrorClicks(_a) {
    var amplitude = _a.amplitude, allObservables = _a.allObservables, shouldTrackErrorClick = _a.shouldTrackErrorClick;
    var clickObservable = allObservables.clickObservable, browserErrorObservable = allObservables.browserErrorObservable;
    var filteredClickObservable = clickObservable.filter(function (click) {
        return ((0, helpers_1.filterOutNonTrackableEvents)(click) &&
            shouldTrackErrorClick('click', click.closestTrackedAncestor) &&
            click.event.target instanceof Element &&
            click.event.target.closest('a[target="_blank"]') === null &&
            click.event.button === helpers_1.MouseButton.LEFT_OR_TOUCH_CONTACT);
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
    return (0, analytics_core_1.merge)(filteredClickObservable, browserErrorObservable).subscribe(function (event) {
        var _a;
        if (event.type === 'click') {
            clearClickTimer();
            latestClickEvent = event;
            errorClickTimer = setTimeout(clearClickTimer, ERROR_CLICK_TIMEOUT);
            return;
        }
        if (event.type === 'error' && latestClickEvent) {
            amplitude.track(constants_1.AMPLITUDE_ELEMENT_ERROR_CLICKED_EVENT, tslib_1.__assign((_a = {}, _a['[Amplitude] Kind'] = event.event.kind, _a['[Amplitude] Message'] = event.event.message, _a['[Amplitude] Stack'] = event.event.stack, _a['[Amplitude] Filename'] = event.event.filename, _a['[Amplitude] Line Number'] = event.event.lineNumber, _a['[Amplitude] Column Number'] = event.event.columnNumber, _a), latestClickEvent.targetElementProperties));
            clearClickTimer();
        }
    });
}
exports.trackErrorClicks = trackErrorClicks;
//# sourceMappingURL=track-error-click.js.map