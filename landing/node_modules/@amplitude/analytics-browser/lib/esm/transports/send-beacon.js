import { __awaiter, __extends, __generator } from "tslib";
import { BaseTransport, getGlobalScope } from '@amplitude/analytics-core';
/**
 * SendBeacon does not support custom headers (e.g. Content-Encoding: gzip),
 * so request body compression is not applied even when enableRequestBodyCompression is true.
 */
var SendBeaconTransport = /** @class */ (function (_super) {
    __extends(SendBeaconTransport, _super);
    function SendBeaconTransport() {
        return _super.call(this) || this;
    }
    SendBeaconTransport.prototype.send = function (serverUrl, payload, _enableRequestBodyCompression) {
        if (_enableRequestBodyCompression === void 0) { _enableRequestBodyCompression = false; }
        return __awaiter(this, void 0, void 0, function () {
            var _this = this;
            return __generator(this, function (_a) {
                return [2 /*return*/, new Promise(function (resolve, reject) {
                        var globalScope = getGlobalScope();
                        /* istanbul ignore if */
                        if (!(globalScope === null || globalScope === void 0 ? void 0 : globalScope.navigator.sendBeacon)) {
                            throw new Error('SendBeaconTransport is not supported');
                        }
                        try {
                            var data = JSON.stringify(payload);
                            // SendBeacon cannot set Content-Encoding, so we always send uncompressed
                            var success = globalScope.navigator.sendBeacon(serverUrl, data);
                            if (success) {
                                return resolve(_this.buildResponse({
                                    code: 200,
                                    events_ingested: payload.events.length,
                                    payload_size_bytes: data.length,
                                    server_upload_time: Date.now(),
                                }));
                            }
                            return resolve(_this.buildResponse({ code: 500 }));
                        }
                        catch (e) {
                            reject(e);
                        }
                    })];
            });
        });
    };
    return SendBeaconTransport;
}(BaseTransport));
export { SendBeaconTransport };
//# sourceMappingURL=send-beacon.js.map