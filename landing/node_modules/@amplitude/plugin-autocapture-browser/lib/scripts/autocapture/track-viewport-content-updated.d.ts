import { BrowserClient } from '@amplitude/analytics-core';
export interface ScrollTracker {
    getState: () => {
        maxX: number;
        maxY: number;
    };
    reset: () => void;
}
export interface ExposureTracker {
    reset: () => void;
}
export declare function fireViewportContentUpdated({ amplitude, scrollTracker, currentElementExposed, elementExposedForPage, exposureTracker, isPageEnd, lastScroll, }: {
    amplitude: BrowserClient;
    scrollTracker: ScrollTracker;
    currentElementExposed: Set<string>;
    elementExposedForPage: Set<string>;
    exposureTracker: ExposureTracker | undefined;
    isPageEnd: boolean;
    lastScroll: {
        maxX: undefined | number;
        maxY: undefined | number;
    };
}): void;
export declare function onExposure(elementPath: string, elementExposedForPage: Set<string>, currentElementExposed: Set<string>, fireViewportContentUpdatedCallback: (isPageEnd: boolean) => void): void;
//# sourceMappingURL=track-viewport-content-updated.d.ts.map