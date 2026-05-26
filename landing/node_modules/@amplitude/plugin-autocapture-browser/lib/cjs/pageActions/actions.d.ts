import type { DataSource, PageAction } from '@amplitude/analytics-core/lib/esm/types/element-interactions';
import type { DataExtractor } from '../data-extractor';
import type { ElementBasedTimestampedEvent, ElementBasedEvent } from '../helpers';
/**
 * Gets the DOM element specified by the data source configuration
 * @param dataSource - Configuration for finding the target element
 * @param contextElement - The element to start searching from
 * @returns The matching DOM element or undefined if not found
 */
export declare const getDataSource: (dataSource: DataSource, contextElement: HTMLElement) => Element | null | undefined;
export declare const executeActions: (actions: (string | PageAction)[], ev: ElementBasedTimestampedEvent<ElementBasedEvent>, dataExtractor: DataExtractor) => void;
//# sourceMappingURL=actions.d.ts.map