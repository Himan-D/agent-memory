import ZenObservable from 'zen-observable';
export { ZenObservable as Observable };
/**
 * asyncMap operator for Zen Observable
 *
 * Maps each value emitted by the source Observable using an async function,
 * emitting the resolved values in the same order they arrive.
 */
declare function asyncMap<T, R>(observable: ZenObservable<T>, fn: (value: T) => Promise<R>): ZenObservable<R>;
type Unsubscribable = {
    unsubscribe: () => void;
};
/**
 * merge operator for Zen Observable
 *
 * Merges two observables into a single observable, emitting values from both sources in the order they arrive.
 * @param sourceA Observable to merge
 * @param sourceB Observable to merge
 * @returns Unsubscribable cleanup function
 */
declare function merge<A, B>(sourceA: ZenObservable<A>, sourceB: ZenObservable<B>): ZenObservable<A | B>;
declare function multicast<T>(source: ZenObservable<T>): ZenObservable<T>;
export { asyncMap, multicast, merge, Unsubscribable };
//# sourceMappingURL=observable.d.ts.map