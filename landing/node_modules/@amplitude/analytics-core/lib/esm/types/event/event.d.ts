import { BaseEvent } from './base-event';
import { RevenueEventProperties } from '../../revenue';
export declare enum IdentifyOperation {
    SET = "$set",
    SET_ONCE = "$setOnce",
    ADD = "$add",
    APPEND = "$append",
    PREPEND = "$prepend",
    REMOVE = "$remove",
    PREINSERT = "$preInsert",
    POSTINSERT = "$postInsert",
    UNSET = "$unset",
    CLEAR_ALL = "$clearAll"
}
export type ValidPropertyType = number | string | boolean | Array<string | number> | {
    [key: string]: ValidPropertyType;
} | Array<{
    [key: string]: ValidPropertyType;
}>;
interface BaseOperationConfig {
    [key: string]: ValidPropertyType;
}
export interface IdentifyUserProperties {
    [IdentifyOperation.ADD]?: {
        [key: string]: number;
    };
    [IdentifyOperation.UNSET]?: BaseOperationConfig;
    [IdentifyOperation.CLEAR_ALL]?: any;
    [IdentifyOperation.SET]?: BaseOperationConfig;
    [IdentifyOperation.SET_ONCE]?: BaseOperationConfig;
    [IdentifyOperation.APPEND]?: BaseOperationConfig;
    [IdentifyOperation.PREPEND]?: BaseOperationConfig;
    [IdentifyOperation.POSTINSERT]?: BaseOperationConfig;
    [IdentifyOperation.PREINSERT]?: BaseOperationConfig;
    [IdentifyOperation.REMOVE]?: BaseOperationConfig;
}
/**
 * Represents the structure of user properties that can be sent with an Identify or GroupIdentify event.
 *
 * This type supports both:
 *
 * 1. Reserved Amplitude identify operations via `IdentifyUserProperties`:
 *    These operations enable structured updates to user properties.
 *
 *    Example:
 *    ```ts
 *    {
 *      $set: { plan: 'premium', login_count: 1 },
 *      $add: { login_count: 1 },
 *      $unset: { plan: '-' },
 *      $clearAll: '-'
 *    }
 *    ```
 *
 * 2. Custom user-defined properties (excluding reserved operation keys):
 *    Useful for assigning static properties without using Identify operations.
 *
 *    Example:
 *    ```ts
 *    {
 *      custom_flag: true,
 *      experiment_group: 'B',
 *      favorite_color: 'blue'
 *    }
 *    ```
 *
 * This union ensures compatibility with Amplitude's identify semantics
 * while allowing flexibility to define arbitrary non-reserved properties.
 */
export type UserProperties = IdentifyUserProperties | {
    [key in Exclude<string, IdentifyOperation>]: any;
};
/**
 * Strings that have special meaning when used as an event's type
 * and have different specifications.
 */
export declare enum SpecialEventType {
    IDENTIFY = "$identify",
    GROUP_IDENTIFY = "$groupidentify",
    REVENUE = "revenue_amount"
}
export interface TrackEvent extends BaseEvent {
    event_type: Exclude<string, SpecialEventType>;
}
export interface IdentifyEvent extends BaseEvent {
    event_type: SpecialEventType.IDENTIFY;
    user_properties: UserProperties;
}
export interface GroupIdentifyEvent extends BaseEvent {
    event_type: SpecialEventType.GROUP_IDENTIFY;
    group_properties: UserProperties;
}
export interface RevenueEvent extends BaseEvent {
    event_type: SpecialEventType.REVENUE;
    event_properties: RevenueEventProperties | {
        [key: string]: any;
    };
}
export type Event = TrackEvent | IdentifyEvent | GroupIdentifyEvent | RevenueEvent;
export {};
//# sourceMappingURL=event.d.ts.map