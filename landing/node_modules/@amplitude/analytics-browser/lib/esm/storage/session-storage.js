import { __extends } from "tslib";
import { getGlobalScope, BrowserStorage } from '@amplitude/analytics-core';
var SessionStorage = /** @class */ (function (_super) {
    __extends(SessionStorage, _super);
    function SessionStorage() {
        var _a;
        return _super.call(this, (_a = getGlobalScope()) === null || _a === void 0 ? void 0 : _a.sessionStorage) || this;
    }
    return SessionStorage;
}(BrowserStorage));
export { SessionStorage };
//# sourceMappingURL=session-storage.js.map