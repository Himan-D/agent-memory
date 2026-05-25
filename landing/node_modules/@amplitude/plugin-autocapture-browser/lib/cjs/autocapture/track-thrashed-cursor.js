"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.trackThrashedCursor = exports.createThrashedCursorObservable = exports.createMouseDirectionChangeObservable = void 0;
var analytics_core_1 = require("@amplitude/analytics-core");
var constants_1 = require("../constants");
var helpers_1 = require("../helpers");
var Direction;
(function (Direction) {
    Direction["INCREASING"] = "increasing";
    Direction["DECREASING"] = "decreasing";
})(Direction || (Direction = {}));
var Axis;
(function (Axis) {
    Axis["X"] = "x";
    Axis["Y"] = "y";
})(Axis || (Axis = {}));
var createMouseDirectionChangeObservable = function (_a) {
    var allWindowObservables = _a.allWindowObservables;
    var mouseMoveObservable = allWindowObservables.mouseMoveObservable;
    return new analytics_core_1.Observable(function (observer) {
        var lastPosition = null;
        var xDirection = null;
        var yDirection = null;
        return mouseMoveObservable.subscribe(function (event) {
            var currentPosition = { x: event.clientX, y: event.clientY };
            if (lastPosition === null) {
                lastPosition = currentPosition;
                return;
            }
            if (currentPosition.x > lastPosition.x) {
                if (xDirection === Direction.DECREASING) {
                    observer.next(Axis.X);
                }
                xDirection = Direction.INCREASING;
            }
            else if (currentPosition.x < lastPosition.x) {
                if (xDirection === Direction.INCREASING) {
                    observer.next(Axis.X);
                }
                xDirection = Direction.DECREASING;
            }
            if (currentPosition.y > lastPosition.y) {
                if (yDirection === Direction.DECREASING) {
                    observer.next(Axis.Y);
                }
                yDirection = Direction.INCREASING;
            }
            else if (currentPosition.y < lastPosition.y) {
                if (yDirection === Direction.INCREASING) {
                    observer.next(Axis.Y);
                }
                yDirection = Direction.DECREASING;
            }
            lastPosition = currentPosition;
        });
    });
};
exports.createMouseDirectionChangeObservable = createMouseDirectionChangeObservable;
function addDirectionChange(directionChangeSeries) {
    var now = +Date.now();
    directionChangeSeries.startTime = directionChangeSeries.startTime || now;
    // add this direction change to the series (fixed length array to avoid memory leaks)
    var changes = directionChangeSeries.changes, changesThreshold = directionChangeSeries.changesThreshold;
    changes.push(now);
    if (changes.length > changesThreshold)
        changes.shift();
}
// checks if there are enough direction changes within window + threshold
// for it to be considered a thrashed cursor
function isThrashedCursor(directionChanges) {
    var changes = directionChanges.changes, changesThreshold = directionChanges.changesThreshold, thresholdMs = directionChanges.thresholdMs;
    if (changes.length < changesThreshold)
        return false;
    var delta = changes[changes.length - 1] - changes[0];
    return delta < thresholdMs;
}
function resetDirectionChangeSeries(directionChangeSeries) {
    directionChangeSeries.changes = [];
    directionChangeSeries.startTime = undefined;
}
// if the time between first and last change is greater than the threshold,
// shift the window to the right until it is below the threshold
function adjustWindow(directionChanges) {
    var changes = directionChanges.changes, thresholdMs = directionChanges.thresholdMs;
    // find the first change that is within the threshold
    var leftPtr = 0;
    var lastChange = changes[changes.length - 1];
    for (; leftPtr < changes.length; leftPtr++) {
        var delta = lastChange - changes[leftPtr];
        if (delta < thresholdMs) {
            break;
        }
    }
    if (leftPtr === 0)
        return;
    directionChanges.startTime = changes[leftPtr];
    directionChanges.changes.splice(0, leftPtr);
}
function getPendingThrashedCursor(directionChangesX, directionChangesY) {
    var startTime = undefined;
    if (isThrashedCursor(directionChangesX)) {
        startTime = directionChangesX.startTime;
    }
    if (isThrashedCursor(directionChangesY)) {
        var startTimeY = directionChangesY.startTime;
        if (startTimeY && (!startTime || startTimeY < startTime)) {
            startTime = startTimeY;
        }
    }
    return startTime;
}
var DEFAULT_THRESHOLD = 20;
var DEFAULT_WINDOW_MS = 2000;
var createThrashedCursorObservable = function (_a) {
    var mouseDirectionChangeObservable = _a.mouseDirectionChangeObservable, _b = _a.directionChanges, directionChanges = _b === void 0 ? DEFAULT_THRESHOLD : _b, _c = _a.thresholdMs, thresholdMs = _c === void 0 ? DEFAULT_WINDOW_MS : _c;
    return new analytics_core_1.Observable(function (observer) {
        var xDirectionChanges = { changes: [], changesThreshold: directionChanges, thresholdMs: thresholdMs };
        var yDirectionChanges = { changes: [], changesThreshold: directionChanges, thresholdMs: thresholdMs };
        var pendingThrashedCursor = undefined;
        var timer = null;
        function emitPendingThrashedCursor() {
            if (pendingThrashedCursor !== undefined) {
                observer.next(pendingThrashedCursor);
                pendingThrashedCursor = undefined;
                // reset window
                if (timer !== null)
                    clearTimeout(timer);
                resetDirectionChangeSeries(xDirectionChanges);
                resetDirectionChangeSeries(yDirectionChanges);
            }
        }
        return mouseDirectionChangeObservable.subscribe(function (axis) {
            if (timer !== null)
                clearTimeout(timer);
            addDirectionChange(axis === Axis.X ? xDirectionChanges : yDirectionChanges);
            var nextPendingThrashedCursor = getPendingThrashedCursor(xDirectionChanges, yDirectionChanges);
            if (nextPendingThrashedCursor) {
                // if we're in a thrashed cursor window, debounce it for "thresholdMs" duration
                // this is so that we do not restart the window if more direction changes are
                // detected in this series
                pendingThrashedCursor = pendingThrashedCursor || nextPendingThrashedCursor;
                timer = setTimeout(function () {
                    emitPendingThrashedCursor();
                    timer = null;
                }, thresholdMs);
            }
            else {
                emitPendingThrashedCursor();
            }
            adjustWindow(xDirectionChanges);
            adjustWindow(yDirectionChanges);
            /* istanbul ignore next */
            return function () {
                /* istanbul ignore if */
                if (timer !== null) {
                    clearTimeout(timer);
                    timer = null;
                }
            };
        });
    });
};
exports.createThrashedCursorObservable = createThrashedCursorObservable;
var trackThrashedCursor = function (_a) {
    var amplitude = _a.amplitude, options = _a.options, allObservables = _a.allObservables, _b = _a.directionChanges, directionChanges = _b === void 0 ? DEFAULT_THRESHOLD : _b, _c = _a.thresholdMs, thresholdMs = _c === void 0 ? DEFAULT_WINDOW_MS : _c;
    var mouseDirectionChangeObservable = (0, exports.createMouseDirectionChangeObservable)({ allWindowObservables: allObservables });
    var thrashedCursorObservable = (0, exports.createThrashedCursorObservable)({
        mouseDirectionChangeObservable: mouseDirectionChangeObservable,
        directionChanges: directionChanges,
        thresholdMs: thresholdMs,
    });
    return thrashedCursorObservable.subscribe(function (time) {
        if (!(0, helpers_1.isUrlAllowed)(options)) {
            return;
        }
        amplitude.track(constants_1.AMPLITUDE_THRASHED_CURSOR_EVENT, undefined, { time: time });
    });
};
exports.trackThrashedCursor = trackThrashedCursor;
//# sourceMappingURL=track-thrashed-cursor.js.map