"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createTriggerEvaluator = exports.TriggerEvaluator = exports.matchLabeledEventsToTriggers = exports.matchEventToLabeledEvents = exports.createLabeledEventToTriggerMap = exports.groupLabeledEventIdsByEventType = void 0;
var tslib_1 = require("tslib");
var matchEventToFilter_1 = require("./matchEventToFilter");
var actions_1 = require("./actions");
var eventTypeToBrowserEventMap = {
    '[Amplitude] Element Clicked': 'click',
    '[Amplitude] Element Changed': 'change',
};
// groups labeled events by event type
// skips any labeled events with malformed definitions or unexpected event_type
var groupLabeledEventIdsByEventType = function (labeledEvents) {
    var e_1, _a, e_2, _b;
    // Initialize all event types with empty sets
    var groupedLabeledEvents = Object.values(eventTypeToBrowserEventMap).reduce(function (acc, browserEvent) {
        acc[browserEvent] = new Set();
        return acc;
    }, {});
    // If there are no labeled events, return the initialized groupedLabeledEvents
    if (!labeledEvents) {
        return groupedLabeledEvents;
    }
    try {
        // Group labeled events by event type
        for (var labeledEvents_1 = tslib_1.__values(labeledEvents), labeledEvents_1_1 = labeledEvents_1.next(); !labeledEvents_1_1.done; labeledEvents_1_1 = labeledEvents_1.next()) {
            var le = labeledEvents_1_1.value;
            try {
                try {
                    for (var _c = (e_2 = void 0, tslib_1.__values(le.definition)), _d = _c.next(); !_d.done; _d = _c.next()) {
                        var def = _d.value;
                        var browserEvent = eventTypeToBrowserEventMap[def.event_type];
                        if (browserEvent) {
                            groupedLabeledEvents[browserEvent].add(le.id);
                        }
                    }
                }
                catch (e_2_1) { e_2 = { error: e_2_1 }; }
                finally {
                    try {
                        if (_d && !_d.done && (_b = _c.return)) _b.call(_c);
                    }
                    finally { if (e_2) throw e_2.error; }
                }
            }
            catch (e) {
                // Skip this labeled event if there is an error
                console.warn('Skipping Labeled Event due to malformed definition', le === null || le === void 0 ? void 0 : le.id, e);
            }
        }
    }
    catch (e_1_1) { e_1 = { error: e_1_1 }; }
    finally {
        try {
            if (labeledEvents_1_1 && !labeledEvents_1_1.done && (_a = labeledEvents_1.return)) _a.call(labeledEvents_1);
        }
        finally { if (e_1) throw e_1.error; }
    }
    return groupedLabeledEvents;
};
exports.groupLabeledEventIdsByEventType = groupLabeledEventIdsByEventType;
// TODO: add tests
var createLabeledEventToTriggerMap = function (triggers) {
    var e_3, _a, e_4, _b;
    var labeledEventToTriggerMap = new Map();
    try {
        for (var triggers_1 = tslib_1.__values(triggers), triggers_1_1 = triggers_1.next(); !triggers_1_1.done; triggers_1_1 = triggers_1.next()) {
            var trigger = triggers_1_1.value;
            try {
                for (var _c = (e_4 = void 0, tslib_1.__values(trigger.conditions)), _d = _c.next(); !_d.done; _d = _c.next()) {
                    var condition = _d.value;
                    if (condition.type === 'LABELED_EVENT') {
                        var eventId = condition.match.eventId;
                        // Get existing triggers for this event ID or initialize empty array
                        var existingTriggers = labeledEventToTriggerMap.get(eventId);
                        if (!existingTriggers) {
                            existingTriggers = [];
                            labeledEventToTriggerMap.set(eventId, existingTriggers);
                        }
                        // Add current trigger to the list of triggers for this event ID
                        existingTriggers.push(trigger);
                    }
                }
            }
            catch (e_4_1) { e_4 = { error: e_4_1 }; }
            finally {
                try {
                    if (_d && !_d.done && (_b = _c.return)) _b.call(_c);
                }
                finally { if (e_4) throw e_4.error; }
            }
        }
    }
    catch (e_3_1) { e_3 = { error: e_3_1 }; }
    finally {
        try {
            if (triggers_1_1 && !triggers_1_1.done && (_a = triggers_1.return)) _a.call(triggers_1);
        }
        finally { if (e_3) throw e_3.error; }
    }
    return labeledEventToTriggerMap;
};
exports.createLabeledEventToTriggerMap = createLabeledEventToTriggerMap;
/**
 * Matches an event to labeled events based on the event's properties.
 * The logic matches exactly what is supported by the query backend.
 * TODO: later pre-filter the labeled events based on URL first
 *
 * @param event - The event to match against labeled events
 * @param labeledEvents - Array of labeled events to match against
 * @returns Array of matching labeled events
 */
