import { Transport } from '../types/transport';
import { Status } from '../types/status';
import { Payload } from '../types/payload';
import { Response } from '../types/response';
export declare class BaseTransport implements Transport {
    send(_serverUrl: string, _payload: Payload, _enableRequestBodyCompression?: boolean): Promise<Response | null>;
    buildResponse(responseJSON: Record<string, any>): Response | null;
    buildStatus(code: number): Status;
}
//# sourceMappingURL=base.d.ts.map