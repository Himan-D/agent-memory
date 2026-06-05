"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.FetchTransport = void 0;
var tslib_1 = require("tslib");
var base_1 = require("./base");
var FetchTransport = /** @class */ (function (_super) {
    tslib_1.__extends(FetchTransport, _super);
    function FetchTransport(customHeaders) {
        if (customHeaders === void 0) { customHeaders = {}; }
        var _this = _super.call(this) || this;
        _this.customHeaders = customHeaders;
        return _this;
    }
    FetchTransport.prototype.send = function (serverUrl, payload) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var options, response, responseText;
            return tslib_1.__generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        /* istanbul ignore if */
                        if (typeof fetch === 'undefined') {
                            throw new Error('FetchTransport is not supported');
                        }
                        options = {
                            headers: tslib_1.__assign({ 'Content-Type': 'application/json', Accept: '*/*' }, this.customHeaders),
                            body: JSON.stringify(payload),
                            method: 'POST',
                        };
                        return [4 /*yield*/, fetch(serverUrl, options)];
                    case 1:
                        response = _a.sent();
                        return [4 /*yield*/, response.text()];
                    case 2:
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
}(base_1.BaseTransport));
exports.FetchTransport = FetchTransport;
//# sourceMappingURL=fetch.js.map