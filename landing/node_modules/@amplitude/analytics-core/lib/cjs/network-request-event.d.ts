type BlobSafe = {
    size: number;
};
type ArrayBufferSafe = {
    byteLength: number;
};
type ArrayBufferViewSafe = {
    byteLength: number;
};
type URLSearchParamsSafe = {
    toString(): string;
};
type FormDataEntryValueSafe = string | BlobSafe | null;
type BodyInitSafe = string | Blob | ArrayBufferSafe | FormDataSafe | URLSearchParamsSafe | ArrayBufferViewSafe | null | undefined;
type HeadersRequestSafe = {
    entries(): IterableIterator<[string, string]>;
    forEach(callbackfn: (value: string, key: string) => void): void;
};
type HeadersResponseSafe = {
    get(name: string): string | null;
    forEach(callbackfn: (value: string, key: string) => void): void;
};
type HeadersInitSafe = HeadersRequestSafe | Record<string, string> | string[][];
type ResponseSafe = {
    status: number;
    headers: HeadersResponseSafe | undefined;
    clone(): ResponseCloneSafe;
};
type ResponseCloneSafe = {
    text(): Promise<string>;
};
export type RequestInitSafe = {
    method?: string;
    headers?: HeadersInitSafe;
    body?: BodyInitSafe;
};
export interface FormDataSafe {
    entries(): IterableIterator<[string, FormDataEntryValueSafe]>;
}
export type XMLHttpRequestBodyInitSafe = BlobSafe | FormDataSafe | URLSearchParamsSafe | string;
export type FetchRequestBody = string | BlobSafe | ArrayBufferSafe | FormDataSafe | URLSearchParamsSafe | ArrayBufferViewSafe | null | undefined;
export interface IRequestWrapper {
    /**
     * Get the headers of the request.
     * @param allow - The headers to allow.
     * @returns The pruned headers
     */
    headers(allow?: string[]): Record<string, string> | undefined;
    bodySize?: number;
    method?: string;
    body?: FetchRequestBody | XMLHttpRequestBodyInitSafe | null;
    json: (allow?: string[], exclude?: string[]) => Promise<JsonObject | null>;
}
export declare const MAXIMUM_ENTRIES = 100;
/**
 * This class encapsulates the RequestInit (https://developer.mozilla.org/en-US/docs/Web/API/RequestInit)
 * object so that the consumer can only get access to the headers, method and body size.
 *
 * This is to prevent consumers from directly accessing the Request object
 * and mutating it or running costly operations on it.
 *
 * IMPORTANT:
 *    * Do not make changes to this class without careful consideration
 *      of performance implications, memory usage and potential to mutate the customer's
 *      request.
 *   * NEVER .clone() the RequestInit object. This will 2x's the memory overhead of the request
 *   * NEVER: call .arrayBuffer(), text(), json() or any other method on the body that
 *     consumes the body's stream. This will cause the response to be consumed
 *     meaning the body will be empty when the customer tries to access it.
 *     (ie: if the body is an instanceof https://developer.mozilla.org/en-US/docs/Web/API/ReadableStream
 *      never call any of the methods on it)
 */
