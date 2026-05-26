"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.safeJsonStringify = void 0;
var tslib_1 = require("tslib");
/* eslint-disable @typescript-eslint/no-unsafe-assignment */
/* eslint-disable @typescript-eslint/no-unsafe-member-access */
var safeJsonStringifyModule = tslib_1.__importStar(require("safe-json-stringify"));
/* istanbul ignore next */
var safeJsonStringify = safeJsonStringifyModule.default || safeJsonStringifyModule;
exports.safeJsonStringify = safeJsonStringify;
//# sourceMappingURL=safe-stringify.js.map