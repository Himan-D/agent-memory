import { RemoteConfigStorage, RemoteConfigInfo } from './remote-config';
import { ILogger } from '../logger';
export declare class RemoteConfigLocalStorage implements RemoteConfigStorage {
    private readonly key;
    private readonly logger;
    constructor(apiKey: string, logger: ILogger);
    fetchConfig(): Promise<RemoteConfigInfo>;
    setConfig(config: RemoteConfigInfo): Promise<boolean>;
}
//# sourceMappingURL=remote-config-localstorage.d.ts.map