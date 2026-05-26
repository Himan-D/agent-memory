import { ILogger } from '../logger';
import { IDiagnosticsStorage } from './diagnostics-storage';
import { ServerZoneType } from '../types/server-zone';
export declare const SAVE_INTERVAL_MS = 1000;
export declare const FLUSH_INTERVAL_MS: number;
export declare const DIAGNOSTICS_US_SERVER_URL = "https://diagnostics.prod.us-west-2.amplitude.com/v1/capture";
export declare const DIAGNOSTICS_EU_SERVER_URL = "https://diagnostics.prod.eu-central-1.amplitude.com/v1/capture";
export declare const MAX_MEMORY_STORAGE_COUNT = 10000;
export declare const MAX_MEMORY_STORAGE_EVENTS_COUNT = 10;
/**
 * Key-value pairs for environment/context information
 */
type DiagnosticsTags = Record<string, string>;
/**
 * Numeric counters that can be incremented
 */
type DiagnosticsCounters = Record<string, number>;
/**
 * Properties for diagnostic events
 */
type EventProperties = Record<string, any>;
/**
 * Individual diagnostic event
 */
interface DiagnosticsEvent {
    readonly event_name: string;
    readonly time: number;
    readonly event_properties: EventProperties;
}
/**
 * Computed histogram statistics for final payload
 */
interface HistogramResult {
    readonly count: number;
    readonly min: number;
    readonly max: number;
    readonly avg: number;
}
/**
 * Internal histogram statistics with sum for efficient incremental updates
 */
export interface HistogramStats {
    count: number;
    min: number;
    max: number;
    sum: number;
}
/**
 * Collection of histogram results keyed by histogram name
 */
type DiagnosticsHistograms = Record<string, HistogramResult>;
/**
 * Collection of histogram stats keyed by histogram name (internal use for memory + persistence storage)
 */
type DiagnosticsHistogramStats = Record<string, HistogramStats>;
/**
 * Complete diagnostics payload sent to backend
 */
interface FlushPayload {
    readonly tags: DiagnosticsTags;
    readonly histogram: DiagnosticsHistograms;
    readonly counters: DiagnosticsCounters;
    readonly events: readonly DiagnosticsEvent[];
}
/**
 * Amplitude Diagnostics Client
 *
 * A client for collecting and managing diagnostics data including tags, counters,
 * histograms, and events. Data is stored persistently using IndexedDB to survive browser restarts and offline scenarios.
 *
 * Key Features:
 * - IndexedDB storage
 * - Time-based persistent storage flush interval (5 minutes since last flush)
 * - 1 second time-based memory storage flush to persistent storage
 * - Histogram statistics calculation (min, max, avg)
 */
export interface IDiagnosticsClient {
    /**
     * Set or update a tag
     *
     * @example
     * ```typescript
     * // Set environment tags
     * diagnostics.setTag('library', 'amplitude-typescript/2.0.0');
     * diagnostics.setTag('user_agent', navigator.userAgent);
     * ```
     */
    setTag(name: string, value: string): void;
    /**
     * Increment a counter. If doesn't exist, create a counter and set value to 1
     *
     * @example
     * ```typescript
     * // Track counters
     * diagnostics.increment('analytics.fileNotFound');
     * diagnostics.increment('network.retry', 3);
     * ```
     */
    increment(name: string, size?: number): void;
    /**
     * Record a histogram value
     *
     * @example
     * ```typescript
     * // Record performance metrics
     * diagnostics.recordHistogram('sr.time', 5.2);
     * diagnostics.recordHistogram('network.latency', 150);
     * ```
     */
    recordHistogram(name: string, value: number): void;
    /**
     * Record an event
     *
     * @example
     * ```typescript
     * // Record diagnostic events
     * diagnostics.recordEvent('error', {
     *   stack_trace: '...',
     * });
     * ```
     */
    recordEvent(name: string, properties: EventProperties): void;
    _flush(): void;
    /**
     * Sets the sample rate for diagnostics.
     *
     * @example
     * ```typescript
     * diagnostics.setSampleRate(0.5);
     * ```
     */
    _setSampleRate(sampleRate: number): void;
}
export declare class DiagnosticsClient implements IDiagnosticsClient {
    storage?: IDiagnosticsStorage;
    logger: ILogger;
    serverUrl: string;
    apiKey: string;
    shouldTrack: boolean;
    config: {
        enabled: boolean;
        sampleRate: number;
    };
    /**
     * The timestamp when the diagnostics client was initialized.
     * Save in memory to keep lifecycle sample rate calculation consistency.
     */
    startTimestamp: number;
    inMemoryTags: DiagnosticsTags;
    inMemoryCounters: DiagnosticsCounters;
    inMemoryHistograms: DiagnosticsHistogramStats;
    inMemoryEvents: DiagnosticsEvent[];
    saveTimer: ReturnType<typeof setTimeout> | null;
    flushTimer: ReturnType<typeof setTimeout> | null;
    constructor(apiKey: string, logger: ILogger, serverZone?: ServerZoneType, options?: {
        enabled?: boolean;
        sampleRate?: number;
    });
    /**
     * Check if storage is available and tracking is enabled
     */
    isStorageAndTrackEnabled(): boolean;
    setTag(name: string, value: string): void;
    increment(name: string, size?: number): void;
    recordHistogram(name: string, value: number): void;
    recordEvent(name: string, properties: EventProperties): void;
    startTimersIfNeeded(): void;
    saveAllDataToStorage(): Promise<void>;
    _flush(): Promise<void>;
    /**
     * Send diagnostics data to the server
     */
    fetch(payload: FlushPayload): Promise<void>;
    /**
     * Initialize flush interval logic.
     * Check if 5 minutes has passed since last flush, if so flush immediately.
     * Otherwise set a timer to flush when the interval is reached.
     */
    initializeFlushInterval(): Promise<void>;
    /**
     * Helper method to set flush timer with consistent error handling
     */
    private _setFlushTimer;
    _setSampleRate(sampleRate: number): void;
}
export {};
//# sourceMappingURL=diagnostics-client.d.ts.map