type Json = Record<string, any>;
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
export declare function pruneJson(json: Json | null | undefined, allowlist: string[], excludelist: string[]): void;
export declare function _pruneJson({ json, targetObject, allowlist, excludelist, ancestors, parentObject, targetKey, }: {
    json: Json;
    targetObject?: Json;
    allowlist: string[][];
    excludelist: string[][];
    ancestors: string[];
    parentObject?: Json;
    targetKey?: string;
}): void;
/**
 * Tokenize a JSON path string into an array of strings.
 * Escapes ~0 and ~1 to ~ and / respectively.
 *
 * e.g.) turns string "a/b/c" into ["a", "b", "c"]
 *
 * @param path - The JSON path to tokenize.
 * @returns The tokenized JSON path.
 */
export declare function tokenizeJsonPath(path: string): string[];
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
export declare function isPathMatch(path: string[], pathMatcher: string[], i?: number, j?: number): boolean;
export {};
//# sourceMappingURL=json-query.d.ts.map