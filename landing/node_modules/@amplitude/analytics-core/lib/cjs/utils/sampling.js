"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.isTimestampInSampleTemp = exports.isTimestampInSample = exports.generateHashCode = void 0;
var generateHashCode = function (str) {
    var hash = 0;
    if (str.length === 0)
        return hash;
    for (var i = 0; i < str.length; i++) {
        var chr = str.charCodeAt(i);
        hash = (hash << 5) - hash + chr;
        hash |= 0;
    }
    return hash;
};
exports.generateHashCode = generateHashCode;
var isTimestampInSample = function (timestamp, sampleRate) {
    var hashNumber = (0, exports.generateHashCode)(timestamp.toString());
    var absHash = Math.abs(hashNumber);
    var absHashMultiply = absHash * 31;
    var mod = absHashMultiply % 1000000;
    return mod / 1000000 < sampleRate;
};
exports.isTimestampInSample = isTimestampInSample;
// TODO(xinyi): replace the temp one in diagnostics client after the fix
// istanbul ignore next
var isTimestampInSampleTemp = function (timestamp, sampleRate) {
    var hashNumber = (0, exports.generateHashCode)(timestamp.toString());
    var absHash = Math.abs(hashNumber);
    var absHashMultiply = absHash * 31;
    var mod = absHashMultiply % 100000;
    return mod / 100000 < sampleRate;
};
exports.isTimestampInSampleTemp = isTimestampInSampleTemp;
//# sourceMappingURL=sampling.js.map