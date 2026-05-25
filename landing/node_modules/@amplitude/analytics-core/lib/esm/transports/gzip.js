import { __awaiter, __generator } from "tslib";
export var MIN_GZIP_UPLOAD_BODY_SIZE_BYTES = 2 * 1024;
/**
 * Returns true if CompressionStream is available (e.g. in supported browsers).
 */
export function isCompressionStreamAvailable() {
    return typeof CompressionStream !== 'undefined';
}
/**
 * Compress a string to gzip and return the result as an ArrayBuffer.
 * Best-effort: returns undefined if CompressionStream is unavailable or compression fails.
 * Payload is small so buffering is fine. Used by Fetch and XHR transports.
 */
export function compressToGzipArrayBuffer(data) {
    return __awaiter(this, void 0, void 0, function () {
        var CompressionStreamImpl, stream, _a;
        return __generator(this, function (_b) {
            switch (_b.label) {
                case 0:
                    CompressionStreamImpl = CompressionStream;
                    if (typeof CompressionStreamImpl === 'undefined') {
                        return [2 /*return*/, undefined];
                    }
                    _b.label = 1;
                case 1:
                    _b.trys.push([1, 3, , 4]);
                    stream = new Blob([data]).stream().pipeThrough(new CompressionStreamImpl('gzip'));
                    return [4 /*yield*/, new Response(stream).arrayBuffer()];
                case 2: return [2 /*return*/, _b.sent()];
                case 3:
                    _a = _b.sent();
                    return [2 /*return*/, undefined];
                case 4: return [2 /*return*/];
            }
        });
    });
}
//# sourceMappingURL=gzip.js.map