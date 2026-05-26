import { Event, BeforePlugin, BrowserConfig } from '@amplitude/analytics-core';
export declare class Context implements BeforePlugin {
    name: string;
    type: "before";
    config: BrowserConfig;
    userAgent: string | undefined;
    library: string;
    constructor();
    setup(config: BrowserConfig): Promise<undefined>;
    execute(context: Event): Promise<Event>;
}
//# sourceMappingURL=context.d.ts.map