import { TimestampedEvent } from './helpers';
import { Observable } from '@amplitude/analytics-core';
export declare const createMutationObservable: () => Observable<MutationRecord[]>;
/**
 * Creates an observable that tracks click events on the document.
 * @param clickType - The type of click event to track (click or pointerdown)
 */
export declare const createClickObservable: (clickType?: 'click' | 'pointerdown') => Observable<MouseEvent | PointerEvent>;
export declare const createScrollObservable: () => Observable<Event>;
export declare const createExposureObservable: (mutationObservable: Observable<TimestampedEvent<MutationRecord[]>>, selectorAllowlist: string[]) => Observable<Event>;
export type BrowserErrorEvent = {
    kind: 'error' | 'unhandledrejection' | 'console';
    message?: string;
    filename?: string;
    lineNumber?: number;
    columnNumber?: number;
    stack?: string;
};
export declare const createErrorObservable: () => Observable<BrowserErrorEvent>;
export declare const createMouseMoveObservable: () => Observable<MouseEvent>;
//# sourceMappingURL=observables.d.ts.map