import { ElementInteractionsOptions, ActionType, IDiagnosticsClient } from '@amplitude/analytics-core';
import type { DataSource } from '@amplitude/analytics-core/lib/esm/types/element-interactions';
import type { BaseTimestampedEvent, ElementBasedTimestampedEvent, TimestampedEvent, JSONValue } from './helpers';
import { Hierarchy } from './typings/autocapture';
export declare class DataExtractor {
    private readonly additionalMaskTextPatterns;
    diagnosticsClient?: IDiagnosticsClient;
    constructor(options: ElementInteractionsOptions, context?: {
        diagnosticsClient: IDiagnosticsClient;
    });
    /**
     * Wrapper method to replace sensitive strings using the helper function
     * @param text - The text to search for sensitive data
     * @returns The text with sensitive data replaced by masked text
     */
    replaceSensitiveString: (text: string | null) => string;
    getHierarchy: (element: Element | null) => Hierarchy;
    getNearestLabel: (element: Element) => string;
    getElementPath: (element: Element | null) => string;
    getEventProperties: (actionType: ActionType, element: Element, dataAttributePrefix: string) => {
        [key: string]: unknown;
    };
    addTypeAndTimestamp: <T>(event: T, type: "error" | "navigate" | "change" | "click" | "rage" | "mutation") => BaseTimestampedEvent<T> | ElementBasedTimestampedEvent<T>;
    addAdditionalEventProperties: <T>(event: T, type: "error" | "navigate" | "change" | "click" | "rage" | "mutation", selectorAllowlist: string[], dataAttributePrefix: string, isCapturingCursorPointer?: boolean) => TimestampedEvent<T>;
    extractDataFromDataSource: (dataSource: DataSource, contextElement: HTMLElement) => string | null | undefined;
    private getTextWithMaskedDescendants;
    getText: (element: Element) => string;
    getEventTagProps: (element: Element) => Record<string, JSONValue>;
}
//# sourceMappingURL=data-extractor.d.ts.map