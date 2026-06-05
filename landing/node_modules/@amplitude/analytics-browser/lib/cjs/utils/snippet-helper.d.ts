import { Result } from '@amplitude/analytics-core';
interface ProxyItem {
    name: string;
    args: any[];
    resolve?: (promise: Promise<Result>) => void;
}
type QueueProxy = Array<ProxyItem>;
interface InstanceProxy {
    _q: QueueProxy;
    _iq: Record<string, InstanceProxy>;
}
/**
 * Applies the proxied functions on the proxied amplitude snippet to an instance of the real object.
 * @ignore
 */
export declare const runQueuedFunctions: (instance: object, queue: QueueProxy) => void;
/**
 * Applies the proxied functions on the proxied object to an instance of the real object.
 * Used to convert proxied Identify and Revenue objects.
 */
export declare const convertProxyObjectToRealObject: <T>(instance: T, queue: QueueProxy) => T;
/**
 * Check if the param is snippet proxy
 */
export declare const isInstanceProxy: (instance: unknown) => instance is InstanceProxy;
export {};
//# sourceMappingURL=snippet-helper.d.ts.map