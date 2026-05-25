import { AllWindowObservables } from '../autocapture-plugin';
import { shouldTrackEvent, type evaluateTriggersFn } from '../helpers';
import { BrowserClient, ActionType } from '@amplitude/analytics-core';
export declare function trackChange({ amplitude, allObservables, getEventProperties, shouldTrackEvent, evaluateTriggers, }: {
    amplitude: BrowserClient;
    allObservables: AllWindowObservables;
    getEventProperties: (actionType: ActionType, element: Element) => Record<string, any>;
    shouldTrackEvent: shouldTrackEvent;
    evaluateTriggers: evaluateTriggersFn;
}): ZenObservable.Subscription;
//# sourceMappingURL=track-change.d.ts.map