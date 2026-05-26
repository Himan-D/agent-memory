"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.decode = exports.parseTime = exports.parseLegacyCookies = void 0;
var tslib_1 = require("tslib");
var analytics_core_1 = require("@amplitude/analytics-core");
var parseLegacyCookies = function (apiKey, cookieStorage, deleteLegacyCookies) {
    if (deleteLegacyCookies === void 0) { deleteLegacyCookies = true; }
    return tslib_1.__awaiter(void 0, void 0, void 0, function () {
        var cookieName, cookies, _a, deviceId, userId, optOut, sessionId, lastEventTime, lastEventId;
        return tslib_1.__generator(this, function (_b) {
            switch (_b.label) {
                case 0:
                    cookieName = (0, analytics_core_1.getOldCookieName)(apiKey);
                    return [4 /*yield*/, cookieStorage.getRaw(cookieName)];
                case 1:
                    cookies = _b.sent();
                    if (!cookies) {
                        return [2 /*return*/, {
                                optOut: false,
                            }];
                    }
                    if (!deleteLegacyCookies) return [3 /*break*/, 3];
                    return [4 /*yield*/, cookieStorage.remove(cookieName)];
                case 2:
                    _b.sent();
                    _b.label = 3;
                case 3:
                    _a = tslib_1.__read(cookies.split('.'), 6), deviceId = _a[0], userId = _a[1], optOut = _a[2], sessionId = _a[3], lastEventTime = _a[4], lastEventId = _a[5];
                    return [2 /*return*/, {
                            deviceId: deviceId,
                            userId: (0, exports.decode)(userId),
                            sessionId: (0, exports.parseTime)(sessionId),
                            lastEventId: (0, exports.parseTime)(lastEventId),
                            lastEventTime: (0, exports.parseTime)(lastEventTime),
                            optOut: Boolean(optOut),
                        }];
            }
        });
    });
};
exports.parseLegacyCookies = parseLegacyCookies;
var parseTime = function (num) {
    var integer = parseInt(num, 32);
    if (isNaN(integer)) {
        return undefined;
    }
    return integer;
};
exports.parseTime = parseTime;
var decode = function (value) {
    if (!atob || !escape || !value) {
        return undefined;
    }
    try {
        return decodeURIComponent(escape(atob(value)));
    }
    catch (_a) {
        return undefined;
    }
};
exports.decode = decode;
//# sourceMappingURL=index.js.map