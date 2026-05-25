import { ILogger } from '../logger';
/**
 * Checks if a given URL matches any pattern in an allowlist of URLs or regex patterns.
 * @param url - The URL to check
 * @param allowlist - Array of allowed URLs (strings) or regex patterns
 * @returns true if the URL matches any pattern in the allowlist, false otherwise
 */
export declare const isUrlMatchAllowlist: (url: string, allowlist: (string | RegExp)[] | undefined) => boolean;
export declare const getDecodeURI: (locationStr: string, loggerProvider?: ILogger) => string;
//# sourceMappingURL=url-utils.d.ts.map