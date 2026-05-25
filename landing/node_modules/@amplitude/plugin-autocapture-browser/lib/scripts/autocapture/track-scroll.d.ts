import { AllWindowObservables } from '../autocapture-plugin';
import { BrowserClient } from '@amplitude/analytics-core';
export interface ScrollState {
    maxX: number;
    maxY: number;
}
export declare function trackScroll({ amplitude, allObservables, }: {
    amplitude: BrowserClient;
    allObservables: AllWindowObservables;
}): {
    unsubscribe: () => void;
    getState: () => ScrollState;
    reset: () => void;
};
//# sourceMappingURL=track-scroll.d.ts.map