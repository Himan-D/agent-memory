import { BaseTransport, Payload, Response, Transport } from '@amplitude/analytics-core';
/**
 * SendBeacon does not support custom headers (e.g. Content-Encoding: gzip),
 * so request body compression is not applied even when enableRequestBodyCompression is true.
 */
export declare class SendBeaconTransport extends BaseTransport implements Transport {
    constructor();
    send(serverUrl: string, payload: Payload, _enableRequestBodyCompression?: boolean): Promise<Response | null>;
}
//# sourceMappingURL=send-beacon.d.ts.map