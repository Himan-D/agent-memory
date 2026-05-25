"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.isPathMatch = exports.tokenizeJsonPath = exports._pruneJson = exports.pruneJson = void 0;
var tslib_1 = require("tslib");
function isJsonPrimitive(json) {
    return (typeof json === 'string' ||
        typeof json === 'number' ||
        typeof json === 'boolean' ||
        json === null ||
        json === undefined);
}
/**
 * Prune a JSON object to only include the keys in the allowlist and excludes the keys
 * in the exclude list.
 *
 * This function is a mutative function that will modify the original JSON object.
 * This is done to avoid creating a new JSON object and copying the data.
 *
 * @param json - The JSON object to prune.
 * @param allowlist - The keys to include in the pruned JSON object.
 * @param excludelist - The keys to exclude from the pruned JSON object.
 */
function pruneJson(json, allowlist, excludelist) {
    if (!json)
        return;
    // tokenize the allowlist and excludelist
    var allowlistTokens = allowlist.map(tokenizeJsonPath);
    var excludelistTokens = excludelist.map(tokenizeJsonPath);
    _pruneJson({
        json: json,
        allowlist: allowlistTokens,
        excludelist: excludelistTokens,
        ancestors: [],
    });
}
exports.pruneJson = pruneJson;
function _pruneJson(_a) {
    var e_1, _b;
    var json = _a.json, targetObject = _a.targetObject, allowlist = _a.allowlist, excludelist = _a.excludelist, ancestors = _a.ancestors, parentObject = _a.parentObject, targetKey = _a.targetKey;
    if (!targetObject) {
        targetObject = json;
    }
    var keys = Object.keys(targetObject);
    try {
        for (var keys_1 = tslib_1.__values(keys), keys_1_1 = keys_1.next(); !keys_1_1.done; keys_1_1 = keys_1.next()) {
            var key = keys_1_1.value;
            var path = tslib_1.__spreadArray(tslib_1.__spreadArray([], tslib_1.__read(ancestors), false), [key], false);
            if (isJsonPrimitive(targetObject[key])) {
                // if the value does not match allowlist or matches exclude list, delete it
                if (!hasPathMatchInList(path, allowlist) || hasPathMatchInList(path, excludelist)) {
                    delete targetObject[key];
                }
            }
            else {
                _pruneJson({
                    json: json,
                    targetObject: targetObject[key],
                    allowlist: allowlist,
                    excludelist: excludelist,
                    ancestors: path,
                    parentObject: targetObject,
                    targetKey: key,
                });
            }
        }
    }
    catch (e_1_1) { e_1 = { error: e_1_1 }; }
    finally {
        try {
            if (keys_1_1 && !keys_1_1.done && (_b = keys_1.return)) _b.call(keys_1);
        }
        finally { if (e_1) throw e_1.error; }
    }
    // if this object is empty now, delete the whole object
    if (Object.keys(targetObject).length === 0 && parentObject && targetKey) {
        delete parentObject[targetKey];
    }
}
exports._pruneJson = _pruneJson;
/**
 * Tokenize a JSON path string into an array of strings.
 * Escapes ~0 and ~1 to ~ and / respectively.
 *
 * e.g.) turns string "a/b/c" into ["a", "b", "c"]
 *
 * @param path - The JSON path to tokenize.
 * @returns The tokenized JSON path.
 */
function tokenizeJsonPath(path) {
    if (path.startsWith('/')) {
        path = path.slice(1);
    }
    return path.split('/').map(function (token) { return token.replace(/~0/g, '~').replace(/~1/g, '/'); });
}
exports.tokenizeJsonPath = tokenizeJsonPath;
/**
 * Check if a JSON path matches a path matcher.
 *
 * Rules:
 * 1. If a key in a path and a matcher are the same, then they match, move to the next
 * 2. If the matcher is a *, then it matches the key, move to the next
 * 3. If the matcher is a **, then it matches >=0 keys
 *
 * @param path - The path to check.
 * @param pathMatcher - The path matcher to check against.
 * @param i - The current index of the path.
 * @param j - The current index of the path matcher.
 * @returns True if the path matches the path matcher, false otherwise.
 */
function isPathMatch(path, pathMatcher, i, j) {
    if (i === void 0) { i = 0; }
    if (j === void 0) { j = 0; }
    if (j === pathMatcher.length) {
        return i === path.length;
    }
    if (i === path.length) {
        while (j < pathMatcher.length && pathMatcher[j] === '**') {
            j++;
        }
        return j === pathMatcher.length;
    }
    var currentMatcher = pathMatcher[j];
    if (currentMatcher === '**') {
        if (j + 1 === pathMatcher.length) {
            return true;
        }
        for (var k = i; k <= path.length; k++) {
            if (isPathMatch(path, pathMatcher, k, j + 1)) {
                return true;
            }
        }
        return false;
    }
    else if (currentMatcher === '*' || currentMatcher === path[i]) {
        return isPathMatch(path, pathMatcher, i + 1, j + 1);
    }
    else {
        return false;
    }
}
exports.isPathMatch = isPathMatch;
/**
 * Check if a JSON path matches any of the path matchers in the allow or exclude list.
 *
 * @param path - The JSON path to check.
 * @param allowOrExcludeList - The allow or exclude list to check against.
 * @returns True if the path matches any of the path matchers in the allow or exclude list, false otherwise.
 */
function hasPathMatchInList(path, allowOrExcludeList) {
    return allowOrExcludeList.some(function (l) { return isPathMatch(path, l); });
}
//# sourceMappingURL=json-query.js.map