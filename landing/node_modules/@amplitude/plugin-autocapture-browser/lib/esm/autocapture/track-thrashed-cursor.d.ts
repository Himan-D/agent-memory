import { BrowserClient, ElementInteractionsOptions, Observable } from '@amplitude/analytics-core';
import { AllWindowObservables } from '../frustration-plugin';
declare enum Axis {
    X = "x",
    Y = "y"
}
export declare const createMouseDirectionChangeObservable: ({ allWindowObservables, }: {
    allWindowObservables: AllWindowObservables;
}) => Observable<Axis>;
export declare const createThrashedCursorObservable: ({ mouseDirectionChangeObservable, directionChanges, thresholdMs, }: {
    mouseDirectionChangeObservable: Observable<Axis>;
    directionChanges?: number | undefined;
    thresholdMs?: number | undefined;
}) => Observable<number>;
export declare const trackThrashedCursor: ({ amplitude, options, allObservables, directionChanges, thresholdMs, }: {
    amplitude: BrowserClient;
    options: ElementInteractionsOptions;
    allObservables: AllWindowObservables;
    directionChanges?: number | undefined;
    thresholdMs?: number | undefined;
}) => ZenObservable.Subscription;
export {};
//# sourceMappingURL=track-thrashed-cursor.d.ts.map