/**
 * Dynamically loads an external script by appending a <script> tag to the document head.
 * Deduplicates by checking if a script with the same src already exists.
 */
export declare const asyncLoadScript: (url: string) => Promise<{
    status: boolean;
}>;
/**
 * Generates a simple unique ID for message request/response correlation.
 */
export declare function generateUniqueId(): string;
//# sourceMappingURL=utils.d.ts.map