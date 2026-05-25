import type { HierarchyNode } from './typings/autocapture';
export declare const MAX_HIERARCHY_LENGTH = 1024;
export declare function getElementProperties(element: Element | null, userMaskedAttributeNames: Set<string>): HierarchyNode | null;
export declare function getAncestors(targetEl: Element | null): Element[];
//# sourceMappingURL=hierarchy.d.ts.map