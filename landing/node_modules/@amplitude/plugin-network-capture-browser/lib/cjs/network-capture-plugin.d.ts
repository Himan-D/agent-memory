import { BrowserClient, BrowserConfig, EnrichmentPlugin, NetworkRequestEvent, NetworkTrackingOptions } from '@amplitude/analytics-core';
import { Observable } from '@amplitude/analytics-core';
export type BrowserEnrichmentPlugin = EnrichmentPlugin<BrowserClient, BrowserConfig>;
export declare enum ObservablesEnum {
    NetworkObservable = "networkObservable"
}
type BaseTimestampedEvent<T> = {
    event: T;
    timestamp: number;
    type: 'rage' | 'click' | 'change' | 'error' | 'navigate' | 'mutation' | 'network';
};
export type ElementBasedEvent = MouseEvent | Event;
export type ElementBasedTimestampedEvent<T> = BaseTimestampedEvent<T> & {
    event: MouseEvent | Event;
    type: 'click' | 'change';
    closestTrackedAncestor: Element;
    targetElementProperties: Record<string, any>;
};
export type TimestampedEvent<T> = BaseTimestampedEvent<T> | ElementBasedTimestampedEvent<T>;
export interface AllWindowObservables {
    [ObservablesEnum.NetworkObservable]: Observable<TimestampedEvent<NetworkRequestEvent>>;
}
export declare const networkCapturePlugin: (options?: NetworkTrackingOptions) => BrowserEnrichmentPlugin;
export {};
//# sourceMappingURL=network-capture-plugin.d.ts.map