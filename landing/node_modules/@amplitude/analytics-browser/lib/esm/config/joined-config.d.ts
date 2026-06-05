import { AutocaptureOptions, type ElementInteractionsOptions, BrowserConfig, RemoteConfig, NetworkTrackingOptions, NetworkCaptureRule } from '@amplitude/analytics-core';
export interface AutocaptureOptionsRemoteConfig extends AutocaptureOptions {
    elementInteractions?: boolean | ElementInteractionsOptionsRemoteConfig;
    networkTracking?: boolean | NetworkTrackingOptionsRemoteConfig;
}
export interface ElementInteractionsOptionsRemoteConfig extends ElementInteractionsOptions {
    /**
     * Related to pageUrlAllowlist but holds regex strings which will be initialized and appended to pageUrlAllowlist
     */
    pageUrlAllowlistRegex?: string[];
}
export interface NetworkCaptureRuleRemoteConfig extends NetworkCaptureRule {
    /**
     * Related to urls but holds regex strings which will be initialized and appended to urls
     */
    urlsRegex?: string[];
}
export interface NetworkTrackingOptionsRemoteConfig extends NetworkTrackingOptions {
    /**
     * Related to pageUrlAllowlist but holds regex strings which will be initialized and appended to pageUrlAllowlist
     */
    captureRules?: NetworkCaptureRuleRemoteConfig[];
}
/**
 * Performs a deep transformation of a remote config object so that
 * it matches the expected schema of the local config.
 *
 * Specifically, it normalizes nested `enabled` flags into concise union types.
 *
 * ### Transformation Rules:
 * - If an object has `enabled: true`, it is replaced by the same object without the `enabled` field.
 * - If it has only `enabled: true`, it is replaced with `true`.
 * - If it has `enabled: false`, it is replaced with `false` regardless of other fields.
 *
 * ### Examples:
 * Input:  { prop: { enabled: true, hello: 'world' }}
 * Output: { prop: { hello: 'world' } }
 *
 * Input:  { prop: { enabled: true }}
 * Output: { prop: true }
 *
 * Input:  { prop: { enabled: false, hello: 'world' }}
 * Output: { prop: false }
 *
 * Input:  { prop: { hello: 'world' }}
 * Output: { prop: { hello: 'world' } } // No change
 *
 * @param config Remote config object to be transformed
 * @returns Transformed config object compatible with local schema
 */
export declare function translateRemoteConfigToLocal(config?: Record<string, any>): void;
/**
 * Updates the browser config in place by applying remote configuration settings.
 * Primarily merges autocapture settings from the remote config into the browser config.
 *
 * @param remoteConfig - The remote configuration to apply, or null if none available
 * @param browserConfig - The browser config object to update (modified in place)
 */
export declare function updateBrowserConfigWithRemoteConfig(remoteConfig: RemoteConfig | null, browserConfig: BrowserConfig): void;
//# sourceMappingURL=joined-config.d.ts.map