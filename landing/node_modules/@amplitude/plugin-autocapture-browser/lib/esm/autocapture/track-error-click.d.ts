import { AllWindowObservables } from '../frustration-plugin';
import { shouldTrackEvent } from '../helpers';
import { BrowserClient } from '@amplitude/analytics-core';
export declare function trackErrorClicks({ amplitude, allObservables, shouldTrackErrorClick, }: {
    amplitude: BrowserClient;
    allObservables: AllWindowObservables;
    shouldTrackErrorClick: shouldTrackEvent;
}): ZenObservable.Subscription;
//# sourceMappingURL=track-error-click.d.ts.map