import { ILogger } from '../logger';
import { HistogramStats } from './diagnostics-client';
export declare const TABLE_NAMES: {
    readonly TAGS: "tags";
    readonly COUNTERS: "counters";
    readonly HISTOGRAMS: "histograms";
    readonly EVENTS: "events";
    readonly INTERNAL: "internal";
};
export declare const INTERNAL_KEYS: {
    readonly LAST_FLUSH_TIMESTAMP: "last_flush_timestamp";
};
export interface TagRecord {
    key: string;
    value: string;
}
export interface CounterRecord {
    key: string;
    value: number;
}
export interface HistogramRecord {
    key: string;
    count: number;
    min: number;
    max: number;
    sum: number;
}
export interface EventRecord {
    id?: number;
    event_name: string;
    time: number;
    event_properties: Record<string, any>;
}
export interface InternalRecord {
    key: string;
    value: string;
}
export interface IDiagnosticsStorage {
    /**
     * Set multiple tags in a single transaction (batch operation)
     * Promise never rejects - errors are logged and operation continues gracefully
     */
    setTags(tags: Record<string, string>): Promise<void>;
    /**
     * Increment multiple counters in a single transaction (batch operation)
     * Uses read-modify-write pattern to accumulate with existing values
     * Promise never rejects - errors are logged and operation continues gracefully
     */
    incrementCounters(counters: Record<string, number>): Promise<void>;
    /**
     * Set multiple histogram stats in a single transaction (batch operation)
     * Uses read-modify-write pattern to accumulate count/sum and update min/max with existing values
     * Promise never rejects - errors are logged and operation continues gracefully
     */
    setHistogramStats(histogramStats: Record<string, {
        count: number;
        min: number;
        max: number;
        sum: number;
    }>): Promise<void>;
    /**
     * Add multiple event records in a single transaction (batch operation)
     * Promise never rejects - errors are logged and operation continues gracefully
     */
    addEventRecords(events: Array<{
        event_name: string;
        time: number;
        event_properties: Record<string, any>;
    }>): Promise<void>;
    setLastFlushTimestamp(timestamp: number): Promise<void>;
    getLastFlushTimestamp(): Promise<number | undefined>;
    /**
     * Get all data except internal data from storage and clear it
     */
    getAllAndClear(): Promise<{
        tags: TagRecord[];
        counters: CounterRecord[];
        histogramStats: HistogramRecord[];
        events: EventRecord[];
    }>;
}
/**
 * Purpose-specific IndexedDB storage for diagnostics data
 * Provides optimized methods for each type of diagnostics data
 */
export declare class DiagnosticsStorage implements IDiagnosticsStorage {
    dbPromise: Promise<IDBDatabase> | null;
    dbName: string;
    logger: ILogger;
    constructor(apiKey: string, logger: ILogger);
    /**
     * Check if IndexedDB is supported in the current environment
     * @returns true if IndexedDB is available, false otherwise
     */
    static isSupported(): boolean;
    getDB(): Promise<IDBDatabase>;
    openDB(): Promise<IDBDatabase>;
    createTables(db: IDBDatabase): void;
    setTags(tags: Record<string, string>): Promise<void>;
    incrementCounters(counters: Record<string, number>): Promise<void>;
    setHistogramStats(histogramStats: Record<string, HistogramStats>): Promise<void>;
    addEventRecords(events: Array<{
        event_name: string;
        time: number;
        event_properties: Record<string, any>;
    }>): Promise<void>;
    setInternal(key: string, value: string): Promise<void>;
    getInternal(key: string): Promise<InternalRecord | undefined>;
    getLastFlushTimestamp(): Promise<number | undefined>;
    setLastFlushTimestamp(timestamp: number): Promise<void>;
    clearTable(transaction: IDBTransaction, tableName: string): Promise<void>;
    getAllAndClear(): Promise<{
        tags: TagRecord[];
        counters: CounterRecord[];
        histogramStats: HistogramRecord[];
        events: EventRecord[];
    }>;
    /**
     * Helper method to get all records from a store within a transaction
     */
    private getAllFromStore;
}
//# sourceMappingURL=diagnostics-storage.d.ts.map