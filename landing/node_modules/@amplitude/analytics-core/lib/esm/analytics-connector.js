import { AnalyticsConnector } from '@amplitude/analytics-connector';
import { DEFAULT_INSTANCE_NAME } from './types/constants';
export var getAnalyticsConnector = function (instanceName) {
    if (instanceName === void 0) { instanceName = DEFAULT_INSTANCE_NAME; }
    return AnalyticsConnector.getInstance(instanceName);
};
export var setConnectorUserId = function (userId, instanceName) {
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore
    getAnalyticsConnector(instanceName).identityStore.editIdentity().setUserId(userId).commit();
};
export var setConnectorDeviceId = function (deviceId, instanceName) {
    getAnalyticsConnector(instanceName).identityStore.editIdentity().setDeviceId(deviceId).commit();
};
//# sourceMappingURL=analytics-connector.js.map