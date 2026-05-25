/* istanbul ignore file */
/* eslint-disable no-restricted-globals */
import { AMPLITUDE_VISUAL_TAGGING_SELECTOR_SCRIPT_URL, AMPLITUDE_VISUAL_TAGGING_HIGHLIGHT_CLASS } from '../constants';
import { VERSION } from '../version';
/**
 * Brand key to track whether visual tagging has been enabled on a messenger.
 */
var VISUAL_TAGGING_BRAND = '__AMPLITUDE_VISUAL_TAGGING__';
/**
 * Enable visual tagging on a messenger instance.
 * The first call registers the handlers; subsequent calls are no-ops.
 *
 * @param messenger - The messenger to enable visual tagging on
 * @param options - Visual tagging configuration
 */
export function enableVisualTagging(messenger, options) {
    // Idempotency guard — works across bundle boundaries
    var branded = messenger;
    if (branded[VISUAL_TAGGING_BRAND] === true) {
        return;
    }
    branded[VISUAL_TAGGING_BRAND] = true;
    var dataExtractor = options.dataExtractor, isElementSelectable = options.isElementSelectable, cssSelectorAllowlist = options.cssSelectorAllowlist, actionClickAllowlist = options.actionClickAllowlist;
    var amplitudeVisualTaggingSelectorInstance = null;
    var onSelect = function (data) {
        messenger.notify({ action: 'element-selected', data: data });
    };
    var onTrack = function (type, properties) {
        if (type === 'selector-mode-changed') {
            messenger.notify({ action: 'track-selector-mode-changed', data: properties });
        }
        else if (type === 'selector-moved') {
            messenger.notify({ action: 'track-selector-moved', data: properties });
        }
    };
    messenger.registerActionHandler('initialize-visual-tagging-selector', function (actionData) {
        messenger
            .loadScriptOnce(AMPLITUDE_VISUAL_TAGGING_SELECTOR_SCRIPT_URL)
            .then(function () {
            var _a;
            // eslint-disable-next-line
            amplitudeVisualTaggingSelectorInstance = (_a = window === null || window === void 0 ? void 0 : window.amplitudeVisualTaggingSelector) === null || _a === void 0 ? void 0 : _a.call(window, {
                getEventTagProps: dataExtractor.getEventTagProps,
                isElementSelectable: function (element) {
                    if (isElementSelectable) {
                        return isElementSelectable((actionData === null || actionData === void 0 ? void 0 : actionData.actionType) || 'click', element);
                    }
                    return true;
                },
                onTrack: onTrack,
                onSelect: onSelect,
                visualHighlightClass: AMPLITUDE_VISUAL_TAGGING_HIGHLIGHT_CLASS,
                messenger: messenger,
                cssSelectorAllowlist: cssSelectorAllowlist,
                actionClickAllowlist: actionClickAllowlist,
                extractDataFromDataSource: dataExtractor.extractDataFromDataSource,
                dataExtractor: dataExtractor,
                diagnostics: {
                    autocapture: {
                        version: VERSION,
                    },
                },
            });
            messenger.notify({ action: 'selector-loaded' });
        })
            .catch(function () {
            var _a;
            (_a = messenger.logger) === null || _a === void 0 ? void 0 : _a.warn('Failed to initialize visual tagging selector');
        });
    });
    messenger.registerActionHandler('close-visual-tagging-selector', function () {
        var _a;
        // eslint-disable-next-line
        (_a = amplitudeVisualTaggingSelectorInstance === null || amplitudeVisualTaggingSelectorInstance === void 0 ? void 0 : amplitudeVisualTaggingSelectorInstance.close) === null || _a === void 0 ? void 0 : _a.call(amplitudeVisualTaggingSelectorInstance);
    });
}
//# sourceMappingURL=messenger.js.map