import { BrowserClient, BrowserConfig, EnrichmentPlugin, FrustrationInteractionsOptions, Observable } from '@amplitude/analytics-core';
import { ElementBasedTimestampedEvent, NavigateEvent, TimestampedEvent } from './helpers';
import { ObservablesEnum } from './autocapture-plugin';
import { BrowserErrorEvent } from './observables';
export interface AllWindowObservables {
    [ObservablesEnum.ClickObservable]: Observable<ElementBasedTimestampedEvent<MouseEvent>>;
    [ObservablesEnum.MutationObservable]: Observable<TimestampedEvent<MutationRecord[]>>;
    [ObservablesEnum.BrowserErrorObservable]: Observable<TimestampedEvent<BrowserErrorEvent>>;
    [ObservablesEnum.NavigateObservable]?: Observable<TimestampedEvent<NavigateEvent>>;
    [ObservablesEnum.SelectionObservable]?: Observable<void>;
    [ObservablesEnum.MouseMoveObservable]: Observable<MouseEvent>;
}
type BrowserEnrichmentPlugin = EnrichmentPlugin<BrowserClient, BrowserConfig>;
export declare const frustrationPlugin: (options?: FrustrationInteractionsOptions) => BrowserEnrichmentPlugin;
export {};
//# sourceMappingURL=frustration-plugin.d.ts.map