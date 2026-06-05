import { __values } from "tslib";
import ZenObservable from 'zen-observable';
export { ZenObservable as Observable };
/**
 * asyncMap operator for Zen Observable
 *
 * Maps each value emitted by the source Observable using an async function,
 * emitting the resolved values in the same order they arrive.
 */
function asyncMap(observable, fn) {
    return new ZenObservable(function (observer) {
        observable.subscribe({
            next: function (value) {
                fn(value)
                    .then(function (result) {
                    return observer.next(result);
                })
                    .catch(function (error) { return observer.error(error); });
            },
            error: function (error) {
                observer.error(error);
            },
            complete: function () {
                observer.complete();
            },
        });
    });
}
/**
 * merge operator for Zen Observable
 *
 * Merges two observables into a single observable, emitting values from both sources in the order they arrive.
 * @param sourceA Observable to merge
 * @param sourceB Observable to merge
 * @returns Unsubscribable cleanup function
 */
function merge(sourceA, sourceB) {
    return new ZenObservable(function (observer) {
        var closed = false;
        var subscriptions = new Set();
        var cleanup = function () {
            var e_1, _a;
            closed = true;
            try {
                for (var subscriptions_1 = __values(subscriptions), subscriptions_1_1 = subscriptions_1.next(); !subscriptions_1_1.done; subscriptions_1_1 = subscriptions_1.next()) {
                    var sub = subscriptions_1_1.value;
                    try {
                        sub.unsubscribe();
                    }
                    catch (_b) {
                        /* do nothing */
                    }
                }
            }
            catch (e_1_1) { e_1 = { error: e_1_1 }; }
            finally {
                try {
                    if (subscriptions_1_1 && !subscriptions_1_1.done && (_a = subscriptions_1.return)) _a.call(subscriptions_1);
                }
                finally { if (e_1) throw e_1.error; }
            }
            subscriptions.clear();
        };
        var subscribeTo = function (source) {
            var sub = source.subscribe({
                next: function (value) {
                    if (!closed)
                        observer.next(value);
                },
                error: function (err) {
                    if (!closed) {
                        closed = true;
                        observer.error(err);
                        cleanup();
                    }
                },
                complete: function () {
                    subscriptions.delete(sub);
                    if (!closed && subscriptions.size === 0) {
                        observer.complete();
                        cleanup();
                        closed = true;
                    }
                },
            });
            subscriptions.add(sub);
        };
        subscribeTo(sourceA);
        subscribeTo(sourceB);
        return cleanup;
    });
}
// function share() {
function multicast(source) {
    var observers = new Set();
    var subscription = null;
    function cleanup() {
        /* istanbul ignore next */
        subscription === null || subscription === void 0 ? void 0 : subscription.unsubscribe();
        subscription = null;
        observers.clear();
    }
    return new ZenObservable(function (observer) {
        observers.add(observer);
        if (subscription === null) {
            subscription = source.subscribe({
                next: function (value) {
                    var e_2, _a;
                    var _b;
                    try {
                        for (var observers_1 = __values(observers), observers_1_1 = observers_1.next(); !observers_1_1.done; observers_1_1 = observers_1.next()) {
                            var obs = observers_1_1.value;
                            /* istanbul ignore next */
                            (_b = obs.next) === null || _b === void 0 ? void 0 : _b.call(obs, value);
                        }
                    }
                    catch (e_2_1) { e_2 = { error: e_2_1 }; }
                    finally {
                        try {
                            if (observers_1_1 && !observers_1_1.done && (_a = observers_1.return)) _a.call(observers_1);
                        }
                        finally { if (e_2) throw e_2.error; }
                    }
                },
                error: function (err) {
                    var e_3, _a;
                    var _b;
                    try {
                        for (var observers_2 = __values(observers), observers_2_1 = observers_2.next(); !observers_2_1.done; observers_2_1 = observers_2.next()) {
                            var obs = observers_2_1.value;
                            /* istanbul ignore next */
                            (_b = obs.error) === null || _b === void 0 ? void 0 : _b.call(obs, err);
                        }
                    }
                    catch (e_3_1) { e_3 = { error: e_3_1 }; }
                    finally {
                        try {
                            if (observers_2_1 && !observers_2_1.done && (_a = observers_2.return)) _a.call(observers_2);
                        }
                        finally { if (e_3) throw e_3.error; }
                    }
                    cleanup();
                },
                complete: function () {
                    var e_4, _a;
                    var _b;
                    try {
                        for (var observers_3 = __values(observers), observers_3_1 = observers_3.next(); !observers_3_1.done; observers_3_1 = observers_3.next()) {
                            var obs = observers_3_1.value;
                            /* istanbul ignore next */
                            (_b = obs.complete) === null || _b === void 0 ? void 0 : _b.call(obs);
                        }
                    }
                    catch (e_4_1) { e_4 = { error: e_4_1 }; }
                    finally {
                        try {
                            if (observers_3_1 && !observers_3_1.done && (_a = observers_3.return)) _a.call(observers_3);
                        }
                        finally { if (e_4) throw e_4.error; }
                    }
                    cleanup();
                },
            });
        }
        // Return unsubscribe function for this observer
        return function () {
            observers.delete(observer);
            // If no observers left, unsubscribe from the source
            if (observers.size === 0 && subscription) {
                subscription.unsubscribe();
                subscription = null;
            }
        };
    });
}
export { asyncMap, multicast, merge };
//# sourceMappingURL=observable.js.map