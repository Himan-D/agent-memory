import { ILogger } from '../logger';
import { IRequestWrapper, NetworkRequestEvent, IResponseWrapper } from '../network-request-event';
export type NetworkEventCallbackFn = (event: NetworkRequestEvent) => void;
export declare class NetworkEventCallback {
    readonly callback: (event: NetworkRequestEvent) => void;
    readonly id: string;
    constructor(callback: (event: NetworkRequestEvent) => void, id?: string);
}
type RequestUrlAndMethod = {
    url: string | URL | undefined;
    method: string | undefined;
};
export declare class NetworkObserver {
    private eventCallbacks;
    private globalScope?;
    private logger?;
    private isObserving;
    constructor(logger?: ILogger);
    static isSupported(): boolean;
    subscribe(eventCallback: NetworkEventCallback, logger?: ILogger): void;
    unsubscribe(eventCallback: NetworkEventCallback): void;
    protected triggerEventCallbacks(event: NetworkRequestEvent): void;
    handleNetworkRequestEvent(requestType: 'fetch' | 'xhr', requestInfo: RequestInfo | URL | RequestUrlAndMethod | undefined, requestWrapper: IRequestWrapper | undefined, responseWrapper: IResponseWrapper | undefined, typedError: Error | undefined, startTime?: number, durationStart?: number): void;
    private getTimestamps;
    private observeFetch;
    /**
     * Creates a function that parses the response of an XMLHttpRequest as JSON.
     *
     * Returns function instead of JSON object to avoid unnecessary parsing if the
     * body is not being captured.
     *
     * @param xhrSafe - The XMLHttpRequest object.
     * @param context - The NetworkObserver instance.
     * @returns A function that parses the response of an XMLHttpRequest as JSON.
     */
    static createXhrJsonParser(xhrUnsafe: XMLHttpRequest, context: NetworkObserver): () => any;
    private observeXhr;
}
export declare const networkObserver: NetworkObserver;
export {};
//# sourceMappingURL=network.d.ts.map