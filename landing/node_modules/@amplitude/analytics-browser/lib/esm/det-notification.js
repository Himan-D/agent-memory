var notified = false;
export var detNotify = function (config) {
    if (notified || config.defaultTracking !== undefined) {
        return;
    }
    var message = "`options.defaultTracking` is set to undefined. This implicitly configures your Amplitude instance to track Page Views, Sessions, File Downloads, and Form Interactions. You can suppress this warning by explicitly setting a value to `options.defaultTracking`. The value must either be a boolean, to enable and disable all default events, or an object, for advanced configuration. For example:\n\namplitude.init(<YOUR_API_KEY>, {\n  defaultTracking: true,\n});\n\nVisit https://www.docs.developers.amplitude.com/data/sdks/browser-2/#tracking-default-events for more details.";
    config.loggerProvider.warn(message);
    notified = true;
};
/**
 * @private
 * This function is meant for testing purposes only
 */
export var resetNotify = function () {
    notified = false;
};
//# sourceMappingURL=det-notification.js.map