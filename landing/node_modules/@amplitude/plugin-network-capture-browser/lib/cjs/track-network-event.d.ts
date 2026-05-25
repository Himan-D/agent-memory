import { BrowserClient, NetworkRequestEvent, NetworkTrackingOptions, ILogger, Unsubscribable } from '@amplitude/analytics-core';
import { AllWindowObservables } from './network-capture-plugin';
/**
 * Takes a user provided header capture rule and returns a
 * HeaderCaptureRule object that sets proper default values.
 *
 * @param rule - The header capture rule to parse.
 * @returns A HeaderCaptureRule object.
 */
export declare function parseHeaderCaptureRule(rule: string[] | boolean | undefined | null): string[] | undefined;
export declare function shouldTrackNetworkEvent(networkEvent: NetworkRequestEvent, options?: NetworkTrackingOptions): boolean;
export type NetworkAnalyticsEvent = {
    ['[Amplitude] URL']: string;
    ['[Amplitude] URL Query']?: string;
    ['[Amplitude] URL Fragment']?: string;
    ['[Amplitude] Request Method']: string;
    ['[Amplitude] Status Code']?: number;
    ['[Amplitude] Start Time']?: number;
    ['[Amplitude] Completion Time']?: number;
    ['[Amplitude] Duration']?: number;
    ['[Amplitude] Request Body Size']?: number;
    ['[Amplitude] Request Headers']?: Record<string, string>;
    ['[Amplitude] Request Body']?: string;
    ['[Amplitude] Response Body Size']?: number;
    ['[Amplitude] Response Headers']?: Record<string, string>;
    ['[Amplitude] Response Body']?: string;
    ['[Amplitude] Request Type']?: 'xhr' | 'fetch';
};
export declare function logNetworkAnalyticsEvent(networkAnalyticsEvent: NetworkAnalyticsEvent, request: NetworkRequestEvent, amplitude: BrowserClient, loggerProvider?: ILogger): Promise<void>;
export declare function trackNetworkEvents({ allObservables, networkTrackingOptions, amplitude, loggerProvider, }: {
    allObservables: AllWindowObservables;
    networkTrackingOptions: NetworkTrackingOptions;
    amplitude: BrowserClient;
    loggerProvider?: ILogger;
}): Unsubscribable;
//# sourceMappingURL=track-network-event.d.ts.map