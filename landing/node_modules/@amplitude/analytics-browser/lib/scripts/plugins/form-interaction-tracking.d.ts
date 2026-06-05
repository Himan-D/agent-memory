import { EnrichmentPlugin } from '@amplitude/analytics-core';
export declare const formInteractionTracking: () => EnrichmentPlugin;
export declare const stringOrUndefined: <T>(name: T) => T extends string ? string : undefined;
export declare const extractFormAction: (form: HTMLFormElement) => string | null;
//# sourceMappingURL=form-interaction-tracking.d.ts.map