export var generateHashCode = function (str) {
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
export var isTimestampInSample = function (timestamp, sampleRate) {
    var hashNumber = generateHashCode(timestamp.toString());
    var absHash = Math.abs(hashNumber);
    var absHashMultiply = absHash * 31;
    var mod = absHashMultiply % 1000000;
    return mod / 1000000 < sampleRate;
};
// TODO(xinyi): replace the temp one in diagnostics client after the fix
// istanbul ignore next
export var isTimestampInSampleTemp = function (timestamp, sampleRate) {
    var hashNumber = generateHashCode(timestamp.toString());
    var absHash = Math.abs(hashNumber);
    var absHashMultiply = absHash * 31;
    var mod = absHashMultiply % 100000;
    return mod / 100000 < sampleRate;
};
//# sourceMappingURL=sampling.js.map