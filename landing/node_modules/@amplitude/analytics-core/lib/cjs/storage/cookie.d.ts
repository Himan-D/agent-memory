import { Storage, CookieStorageOptions, CookieStorageConfig } from '../types/storage';
export declare class CookieStorage<T> implements Storage<T> {
    options: CookieStorageOptions;
    config: CookieStorageConfig;
    private static cachedTlds;
    constructor(options?: CookieStorageOptions, config?: CookieStorageConfig);
    isEnabled(): Promise<boolean>;
    get(key: string): Promise<T | undefined>;
    private decodeCookieValue;
    private getSync;
    getRaw(key: string): Promise<string | undefined>;
    private getRawSync;
    set(key: string, value: T | null): Promise<void>;
    private setSync;
    remove(key: string): Promise<void>;
    reset(): Promise<void>;
    static isDomainWritable(domain: string): Promise<boolean>;
    private transaction;
}
/**
 * Decodes a cookie value that was encoded with btoa(encodeURIComponent(...)).
 * Handles both standard encoding and double URL encoding (used by Ruby Rails v7+).
 */
export declare const decodeCookieValue: (value: string) => string | undefined;
/**
 * Compares two domain strings for equality, ignoring leading dots.
 * This is useful for comparing cookie domains since ".example.com" and "example.com"
 * are effectively equivalent for cookie scoping.
 */
export declare const isDomainEqual: (domain1: string | undefined, domain2: string | undefined) => boolean;
//# sourceMappingURL=cookie.d.ts.map