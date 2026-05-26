export var isNewSession = function (sessionTimeout, lastEventTime) {
    if (lastEventTime === void 0) { lastEventTime = Date.now(); }
    var currentTime = Date.now();
    var timeSinceLastEvent = currentTime - lastEventTime;
    return timeSinceLastEvent > sessionTimeout;
};
//# sourceMappingURL=session.js.map