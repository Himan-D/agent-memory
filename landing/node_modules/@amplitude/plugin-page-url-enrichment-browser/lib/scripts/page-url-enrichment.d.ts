import { type EnrichmentPlugin, type PageUrlEnrichmentOptions } from '@amplitude/analytics-core';
export declare const CURRENT_PAGE_STORAGE_KEY = "AMP_CURRENT_PAGE";
export declare const PREVIOUS_PAGE_STORAGE_KEY = "AMP_PREVIOUS_PAGE";
export declare const URL_INFO_STORAGE_KEY = "AMP_URL_INFO";
export type URLInfo = {
    [CURRENT_PAGE_STORAGE_KEY]?: string;
    [PREVIOUS_PAGE_STORAGE_KEY]?: string;
};
export declare const EXCLUDED_DEFAULT_EVENT_TYPES: Set<string>;
export declare const isPageUrlEnrichmentEnabled: (option: unknown) => boolean;
export declare const pageUrlEnrichmentPlugin: ({ internalDomains }?: PageUrlEnrichmentOptions) => EnrichmentPlugin;
//# sourceMappingURL=page-url-enrichment.d.ts.map