var matchEventToLabeledEvents = function (event, labeledEvents) {
    return labeledEvents.filter(function (le) {
        return le.definition.some(function (def) {
            return (eventTypeToBrowserEventMap[def.event_type] === event.type &&
                def.filters.every(function (filter) { return (0, matchEventToFilter_1.matchEventToFilter)(event, filter); }));
        });
    });
};
exports.matchEventToLabeledEvents = matchEventToLabeledEvents;
var matchLabeledEventsToTriggers = function (labeledEvents, leToTriggerMap) {
    var e_5, _a, e_6, _b;
    var matchingTriggers = new Set();
    try {
        for (var labeledEvents_2 = tslib_1.__values(labeledEvents), labeledEvents_2_1 = labeledEvents_2.next(); !labeledEvents_2_1.done; labeledEvents_2_1 = labeledEvents_2.next()) {
            var le = labeledEvents_2_1.value;
            var triggers = leToTriggerMap.get(le.id);
            if (triggers) {
                try {
                    for (var triggers_2 = (e_6 = void 0, tslib_1.__values(triggers)), triggers_2_1 = triggers_2.next(); !triggers_2_1.done; triggers_2_1 = triggers_2.next()) {
                        var trigger = triggers_2_1.value;
                        matchingTriggers.add(trigger);
                    }
                }
                catch (e_6_1) { e_6 = { error: e_6_1 }; }
                finally {
                    try {
                        if (triggers_2_1 && !triggers_2_1.done && (_b = triggers_2.return)) _b.call(triggers_2);
                    }
                    finally { if (e_6) throw e_6.error; }
                }
            }
        }
    }
    catch (e_5_1) { e_5 = { error: e_5_1 }; }
    finally {
        try {
            if (labeledEvents_2_1 && !labeledEvents_2_1.done && (_a = labeledEvents_2.return)) _a.call(labeledEvents_2);
        }
        finally { if (e_5) throw e_5.error; }
    }
    return Array.from(matchingTriggers);
};
exports.matchLabeledEventsToTriggers = matchLabeledEventsToTriggers;
var TriggerEvaluator = /** @class */ (function () {
    function TriggerEvaluator(groupedLabeledEvents, labeledEventToTriggerMap, dataExtractor, options) {
        this.groupedLabeledEvents = groupedLabeledEvents;
        this.labeledEventToTriggerMap = labeledEventToTriggerMap;
        this.dataExtractor = dataExtractor;
        this.options = options;
    }
    TriggerEvaluator.prototype.evaluate = function (event) {
        var e_7, _a;
        // If there is no pageActions, return the event as is
        var pageActions = this.options.pageActions;
        if (!pageActions) {
            return event;
        }
        // Find matching labeled events
        var matchingLabeledEvents = (0, exports.matchEventToLabeledEvents)(event, Array.from(this.groupedLabeledEvents[event.type]).map(function (id) { return pageActions.labeledEvents[id]; }));
        // Find matching conditions
        var matchingTriggers = (0, exports.matchLabeledEventsToTriggers)(matchingLabeledEvents, this.labeledEventToTriggerMap);
        try {
            for (var matchingTriggers_1 = tslib_1.__values(matchingTriggers), matchingTriggers_1_1 = matchingTriggers_1.next(); !matchingTriggers_1_1.done; matchingTriggers_1_1 = matchingTriggers_1.next()) {
                var trigger = matchingTriggers_1_1.value;
                (0, actions_1.executeActions)(trigger.actions, event, this.dataExtractor);
            }
        }
        catch (e_7_1) { e_7 = { error: e_7_1 }; }
        finally {
            try {
                if (matchingTriggers_1_1 && !matchingTriggers_1_1.done && (_a = matchingTriggers_1.return)) _a.call(matchingTriggers_1);
            }
            finally { if (e_7) throw e_7.error; }
        }
        return event;
    };
    TriggerEvaluator.prototype.update = function (groupedLabeledEvents, labeledEventToTriggerMap, options) {
        this.groupedLabeledEvents = groupedLabeledEvents;
        this.labeledEventToTriggerMap = labeledEventToTriggerMap;
        this.options = options;
    };
    return TriggerEvaluator;
}());
exports.TriggerEvaluator = TriggerEvaluator;
var createTriggerEvaluator = function (groupedLabeledEvents, labeledEventToTriggerMap, dataExtractor, options) {
    return new TriggerEvaluator(groupedLabeledEvents, labeledEventToTriggerMap, dataExtractor, options);
};
exports.createTriggerEvaluator = createTriggerEvaluator;
//# sourceMappingURL=triggers.js.map