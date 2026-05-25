import { ElementInteractionsOptions, ActionType } from '@amplitude/analytics-core';
export type JSONValue = string | number | boolean | null | {
    [x: string]: JSONValue;
} | Array<JSONValue>;
export type shouldTrackEvent = (actionType: ActionType, element: Element) => boolean;
export declare const isElementPointerCursor: (element: Element, actionType: ActionType) => boolean;
export declare const isUrlAllowed: (autocaptureOptions: ElementInteractionsOptions) => boolean;
export declare const createShouldTrackEvent: (autocaptureOptions: ElementInteractionsOptions, allowlist: string[], isAlwaysCaptureCursorPointer?: boolean) => shouldTrackEvent;
export declare const isTextNode: (node: Node) => boolean;
export declare const isNonSensitiveElement: (element: Element) => boolean;
/**
 * Collects redacted attribute names from element and ancestor elements with data-amp-mask-attributes
 * The 'id' and 'class' attributes cannot be redacted as they're critical for element identification
 * @param element - The target element to check for redaction attributes
 * @returns Set of attribute names that should be redacted
 */
/**
 * Parses a comma-separated string of attribute names and filters out protected attributes
 * @param attributeString - Comma-separated string of attribute names
 * @returns Array of valid attribute names, excluding 'id' and 'class'
 */
export declare const parseAttributesToMask: (attributeString: string | null) => string[];
export declare const extractPrefixedAttributes: (attrs: {
    [key: string]: string;
}, prefix: string) => {
    [key: string]: string;
};
export declare const isEmpty: (value: unknown) => boolean;
export declare const removeEmptyProperties: (properties: {
    [key: string]: unknown;
}) => {
    [key: string]: unknown;
};
export declare const getCurrentPageViewId: () => string | undefined;
export declare const querySelectUniqueElements: (root: Element | Document, selectors: string[]) => Element[];
export declare const getClosestElement: (element: Element | null, selectors: string[]) => Element | null;
export declare const asyncLoadScript: (url: string) => Promise<unknown>;
export declare function generateUniqueId(): string;
export declare const filterOutNonTrackableEvents: (event: ElementBasedTimestampedEvent<ElementBasedEvent>) => boolean;
export type AutoCaptureOptionsWithDefaults = Required<Pick<ElementInteractionsOptions, 'debounceTime' | 'cssSelectorAllowlist' | 'actionClickAllowlist'>> & ElementInteractionsOptions;
export type BaseTimestampedEvent<T> = {
    event: T;
    timestamp: number;
    type: 'rage' | 'click' | 'change' | 'error' | 'navigate' | 'mutation';
};
export type ElementBasedEvent = MouseEvent | Event;
export type ElementBasedTimestampedEvent<T> = BaseTimestampedEvent<T> & {
    event: MouseEvent | Event;
    type: 'click' | 'change';
    closestTrackedAncestor: Element;
    targetElementProperties: Record<string, any>;
};
export type evaluateTriggersFn = (event: ElementBasedTimestampedEvent<ElementBasedEvent>) => ElementBasedTimestampedEvent<ElementBasedEvent>;
export type TimestampedEvent<T> = BaseTimestampedEvent<T> | ElementBasedTimestampedEvent<T>;
export declare function isElementBasedEvent<T>(event: BaseTimestampedEvent<T>): event is ElementBasedTimestampedEvent<T>;
export interface NavigateEvent extends Event {
    readonly navigationType: 'reload' | 'push' | 'replace' | 'traverse';
    readonly destination: {
        readonly url: string;
        readonly key: string | null;
        readonly id: string | null;
        readonly index: number;
        readonly sameDocument: boolean;
        getState(): any;
    };
    readonly canIntercept: boolean;
    readonly userInitiated: boolean;
    readonly hashChange: boolean;
    readonly signal: AbortSignal;
    readonly formData: FormData | null;
    readonly downloadRequest: string | null;
    readonly info: any;
    readonly hasUAVisualTransition: boolean;
    /** @see https://github.com/WICG/navigation-api/pull/264 */
    readonly sourceElement: Element | null;
    scroll(): void;
}
export declare enum MouseButton {
    LEFT_OR_TOUCH_CONTACT = 0,
    MIDDLE = 1,
    RIGHT = 2
}
//# sourceMappingURL=helpers.d.ts.map