export declare class RequestWrapperFetch implements IRequestWrapper {
    private request;
    private _bodySize;
    constructor(request: RequestInitSafe);
    headers(allow?: string[]): Record<string, string> | undefined;
    get bodySize(): number | undefined;
    get method(): string | undefined;
    get body(): string | null;
    json(allow?: string[], exclude?: string[]): Promise<JsonObject | null>;
}
export declare class RequestWrapperXhr implements IRequestWrapper {
    readonly bodyRaw: XMLHttpRequestBodyInitSafe | null;
    readonly requestHeaders: Record<string, string>;
    constructor(bodyRaw: XMLHttpRequestBodyInitSafe | null, requestHeaders: Record<string, string>);
    headers(allow?: string[]): Record<string, string> | undefined;
    get bodySize(): number | undefined;
    get body(): string | null;
    json(allow?: string[], exclude?: string[]): Promise<JsonObject | null>;
}
export type JsonObject = {
    [key: string]: JsonValue;
};
export type JsonValue = string | number | boolean | null | JsonObject | JsonArray;
export type JsonArray = Array<JsonValue>;
export interface IResponseWrapper {
    /**
     * Get the headers of the response.
     * @param allow - The headers to allow.
     * @returns The pruned headers
     */
    headers(allow?: string[]): Record<string, string> | undefined;
    bodySize?: number;
    status?: number;
    body?: string | Blob | ReadableStream | ArrayBuffer | FormDataSafe | URLSearchParams | ArrayBufferView | null;
    json: (allow?: string[], exclude?: string[]) => Promise<JsonObject | null>;
}
/**
 * This class encapsulates the Fetch API Response object
 * (https://developer.mozilla.org/en-US/docs/Web/API/Response) so that the consumer can
 * only get access to the headers and body size.
 *
 * This is to prevent consumers from directly accessing the Response object
 * and mutating it or running costly operations on it.
 *
 * IMPORTANT:
 *   * Do not make changes to this class without careful consideration
 *     of performance implications, memory usage and potential to mutate the customer's
 *     response.
 *   * Do not .clone() the Response object unless you need to access the body.
 *     Cloning will 2x the memory overhead of the response.
 *   * NEVER consume the body's stream. This will cause the response to be consumed
 *     meaning the body will be empty when the customer tries to access it.
 *     (ie: if the body is an instanceof https://developer.mozilla.org/en-US/docs/Web/API/ReadableStream
 *      never call any of the methods on it)
 */
export declare class ResponseWrapperFetch implements IResponseWrapper {
    private response;
    private _bodySize;
    private clonedResponse?;
    constructor(response: ResponseSafe);
    headers(allow?: string[]): Record<string, string> | undefined;
    get bodySize(): number | undefined;
    get status(): number;
    text(): Promise<string | null>;
    json(allow?: string[], exclude?: string[]): Promise<JsonObject | null>;
}
export declare class ResponseWrapperXhr implements IResponseWrapper {
    readonly statusCode: number;
    readonly headersString: string;
    readonly size: number | undefined;
    readonly getJson: () => any | null;
    constructor(statusCode: number, headersString: string, size: number | undefined, getJson: () => any | null);
    get bodySize(): number | undefined;
    get status(): number;
    headers(allow?: string[]): Record<string, string> | undefined;
    json(allow?: string[], exclude?: string[]): Promise<JsonObject | null>;
}
export declare enum PRUNE_STRATEGY {
    REDACT = "redact",
    REMOVE = "remove"
}
/**
 * Prune headers from a headers record object.
 * @param headers - The headers to prune.
 * @param options - The options to prune the headers.
 * @param options.exclude - List of headers to delete from headers
 * @param options.include - List of headers to keep in headers, if not provided, all headers are kept by default
 * @returns The pruned headers.
 */
export declare const pruneHeaders: (headers: Record<string, string>, options: {
    allow?: string[] | undefined;
    strategy?: PRUNE_STRATEGY | undefined;
}) => Record<string, string>;
export declare class NetworkRequestEvent {
    readonly type: 'xhr' | 'fetch';
    readonly method: string;
    readonly timestamp: number;
    readonly startTime: number;
    readonly url?: string | undefined;
    readonly requestWrapper?: IRequestWrapper | undefined;
    readonly status: number;
    readonly duration?: number | undefined;
    readonly responseWrapper?: IResponseWrapper | undefined;
    readonly error?: {
        name: string;
        message: string;
    } | undefined;
    readonly endTime?: number | undefined;
    requestHeaders?: Record<string, string>;
    responseHeaders?: Record<string, string>;
    requestBodyJson?: Promise<JsonObject | null>;
    responseBodyJson?: Promise<JsonObject | null>;
    constructor(type: 'xhr' | 'fetch', method: string, timestamp: number, startTime: number, url?: string | undefined, requestWrapper?: IRequestWrapper | undefined, status?: number, duration?: number | undefined, responseWrapper?: IResponseWrapper | undefined, error?: {
        name: string;
        message: string;
    } | undefined, endTime?: number | undefined);
    toSerializable(): Record<string, any>;
}
export {};
//# sourceMappingURL=network-request-event.d.ts.map