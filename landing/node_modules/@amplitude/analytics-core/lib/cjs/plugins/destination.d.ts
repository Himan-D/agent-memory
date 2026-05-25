import { DestinationPlugin } from '../types/plugin';
import { Event } from '../types/event/event';
import { Result } from '../types/result';
import { Response, InvalidResponse, PayloadTooLargeResponse, RateLimitResponse, SuccessResponse } from '../types/response';
import { IConfig } from '../types/config/core-config';
import { EventCallback } from '../types/event-callback';
import { IDiagnosticsClient } from '../diagnostics/diagnostics-client';
export interface Context {
    event: Event;
    attempts: number;
    callback: EventCallback;
    timeout: number;
}
export declare function getResponseBodyString(res: Response): string;
export declare class Destination implements DestinationPlugin {
    name: string;
    type: "destination";
    retryTimeout: number;
    throttleTimeout: number;
    storageKey: string;
    config: IConfig;
    scheduleId: ReturnType<typeof setTimeout> | null;
    scheduledTimeout: number;
    flushId: ReturnType<typeof setTimeout> | null;
    queue: Context[];
    diagnosticsClient: IDiagnosticsClient | undefined;
    constructor(context?: {
        diagnosticsClient: IDiagnosticsClient;
    });
    setup(config: IConfig): Promise<undefined>;
    execute(event: Event): Promise<Result>;
    removeEventsExceedFlushMaxRetries(list: Context[]): Context[];
    scheduleEvents(list: Context[]): void;
    schedule(timeout: number): void;
    resetSchedule(): void;
    flush(useRetry?: boolean): Promise<void>;
    send(list: Context[], useRetry?: boolean): Promise<void>;
    handleResponse(res: Response, list: Context[]): void;
    handleSuccessResponse(res: SuccessResponse, list: Context[]): void;
    handleInvalidResponse(res: InvalidResponse, list: Context[]): void;
    handlePayloadTooLargeResponse(res: PayloadTooLargeResponse, list: Context[]): void;
    handleRateLimitResponse(res: RateLimitResponse, list: Context[]): void;
    handleOtherResponse(list: Context[]): void;
    fulfillRequest(list: Context[], code: number, message: string): void;
    /**
     * This is called on
     * 1) new events are added to queue; or
     * 2) response comes back for a request
     *
     * Update the event storage based on the queue
     */
    saveEvents(): void;
    /**
     * This is called on response comes back for a request
     */
    removeEvents(eventsToRemove: Context[]): void;
}
//# sourceMappingURL=destination.d.ts.map