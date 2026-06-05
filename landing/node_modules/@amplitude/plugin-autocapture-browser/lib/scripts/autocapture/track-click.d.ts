import { AllWindowObservables } from '../autocapture-plugin';
import { type evaluateTriggersFn } from '../helpers';
import { BrowserClient } from '@amplitude/analytics-core';
import { shouldTrackEvent } from '../helpers';
export declare function trackClicks({ amplitude, allObservables, shouldTrackEvent, evaluateTriggers, }: {
    amplitude: BrowserClient;
    allObservables: AllWindowObservables;
    shouldTrackEvent: shouldTrackEvent;
    evaluateTriggers: evaluateTriggersFn;
}): ZenObservable.Subscription;
//# sourceMappingURL=track-click.d.ts.map