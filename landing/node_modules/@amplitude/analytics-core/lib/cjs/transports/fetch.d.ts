import { BaseTransport } from './base';
import { Transport } from '../types/transport';
import { Payload } from '../types/payload';
import { Response } from '../types/response';
export declare class FetchTransport extends BaseTransport implements Transport {
    private customHeaders;
    constructor(customHeaders?: Record<string, string>);
    send(serverUrl: string, payload: Payload): Promise<Response | null>;
}
//# sourceMappingURL=fetch.d.ts.map