export declare const MIN_GZIP_UPLOAD_BODY_SIZE_BYTES: number;
/**
 * Returns true if CompressionStream is available (e.g. in supported browsers).
 */
export declare function isCompressionStreamAvailable(): boolean;
/**
 * Compress a string to gzip and return the result as an ArrayBuffer.
 * Best-effort: returns undefined if CompressionStream is unavailable or compression fails.
 * Payload is small so buffering is fine. Used by Fetch and XHR transports.
 */
export declare function compressToGzipArrayBuffer(data: string): Promise<ArrayBuffer | undefined>;
//# sourceMappingURL=gzip.d.ts.map