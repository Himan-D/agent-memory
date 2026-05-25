import { AnalyticsIdentity, Plugin } from './types/plugin';
import { CoreClient } from './types/client/core-client';
import { IConfig } from './types/config/core-config';
import { ILogger } from './logger';
import { EventCallback } from './types/event-callback';
import { Event } from './types/event/event';
import { Result } from './types/result';
export declare class Timeline {
    private client;
    queue: [Event, EventCallback][];
    applying: boolean;
    plugins: Plugin[];
    loggerProvider: ILogger;
    _optOutListeners: ((optOut: boolean) => Promise<void>)[];
    constructor(client: CoreClient);
    register(plugin: Plugin, config: IConfig): Promise<void>;
    deregister(pluginName: string, config: IConfig): Promise<void>;
    reset(client: CoreClient): void;
    push(event: Event): Promise<Result>;
    scheduleApply(timeout: number): void;
    apply(item: [Event, EventCallback] | undefined): Promise<void>;
    flush(): Promise<void>;
    addOptOutListener(cb: (optOut: boolean) => Promise<void>): void;
    _clearOptOutListeners(): void;
    onIdentityChanged(identity: AnalyticsIdentity): void;
    onSessionIdChanged(sessionId: number): void;
    onOptOutChanged(optOut: boolean): void;
    _callOptOutListeners(optOut: boolean): Promise<void>;
    onReset(): void;
}
//# sourceMappingURL=timeline.d.ts.map