import { type BrowserClient, type BrowserConfig, type EnrichmentPlugin, type ElementInteractionsOptions, IDiagnosticsClient } from '@amplitude/analytics-core';
import { type ElementBasedTimestampedEvent, type TimestampedEvent, type NavigateEvent } from './helpers';
import { Observable } from '@amplitude/analytics-core';
type NavigationType = {
    addEventListener: (type: string, listener: EventListenerOrEventListenerObject) => void;
    removeEventListener: (type: string, listener: EventListenerOrEventListenerObject) => void;
};
declare global {
    interface Window {
        navigation: NavigationType;
    }
}
type BrowserEnrichmentPlugin = EnrichmentPlugin<BrowserClient, BrowserConfig>;
export type AutoCaptureOptionsWithDefaults = Required<Pick<ElementInteractionsOptions, 'debounceTime' | 'cssSelectorAllowlist' | 'actionClickAllowlist'>> & ElementInteractionsOptions;
export declare enum ObservablesEnum {
    ClickObservable = "clickObservable",
    ChangeObservable = "changeObservable",
    NavigateObservable = "navigateObservable",
    MutationObservable = "mutationObservable",
    ScrollObservable = "scrollObservable",
    ExposureObservable = "exposureObservable",
    BrowserErrorObservable = "browserErrorObservable",
    SelectionObservable = "selectionObservable",
    MouseMoveObservable = "mouseMoveObservable"
}
export interface AllWindowObservables {
    [ObservablesEnum.ChangeObservable]: Observable<ElementBasedTimestampedEvent<Event>>;
    [ObservablesEnum.ClickObservable]: Observable<ElementBasedTimestampedEvent<MouseEvent>>;
    [ObservablesEnum.MutationObservable]: Observable<TimestampedEvent<MutationRecord[]>>;
    [ObservablesEnum.NavigateObservable]?: Observable<TimestampedEvent<NavigateEvent>>;
    [ObservablesEnum.ScrollObservable]: Observable<Event>;
    [ObservablesEnum.ExposureObservable]: Observable<Event>;
    [ObservablesEnum.SelectionObservable]?: Observable<void>;
}
export declare const autocapturePlugin: (options?: ElementInteractionsOptions, context?: {
    diagnosticsClient: IDiagnosticsClient;
}) => BrowserEnrichmentPlugin;
export {};
//# sourceMappingURL=autocapture-plugin.d.ts.map