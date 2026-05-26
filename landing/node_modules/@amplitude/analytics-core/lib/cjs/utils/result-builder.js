"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buildResult = void 0;
var status_1 = require("../types/status");
var buildResult = function (event, code, message) {
    if (code === void 0) { code = 0; }
    if (message === void 0) { message = status_1.Status.Unknown; }
    return { event: event, code: code, message: message };
};
exports.buildResult = buildResult;
//# sourceMappingURL=result-builder.js.map