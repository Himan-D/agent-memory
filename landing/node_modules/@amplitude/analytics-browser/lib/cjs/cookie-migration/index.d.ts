import { Storage, UserSession } from '@amplitude/analytics-core';
export declare const parseLegacyCookies: (apiKey: string, cookieStorage: Storage<UserSession>, deleteLegacyCookies?: boolean) => Promise<UserSession>;
export declare const parseTime: (num: string) => number | undefined;
export declare const decode: (value?: string) => string | undefined;
//# sourceMappingURL=index.d.ts.map