"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.executeActions = exports.getDataSource = void 0;
// Get DataSource
/**
 * Gets the DOM element specified by the data source configuration
 * @param dataSource - Configuration for finding the target element
 * @param contextElement - The element to start searching from
 * @returns The matching DOM element or undefined if not found
 */
var getDataSource = function (dataSource, contextElement) {
    // Only process DOM_ELEMENT type data sources
    try {
        if (dataSource.sourceType === 'DOM_ELEMENT') {
            // If scope is specified, find the closest ancestor matching the scope rather than using documentElement (html) as the scope
            var scopingElement = document.documentElement;
            if (dataSource.scope && contextElement) {
                scopingElement = contextElement.closest(dataSource.scope);
            }
            // If we have both a scope and selector, find the matching element
            if (scopingElement && dataSource.selector) {
                return scopingElement.querySelector(dataSource.selector);
            }
            // Return scopingElement if no selector was specified
            return scopingElement;
        }
    }
    catch (error) {
        return undefined;
    }
    // Return undefined for non-DOM_ELEMENT data sources
    return undefined;
};
exports.getDataSource = getDataSource;
// Execute actions for a condition and attach event properties to the event if needed
var executeActions = function (actions, ev, dataExtractor) {
    actions.forEach(function (action) {
        // Skip if actions is string until action set is implemented
        if (typeof action === 'string') {
            return;
        }
        if (action.actionType === 'ATTACH_EVENT_PROPERTY') {
            var data = dataExtractor.extractDataFromDataSource(action.dataSource, ev.closestTrackedAncestor);
            // Attach data to event
            ev.targetElementProperties[action.destinationKey] = data;
        }
    });
};
exports.executeActions = executeActions;
//# sourceMappingURL=actions.js.map