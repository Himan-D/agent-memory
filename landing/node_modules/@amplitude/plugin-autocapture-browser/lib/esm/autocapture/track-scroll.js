import { getGlobalScope } from '@amplitude/analytics-core';
export function trackScroll(_a) {
    var amplitude = _a.amplitude, allObservables = _a.allObservables;
    // amplitude is reserved for future periodic scroll event tracking
    void amplitude;
    var scrollObservable = allObservables.scrollObservable;
    var state = { maxX: 0, maxY: 0 };
    var scrollSubscription = scrollObservable.subscribe(function () {
        var _a, _b, _c, _d;
        var globalScope = getGlobalScope();
        /* istanbul ignore next */
        var currentX = Math.floor((_b = (_a = globalScope === null || globalScope === void 0 ? void 0 : globalScope.scrollX) !== null && _a !== void 0 ? _a : globalScope === null || globalScope === void 0 ? void 0 : globalScope.pageXOffset) !== null && _b !== void 0 ? _b : 0);
        /* istanbul ignore next */
        var currentY = Math.floor((_d = (_c = globalScope === null || globalScope === void 0 ? void 0 : globalScope.scrollY) !== null && _c !== void 0 ? _c : globalScope === null || globalScope === void 0 ? void 0 : globalScope.pageYOffset) !== null && _d !== void 0 ? _d : 0);
        // Update page-level max positions for Page View End event (never resets during page lifetime)
        state.maxX = Math.max(state.maxX, currentX);
        state.maxY = Math.max(state.maxY, currentY);
    });
    return {
        unsubscribe: function () {
            scrollSubscription.unsubscribe();
        },
        getState: function () { return state; },
        reset: function () {
            state.maxX = 0;
            state.maxY = 0;
        },
    };
}
//# sourceMappingURL=track-scroll.js.map