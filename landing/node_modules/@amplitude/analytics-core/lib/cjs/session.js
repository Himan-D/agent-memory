"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.isNewSession = void 0;
var isNewSession = function (sessionTimeout, lastEventTime) {
    if (lastEventTime === void 0) { lastEventTime = Date.now(); }
    var currentTime = Date.now();
    var timeSinceLastEvent = currentTime - lastEventTime;
    return timeSinceLastEvent > sessionTimeout;
};
exports.isNewSession = isNewSession;
//# sourceMappingURL=session.js.map