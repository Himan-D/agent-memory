import { EnrichmentPlugin, PageTrackingOptions as Options } from '@amplitude/analytics-core';
export { PageTrackingOptions as Options, PageTrackingTrackOn, PageTrackingHistoryChanges, } from '@amplitude/analytics-core';
export interface CreatePageViewTrackingPlugin {
    (options?: Options): EnrichmentPlugin;
}
//# sourceMappingURL=page-view-tracking.d.ts.map