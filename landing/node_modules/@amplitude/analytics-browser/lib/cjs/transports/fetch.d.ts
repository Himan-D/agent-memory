import { BaseTransport, Payload, Response, Transport } from '@amplitude/analytics-core';
export declare class FetchTransport extends BaseTransport implements Transport {
    private customHeaders;
    constructor(customHeaders?: Record<string, string>);
    send(serverUrl: string, payload: Payload, shouldCompressUploadBody?: boolean): Promise<Response | null>;
}
//# sourceMappingURL=fetch.d.ts.map