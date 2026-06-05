import { ServerZoneType } from '../types/server-zone';
import { ILogger } from '../logger';
/**
 * Modes for receiving remote config updates:
 * - `'all'` – Optimized for both speed and freshness. Returns the fastest response first
 *   (cache or remote), then always waits for and returns the remote response to ensure
 *   the most up-to-date config. Callback may be called once (if remote wins) or twice
 *   (cache first, then remote).
 * - `{ timeout: number }` – Prefers remote data but with a fallback strategy. Waits for
 *   a remote response until the specified timeout (in milliseconds), then falls back to
 *   cached data if available. Callback is called exactly once.
 */
export type DeliveryMode = 'all' | {
    timeout: number;
};
/**
 * Sources of returned remote config:
 * - `cache` - Fetched from local storage.
 * - `remote` - Fetched from remote.
 */
export type Source = 'cache' | 'remote';
export declare const US_SERVER_URL = "https://sr-client-cfg.amplitude.com/config";
export declare const EU_SERVER_URL = "https://sr-client-cfg.eu.amplitude.com/config";
export declare const DEFAULT_MAX_RETRIES = 3;
export interface RemoteConfig {
    [key: string]: any;
}
export interface RemoteConfigInfo {
    remoteConfig: RemoteConfig | null;
    lastFetch: Date;
}
export interface RemoteConfigStorage {
    /**
     * Fetch remote config from storage asynchronously.
     */
    fetchConfig(): Promise<RemoteConfigInfo>;
    /**
     * Set remote config to storage asynchronously.
     */
    setConfig(config: RemoteConfigInfo): Promise<boolean>;
}
/**
 * Information about each callback registered by `RemoteConfigClient.subscribe()`,
 * managed internally by `RemoteConfigClient`.
 */
export interface CallbackInfo {
    id: string;
    key?: string;
    deliveryMode: DeliveryMode;
    callback: RemoteConfigCallback;
    lastCallback?: Date;
}
/**
 * Callback used in `RemoteConfigClient.subscribe()`.
 * This function is called when the remote config is fetched.
 */
type RemoteConfigCallback = (remoteConfig: RemoteConfig | null, source: Source, lastFetch: Date) => void;
export interface IRemoteConfigClient {
    /**
     * Subscribe for updates to remote config.
     * Callback is guaranteed to be called at least once,
     * Whether we are able to fetch a config or not.
     *
     * @param key - a string containing a series of period delimited keys to filter the returned config.
     * Ie, {a: {b: {c: ...}}} would return {b: {c: ...}} for "a" or {c: ...} for "a.b".
     * Set to `undefined` to subscribe all keys.
     * @param deliveryMode - how the initial callback is sent.
     * @param callback - a block that will be called when remote config is fetched.
     * @return id - identification of the subscribe and can be used to unsubscribe from updates.
     */
    subscribe(key: string | undefined, deliveryMode: DeliveryMode, callback: RemoteConfigCallback): string;
    /**
     * Unsubscribe a callback from receiving future updates.
     *
     * @param id - identification of the callback that you want to unsubscribe.
     * It's the return value of subscribe().
     * @return boolean - whether the callback is removed.
     */
    unsubscribe(id: string): boolean;
    /**
     * Request the remote config client to fetch from remote, update cache, and callback.
     */
    updateConfigs(): void;
}
export declare class RemoteConfigClient implements IRemoteConfigClient {
    static readonly CONFIG_GROUP = "browser";
    readonly apiKey: string;
    readonly serverUrl: string;
    readonly logger: ILogger;
    readonly storage: RemoteConfigStorage;
    callbackInfos: CallbackInfo[];
    lastSuccessfulFetch: number | null;
    fetchPromise: Promise<RemoteConfigInfo> | null;
    isLastFetchInvalidApiKey: boolean;
    constructor(apiKey: string, logger: ILogger, serverZone?: ServerZoneType, serverUrl?: string);
    subscribe(key: string | undefined, deliveryMode: DeliveryMode, callback: RemoteConfigCallback): string;
    unsubscribe(id: string): boolean;
    updateConfigs(): Promise<void>;
    /**
     * Get the in-flight fetch promise or create a new one.
     * This ensures multiple subscribe calls share the same network request.
     */
    getOrCreateFetchPromise(): Promise<RemoteConfigInfo>;
    /**
     * Send remote first. If it's already complete, we can skip the cached response.
     * - if remote is fetched first, no cache fetch.
     * - if cache is fetched first, still fetching remote.
     */
    subscribeAll(callbackInfo: CallbackInfo): Promise<void>;
    /**
     * Waits for a remote response until the given timeout, then return a cached copy, if available.
     */
    subscribeWaitForRemote(callbackInfo: CallbackInfo, timeout: number): Promise<void>;
    /**
     * Call the callback with filtered remote config based on key.
     * @param remoteConfigInfo - the whole remote config object without filtering by key.
     */
    sendCallback(callbackInfo: CallbackInfo, remoteConfigInfo: RemoteConfigInfo, source: Source): void;
    /**
     * Fetch remote config from remote.
     * @param retries - the number of retries. default is 3.
     * @param timeout - the timeout in milliseconds. Default is 1000.
     * This timeout serves two purposes:
     * 1. It determines how long to wait for each remote config fetch request before aborting it.
     *    If the fetch does not complete within the specified timeout, the request is cancelled using AbortController,
     *    and the attempt is considered failed (and may be retried if retries remain).
     * 2. It is also used to calculate the interval between retries. The total timeout is divided by the number of retries,
     *    so each retry waits for (timeout / retries) milliseconds before the next attempt (linear backoff).
     * Retry behavior by status code:
     * - 401: invalid API key (stop retries and disable future updateConfigs calls).
     * - 429: retry up to max retries.
     * - other 4xx: no retry.
     * - 5xx and network failures: retry up to max retries.
     * @returns the remote config info. null if failed to fetch or the response is not valid JSON.
     */
    fetch(retries?: number, timeout?: number): Promise<RemoteConfigInfo>;
    /**
     * Return jitter in the bound of [0,baseDelay) and then floor round.
     */
    getJitterDelay(baseDelay: number): number;
    getUrlParams(): string;
}
export {};
//# sourceMappingURL=remote-config.d.ts.map