import { __awaiter, __generator, __read } from "tslib";
import { getOldCookieName } from '@amplitude/analytics-core';
export var parseLegacyCookies = function (apiKey, cookieStorage, deleteLegacyCookies) {
    if (deleteLegacyCookies === void 0) { deleteLegacyCookies = true; }
    return __awaiter(void 0, void 0, void 0, function () {
        var cookieName, cookies, _a, deviceId, userId, optOut, sessionId, lastEventTime, lastEventId;
        return __generator(this, function (_b) {
            switch (_b.label) {
                case 0:
                    cookieName = getOldCookieName(apiKey);
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
                    _a = __read(cookies.split('.'), 6), deviceId = _a[0], userId = _a[1], optOut = _a[2], sessionId = _a[3], lastEventTime = _a[4], lastEventId = _a[5];
                    return [2 /*return*/, {
                            deviceId: deviceId,
                            userId: decode(userId),
                            sessionId: parseTime(sessionId),
                            lastEventId: parseTime(lastEventId),
                            lastEventTime: parseTime(lastEventTime),
                            optOut: Boolean(optOut),
                        }];
            }
        });
    });
};
export var parseTime = function (num) {
    var integer = parseInt(num, 32);
    if (isNaN(integer)) {
        return undefined;
    }
    return integer;
};
export var decode = function (value) {
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
//# sourceMappingURL=index.js.map