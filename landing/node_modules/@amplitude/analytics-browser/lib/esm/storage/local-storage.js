import { __awaiter, __extends, __generator } from "tslib";
import { getGlobalScope, BrowserStorage } from '@amplitude/analytics-core';
var MAX_ARRAY_LENGTH = 1000;
var LocalStorage = /** @class */ (function (_super) {
    __extends(LocalStorage, _super);
    function LocalStorage(config) {
        var _this = this;
        var _a, _b;
        var localStorage;
        try {
            localStorage = (_a = getGlobalScope()) === null || _a === void 0 ? void 0 : _a.localStorage;
        }
        catch (e) {
            (_b = config === null || config === void 0 ? void 0 : config.loggerProvider) === null || _b === void 0 ? void 0 : _b.debug("Failed to access localStorage. error=".concat(JSON.stringify(e)));
            localStorage = undefined;
        }
        _this = _super.call(this, localStorage) || this;
        _this.loggerProvider = config === null || config === void 0 ? void 0 : config.loggerProvider;
        return _this;
    }
    LocalStorage.prototype.set = function (key, value) {
        var _a;
        return __awaiter(this, void 0, void 0, function () {
            var droppedEventsCount;
            return __generator(this, function (_b) {
                switch (_b.label) {
                    case 0:
                        if (!(Array.isArray(value) && value.length > MAX_ARRAY_LENGTH)) return [3 /*break*/, 2];
                        droppedEventsCount = value.length - MAX_ARRAY_LENGTH;
                        return [4 /*yield*/, _super.prototype.set.call(this, key, value.slice(0, MAX_ARRAY_LENGTH))];
                    case 1:
                        _b.sent();
                        (_a = this.loggerProvider) === null || _a === void 0 ? void 0 : _a.error("Failed to save ".concat(droppedEventsCount, " events because the queue length exceeded ").concat(MAX_ARRAY_LENGTH, "."));
                        return [3 /*break*/, 4];
                    case 2: return [4 /*yield*/, _super.prototype.set.call(this, key, value)];
                    case 3:
                        _b.sent();
                        _b.label = 4;
                    case 4: return [2 /*return*/];
                }
            });
        });
    };
    return LocalStorage;
}(BrowserStorage));
export { LocalStorage };
//# sourceMappingURL=local-storage.js.map