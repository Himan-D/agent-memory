"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.omitUndefined = void 0;
var omitUndefined = function (input) {
    var obj = {};
    for (var key in input) {
        var val = input[key];
        if (val) {
            obj[key] = val;
        }
    }
    return obj;
};
exports.omitUndefined = omitUndefined;
//# sourceMappingURL=utils.js.map