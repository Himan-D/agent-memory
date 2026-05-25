import { BaseTransport, Payload, Response, Transport } from '@amplitude/analytics-core';
export declare class XHRTransport extends BaseTransport implements Transport {
    private state;
    private customHeaders;
    constructor(customHeaders?: Record<string, string>);
    send(serverUrl: string, payload: Payload, shouldCompressUploadBody?: boolean): Promise<Response | null>;
}
//# sourceMappingURL=xhr.d.ts.map