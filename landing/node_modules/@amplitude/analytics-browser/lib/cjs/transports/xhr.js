"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.XHRTransport = void 0;
var tslib_1 = require("tslib");
var analytics_core_1 = require("@amplitude/analytics-core");
var XHRTransport = /** @class */ (function (_super) {
    tslib_1.__extends(XHRTransport, _super);
    function XHRTransport(customHeaders) {
        if (customHeaders === void 0) { customHeaders = {}; }
        var _this = _super.call(this) || this;
        _this.state = {
            done: 4,
        };
        _this.customHeaders = customHeaders;
        return _this;
    }
    XHRTransport.prototype.send = function (serverUrl, payload, shouldCompressUploadBody) {
        if (shouldCompressUploadBody === void 0) { shouldCompressUploadBody = false; }
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var _this = this;
            return tslib_1.__generator(this, function (_a) {
                return [2 /*return*/, new Promise(function (resolve, reject) {
                        /* istanbul ignore if */
                        if (typeof XMLHttpRequest === 'undefined') {
                            reject(new Error('XHRTransport is not supported.'));
                        }
                        var xhr = new XMLHttpRequest();
                        xhr.open('POST', serverUrl, true);
                        xhr.onreadystatechange = function () {
                            if (xhr.readyState === _this.state.done) {
                                var responseText = xhr.responseText;
                                try {
                                    // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
                                    resolve(_this.buildResponse(JSON.parse(responseText)));
                                }
                                catch (_a) {
                                    resolve(_this.buildResponse({ code: xhr.status }));
                                }
                            }
                        };
                        var headers = {
                            'Content-Type': 'application/json',
                            Accept: '*/*',
                        };
                        var bodyString = JSON.stringify(payload);
                        var shouldCompressBody = shouldCompressUploadBody &&
                            bodyString.length >= analytics_core_1.MIN_GZIP_UPLOAD_BODY_SIZE_BYTES &&
                            (0, analytics_core_1.isCompressionStreamAvailable)();
                        var sendBody = function (body) {
                            var e_1, _a;
                            headers = tslib_1.__assign(tslib_1.__assign({}, _this.customHeaders), headers);
                            try {
                                for (var _b = tslib_1.__values(Object.entries(headers)), _c = _b.next(); !_c.done; _c = _b.next()) {
                                    var _d = tslib_1.__read(_c.value, 2), key = _d[0], value = _d[1];
                                    xhr.setRequestHeader(key, value);
                                }
                            }
                            catch (e_1_1) { e_1 = { error: e_1_1 }; }
                            finally {
                                try {
                                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                                }
                                finally { if (e_1) throw e_1.error; }
                            }
                            xhr.send(body);
                        };
                        var doSend = function () { return tslib_1.__awaiter(_this, void 0, void 0, function () {
                            var compressed;
                            return tslib_1.__generator(this, function (_a) {
                                switch (_a.label) {
                                    case 0:
                                        if (!shouldCompressBody) return [3 /*break*/, 2];
                                        return [4 /*yield*/, (0, analytics_core_1.compressToGzipArrayBuffer)(bodyString)];
                                    case 1:
                                        compressed = _a.sent();
                                        if (compressed) {
                                            headers['Content-Encoding'] = 'gzip';
                                            sendBody(compressed);
                                        }
                                        else {
                                            sendBody(bodyString);
                                        }
                                        return [3 /*break*/, 3];
                                    case 2:
                                        sendBody(bodyString);
                                        _a.label = 3;
                                    case 3: return [2 /*return*/];
                                }
                            });
                        }); };
                        doSend().catch(reject);
                    })];
            });
        });
    };
    return XHRTransport;
}(analytics_core_1.BaseTransport));
exports.XHRTransport = XHRTransport;
//# sourceMappingURL=xhr.js.map