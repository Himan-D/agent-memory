"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.SessionStorage = void 0;
var tslib_1 = require("tslib");
var analytics_core_1 = require("@amplitude/analytics-core");
var SessionStorage = /** @class */ (function (_super) {
    tslib_1.__extends(SessionStorage, _super);
    function SessionStorage() {
        var _a;
        return _super.call(this, (_a = (0, analytics_core_1.getGlobalScope)()) === null || _a === void 0 ? void 0 : _a.sessionStorage) || this;
    }
    return SessionStorage;
}(analytics_core_1.BrowserStorage));
exports.SessionStorage = SessionStorage;
//# sourceMappingURL=session-storage.js.map