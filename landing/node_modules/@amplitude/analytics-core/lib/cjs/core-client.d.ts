import { Plugin } from './types/plugin';
import { IConfig } from './types/config/core-config';
import { BaseEvent, EventOptions } from './types/event/base-event';
import { Result } from './types/result';
import { Event, UserProperties } from './types/event/event';
import { IIdentify } from './identify';
import { IRevenue } from './revenue';
import { Timeline } from './timeline';
import { returnWrapper } from './utils/return-wrapper';
import { CoreClient, PluginHost } from './types/client/core-client';
export declare class AmplitudeCore implements CoreClient, PluginHost {
    protected initializing: boolean;
    protected name: string;
    config: IConfig;
    timeline: Timeline;
    isReady: boolean;
    protected q: Array<CallableFunction | typeof returnWrapper>;
    protected dispatchQ: Array<CallableFunction>;
    /**
     * Current user properties (operation-applied, flat key-value).
     * Only used by client-side SDKs; server SDKs may leave this undefined.
     */
    userProperties: {
        [key: string]: any;
    } | undefined;
    constructor(name?: string);
    protected _init(config: IConfig): Promise<void>;
    runQueuedFunctions(queueName: 'q' | 'dispatchQ'): Promise<void>;
    track(eventInput: BaseEvent | string, eventProperties?: Record<string, any>, eventOptions?: EventOptions): import("./utils/return-wrapper").AmplitudeReturn<Result>;
    logEvent: (eventInput: BaseEvent | string, eventProperties?: Record<string, any>, eventOptions?: EventOptions) => import("./utils/return-wrapper").AmplitudeReturn<Result>;
    identify(identify: IIdentify, eventOptions?: EventOptions): import("./utils/return-wrapper").AmplitudeReturn<Result>;
    groupIdentify(groupType: string, groupName: string | string[], identify: IIdentify, eventOptions?: EventOptions): import("./utils/return-wrapper").AmplitudeReturn<Result>;
    setGroup(groupType: string, groupName: string | string[], eventOptions?: EventOptions): import("./utils/return-wrapper").AmplitudeReturn<Result>;
    revenue(revenue: IRevenue, eventOptions?: EventOptions): import("./utils/return-wrapper").AmplitudeReturn<Result>;
    add(plugin: Plugin): import("./utils/return-wrapper").AmplitudeReturn<void>;
    _addPlugin(plugin: Plugin): import("./utils/return-wrapper").AmplitudeReturn<void>;
    remove(pluginName: string): import("./utils/return-wrapper").AmplitudeReturn<void>;
    _removePlugin(pluginName: string): import("./utils/return-wrapper").AmplitudeReturn<void>;
    dispatchWithCallback(event: Event, callback: (result: Result) => void): void;
    dispatch(event: Event): Promise<Result>;
    /**
     *
     * This method applies identify operations to user properties and
     * returns a single object representing the final user property state.
     *
     * This is a best-effort api that only supports $set, $clearAll, and $unset.
     * Other operations are not supported and are ignored.
     *
     * Operations are applied on top of current client state (this.userProperties).
     *
     * @param userProperties The new user properties object from identify() or setIdentity().
     * @returns A key-value object user properties without operations.
     *
     * @example
     * Input:
     * {
     *   $set: { plan: 'premium' },
     *   custom_flag: true
     * }
     *
     * Output:
     * {
     *   plan: 'premium',
     *   custom_flag: true
     * }
     */
    getOperationAppliedUserProperties(userProperties: UserProperties | undefined): {
        [key: string]: any;
    };
    process(event: Event): Promise<Result>;
    setOptOut(optOut: boolean): void;
    _setOptOut(optOut: boolean): void;
    flush(): import("./utils/return-wrapper").AmplitudeReturn<void>;
    plugin(name: string): Plugin | undefined;
    plugins<T extends Plugin>(pluginClass: {
        new (...args: any[]): T;
    }): T[];
}
//# sourceMappingURL=core-client.d.ts.map