"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.BrowserStorage = void 0;
var tslib_1 = require("tslib");
var BrowserStorage = /** @class */ (function () {
    function BrowserStorage(storage) {
        this.storage = storage;
    }
    BrowserStorage.prototype.isEnabled = function () {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var random, testStorage, testKey, value, _a;
            return tslib_1.__generator(this, function (_b) {
                switch (_b.label) {
                    case 0:
                        /* istanbul ignore if */
                        if (!this.storage) {
                            return [2 /*return*/, false];
                        }
                        random = String(Date.now());
                        testStorage = new BrowserStorage(this.storage);
                        testKey = 'AMP_TEST';
                        _b.label = 1;
                    case 1:
                        _b.trys.push([1, 4, 5, 7]);
                        return [4 /*yield*/, testStorage.set(testKey, random)];
                    case 2:
                        _b.sent();
                        return [4 /*yield*/, testStorage.get(testKey)];
                    case 3:
                        value = _b.sent();
                        return [2 /*return*/, value === random];
                    case 4:
                        _a = _b.sent();
                        /* istanbul ignore next */
                        return [2 /*return*/, false];
                    case 5: return [4 /*yield*/, testStorage.remove(testKey)];
                    case 6:
                        _b.sent();
                        return [7 /*endfinally*/];
                    case 7: return [2 /*return*/];
                }
            });
        });
    };
    BrowserStorage.prototype.get = function (key) {
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            var value, _a;
            return tslib_1.__generator(this, function (_b) {
                switch (_b.label) {
                    case 0:
                        _b.trys.push([0, 2, , 3]);
                        return [4 /*yield*/, this.getRaw(key)];
                    case 1:
                        value = _b.sent();
                        if (!value) {
                            return [2 /*return*/, undefined];
                        }
                        // eslint-disable-next-line @typescript-eslint/no-unsafe-return
                        return [2 /*return*/, JSON.parse(value)];
                    case 2:
                        _a = _b.sent();
                        console.error("[Amplitude] Error: Could not get value from storage");
                        return [2 /*return*/, undefined];
                    case 3: return [2 /*return*/];
                }
            });
        });
    };
    BrowserStorage.prototype.getRaw = function (key) {
        var _a;
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            return tslib_1.__generator(this, function (_b) {
                return [2 /*return*/, ((_a = this.storage) === null || _a === void 0 ? void 0 : _a.getItem(key)) || undefined];
            });
        });
    };
    BrowserStorage.prototype.set = function (key, value) {
        var _a;
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            return tslib_1.__generator(this, function (_b) {
                try {
                    (_a = this.storage) === null || _a === void 0 ? void 0 : _a.setItem(key, JSON.stringify(value));
                }
                catch (_c) {
                    //
                }
                return [2 /*return*/];
            });
        });
    };
    BrowserStorage.prototype.remove = function (key) {
        var _a;
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            return tslib_1.__generator(this, function (_b) {
                try {
                    (_a = this.storage) === null || _a === void 0 ? void 0 : _a.removeItem(key);
                }
                catch (_c) {
                    //
                }
                return [2 /*return*/];
            });
        });
    };
    BrowserStorage.prototype.reset = function () {
        var _a;
        return tslib_1.__awaiter(this, void 0, void 0, function () {
            return tslib_1.__generator(this, function (_b) {
                try {
                    (_a = this.storage) === null || _a === void 0 ? void 0 : _a.clear();
                }
                catch (_c) {
                    //
                }
                return [2 /*return*/];
            });
        });
    };
    return BrowserStorage;
}());
exports.BrowserStorage = BrowserStorage;
//# sourceMappingURL=browser-storage.js.map