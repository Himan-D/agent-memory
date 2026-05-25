"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.setConnectorDeviceId = exports.setConnectorUserId = exports.getAnalyticsConnector = void 0;
var analytics_connector_1 = require("@amplitude/analytics-connector");
var constants_1 = require("./types/constants");
var getAnalyticsConnector = function (instanceName) {
    if (instanceName === void 0) { instanceName = constants_1.DEFAULT_INSTANCE_NAME; }
    return analytics_connector_1.AnalyticsConnector.getInstance(instanceName);
};
exports.getAnalyticsConnector = getAnalyticsConnector;
var setConnectorUserId = function (userId, instanceName) {
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore
    (0, exports.getAnalyticsConnector)(instanceName).identityStore.editIdentity().setUserId(userId).commit();
};
exports.setConnectorUserId = setConnectorUserId;
var setConnectorDeviceId = function (deviceId, instanceName) {
    (0, exports.getAnalyticsConnector)(instanceName).identityStore.editIdentity().setDeviceId(deviceId).commit();
};
exports.setConnectorDeviceId = setConnectorDeviceId;
//# sourceMappingURL=analytics-connector.js.map