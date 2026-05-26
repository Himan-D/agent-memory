"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.FetchTransport = void 0;
var tslib_1 = require("tslib");
var analytics_core_1 = require("@amplitude/analytics-core");
// Temporary browser-specific fetch transport with gzip support.
// TODO: Merge this implementation back into @amplitude/analytics-core FetchTransport
// once React Native SDK supports request body gzip.
var FetchTransport = /** @class */ (function (_super) {
    tslib_1.__extends(FetchTransport, _super);
    function FetchTransport(customHeaders) {
        if (customHeaders === void 0) { customHeaders = {}; }
        var _this = _super.call(this) || this;
        _this.customHeaders = customHeaders;
        return _this;
    }
    FetchTransport.prototype.send = function (serverUrl, payload, shouldCompressUploadBody) {
        if (shouldCompressUploadBody === void 0) { shouldCompressUploadBody = false; }
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var bodyString, shouldCompressBody, body, headers, compressed, options, response, responseText;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        /* istanbul ignore if */
                        if (typeof fetch === 'undefined') {
                            throw new Error('FetchTransport is not supported');
                        }
                        bodyString = JSON.stringify(payload);
                        shouldCompressBody = shouldCompressUploadBody &&
                            bodyString.length >= analytics_core_1.MIN_GZIP_UPLOAD_BODY_SIZE_BYTES &&
                            (0, analytics_core_1.isCompressionStreamAvailable)();
                        body = bodyString;
                        headers = {
                            'Content-Type': 'application/json',
                            Accept: '*/*',
                        };
                        if (!shouldCompressBody) return [3 /*break*/, 2];
                        return [4 /*yield*/, (0, analytics_core_1.compressToGzipArrayBuffer)(bodyString)];
                    case 1:
                        compressed = _a.sent();
                        if (compressed) {
                            headers['Content-Encoding'] = 'gzip';
                            body = compressed;
                        }
                        _a.label = 2;
                    case 2:
                        headers = tslib_1.__assign(tslib_1.__assign({}, this.customHeaders), headers);
                        options = {
                            headers: headers,
                            body: body,
                            method: 'POST',
                        };
                        return [4 /*yield*/, fetch(serverUrl, options)];
                    case 3:
                        response = _a.sent();
                        return [4 /*yield*/, response.text()];
                    case 4:
                        responseText = _a.sent();
                        try {
                            // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
                            return [2 /*return*/, this.buildResponse(JSON.parse(responseText))];
                        }
                        catch (_b) {
                            return [2 /*return*/, this.buildResponse({ code: response.status })];
                        }
                        return [2 /*return*/];
                }
            });
        });
    };
    return FetchTransport;
}(analytics_core_1.BaseTransport));
exports.FetchTransport = FetchTransport;
//# sourceMappingURL=fetch.js.map