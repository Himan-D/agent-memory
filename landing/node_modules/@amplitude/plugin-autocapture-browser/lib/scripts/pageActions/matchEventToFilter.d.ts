import type { Filter } from '@amplitude/analytics-core/lib/esm/types/element-interactions';
import { ElementBasedTimestampedEvent, ElementBasedEvent } from '../helpers';
/**
 * Matches an event to a single filter
 * @param event - The event to match
 * @param filter - The filter to match against
 * @returns boolean indicating if the event matches the filter
 */
export declare const matchEventToFilter: (event: ElementBasedTimestampedEvent<ElementBasedEvent>, filter: Filter) => boolean;
//# sourceMappingURL=matchEventToFilter.d.ts.map