import type { BaseWindowMessenger } from '@amplitude/analytics-core';
import { ActionType } from '@amplitude/analytics-core';
import { DataExtractor } from '../data-extractor';
export type Action = 'ping' | 'pong' | 'page-loaded' | 'selector-loaded' | 'initialize-visual-tagging-selector' | 'close-visual-tagging-selector' | 'element-selected' | 'track-selector-mode-changed' | 'track-selector-moved' | 'initialize-background-capture' | 'close-background-capture' | 'background-capture-loaded' | 'background-capture-complete';
interface InitializeVisualTaggingSelectorData {
    actionType: ActionType;
}
interface ElementSelectedData {
    '[Amplitude] Element Hierarchy'?: string;
    '[Amplitude] Element Tag'?: string;
    '[Amplitude] Element Text'?: string;
    '[Amplitude] Page URL'?: string;
    elementScreenshot?: Blob;
}
interface TrackSelectorModeChangedData {
    newMode: string;
    pageUrl?: string;
}
interface TrackSelectorMovedData {
    newEditorLocation: string;
    pageUrl?: string;
}
export type ActionData = {
    ping: null | undefined;
    pong: null | undefined;
    'page-loaded': null | undefined;
    'selector-loaded': null | undefined;
    'initialize-visual-tagging-selector': InitializeVisualTaggingSelectorData | null | undefined;
    'close-visual-tagging-selector': null | undefined;
    'element-selected': ElementSelectedData;
    'track-selector-mode-changed': TrackSelectorModeChangedData;
    'track-selector-moved': TrackSelectorMovedData;
    'initialize-background-capture': null | undefined;
    'close-background-capture': null | undefined;
    'background-capture-loaded': null | undefined;
    'background-capture-complete': {
        [key: string]: string | null;
    };
};
export interface Message<A extends Action> {
    action: A;
    data?: ActionData[A];
}
/**
 * Enable visual tagging on a messenger instance.
 * The first call registers the handlers; subsequent calls are no-ops.
 *
 * @param messenger - The messenger to enable visual tagging on
 * @param options - Visual tagging configuration
 */
export declare function enableVisualTagging(messenger: BaseWindowMessenger, options: {
    isElementSelectable?: (action: InitializeVisualTaggingSelectorData['actionType'], element: Element) => boolean;
    cssSelectorAllowlist?: string[];
    actionClickAllowlist?: string[];
    dataExtractor: DataExtractor;
}): void;
export {};
//# sourceMappingURL=messenger.d.ts.map