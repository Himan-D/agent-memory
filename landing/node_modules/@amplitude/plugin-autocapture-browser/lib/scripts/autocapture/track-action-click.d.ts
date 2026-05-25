import { AllWindowObservables, AutoCaptureOptionsWithDefaults } from '../autocapture-plugin';
import { BrowserClient, ActionType } from '@amplitude/analytics-core';
import { shouldTrackEvent } from '../helpers';
export declare function trackActionClick({ amplitude, allObservables, options, getEventProperties, shouldTrackEvent, shouldTrackActionClick, }: {
    amplitude: BrowserClient;
    allObservables: AllWindowObservables;
    options: AutoCaptureOptionsWithDefaults;
    getEventProperties: (actionType: ActionType, element: Element) => Record<string, any>;
    shouldTrackActionClick: shouldTrackEvent;
    shouldTrackEvent: shouldTrackEvent;
}): ZenObservable.Subscription;
//# sourceMappingURL=track-action-click.d.ts.map