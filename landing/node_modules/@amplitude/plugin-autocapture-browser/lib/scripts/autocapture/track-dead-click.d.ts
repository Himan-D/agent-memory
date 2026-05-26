import { AllWindowObservables } from '../frustration-plugin';
import { BrowserClient, ActionType } from '@amplitude/analytics-core';
import { shouldTrackEvent } from '../helpers';
export declare function trackDeadClick({ amplitude, allObservables, getEventProperties, shouldTrackDeadClick, }: {
    amplitude: BrowserClient;
    allObservables: AllWindowObservables;
    getEventProperties: (actionType: ActionType, element: Element) => Record<string, any>;
    shouldTrackDeadClick: shouldTrackEvent;
}): ZenObservable.Subscription;
//# sourceMappingURL=track-dead-click.d.ts.map