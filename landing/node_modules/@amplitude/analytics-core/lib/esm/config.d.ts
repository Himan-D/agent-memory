import { OfflineDisabled } from './types/offline';
import { ServerZoneType } from './types/server-zone';
import { Transport } from './types/transport';
import { Event } from './types/event/event';
import { Plan } from './types/event/plan';
import { IngestionMetadata } from './types/event/ingestion-metadata';
import { Storage } from './types/storage';
import { Logger, ILogger } from './logger';
import { LogLevel } from './types/loglevel';
import { ConfigOptions, IRequestMetadata, IHistogramOptions, HistogramKey, IConfig } from './types/config/core-config';
export declare const getDefaultConfig: () => {
    flushMaxRetries: number;
    flushQueueSize: number;
    flushIntervalMillis: number;
    instanceName: string;
    logLevel: LogLevel;
    loggerProvider: Logger;
    offline: boolean;
    optOut: boolean;
    serverUrl: string;
    serverZone: ServerZoneType;
    useBatch: boolean;
};
export declare class Config implements IConfig {
    apiKey: string;
    flushIntervalMillis: number;
    flushMaxRetries: number;
    flushQueueSize: number;
    instanceName?: string;
    loggerProvider: ILogger;
    logLevel: LogLevel;
    minIdLength?: number;
    offline?: boolean | typeof OfflineDisabled;
    plan?: Plan;
    ingestionMetadata?: IngestionMetadata;
    serverUrl: string | undefined;
    serverZone?: ServerZoneType;
    transportProvider: Transport;
    storageProvider?: Storage<Event[]>;
    useBatch: boolean;
    requestMetadata?: RequestMetadata;
    protected _optOut: boolean;
    get optOut(): boolean;
    set optOut(optOut: boolean);
    constructor(options: ConfigOptions);
}
export declare const getServerUrl: (serverZone: ServerZoneType, useBatch: boolean) => "https://api2.amplitude.com/2/httpapi" | "https://api.eu.amplitude.com/2/httpapi" | "https://api2.amplitude.com/batch" | "https://api.eu.amplitude.com/batch";
export declare const createServerConfig: (serverUrl?: string, serverZone?: ServerZoneType, useBatch?: boolean) => {
    serverUrl: string;
    serverZone: undefined;
} | {
    serverZone: ServerZoneType;
    serverUrl: string;
};
export declare class RequestMetadata implements IRequestMetadata {
    sdk: {
        metrics: {
            histogram: HistogramOptions;
        };
    };
    constructor();
    recordHistogram<T extends HistogramKey>(key: T, value: HistogramOptions[T]): void;
}
declare class HistogramOptions implements IHistogramOptions {
    remote_config_fetch_time_IDB?: number;
    remote_config_fetch_time_API_success?: number;
    remote_config_fetch_time_API_fail?: number;
}
export {};
//# sourceMappingURL=config.d.ts.map