import { IDiagnosticsClient } from './diagnostics-client';
export declare const GLOBAL_KEY = "__AMPLITUDE_SCRIPT_URL__";
export declare const EVENT_NAME_ERROR_UNCAUGHT = "sdk.error.uncaught";
export declare const registerSdkLoaderMetadata: (metadata: {
    scriptUrl?: string;
}) => void;
export declare const enableSdkErrorListeners: (client: IDiagnosticsClient) => void;
//# sourceMappingURL=uncaught-sdk-errors.d.ts.map