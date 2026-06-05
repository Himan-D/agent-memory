import { BeforePlugin } from '../types/plugin';
import { IConfig } from '../types/config/core-config';
import { Event } from '../types/event/event';
export declare class IdentityEventSender implements BeforePlugin {
    name: string;
    type: "before";
    identityStore: import("@amplitude/analytics-connector/dist/types/src/identityStore").IdentityStoreImpl;
    execute(context: Event): Promise<Event>;
    setup(config: IConfig): Promise<void>;
}
//# sourceMappingURL=identity.d.ts.map