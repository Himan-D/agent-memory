import { ILogger } from '../logger';
import { Messenger } from '../types/element-interactions';
type MessageRequest = {
    id: string;
    action: string;
    args: Record<string, any>;
};
export type ActionHandler = (data: any) => void;
/**
 * Brand key used to identify BaseWindowMessenger instances across bundle boundaries.
 */
declare const MESSENGER_BRAND: "__AMPLITUDE_MESSENGER_INSTANCE__";
/**
 * BaseWindowMessenger provides generic cross-window communication via postMessage.
 * Singleton access via getOrCreateWindowMessenger() to prevent duplicate instances
 */
declare class BaseWindowMessenger implements Messenger {
    /** Brand property for cross-bundle instanceof checks. */
    readonly [MESSENGER_BRAND] = true;
    endpoint: string;
    logger?: ILogger;
    private isSetup;
    private messageHandler;
    requestCallbacks: {
        [id: string]: {
            resolve: (data: any) => void;
            reject: (data: any) => void;
        };
    };
    private actionHandlers;
    /**
     * Messages received for actions that had no registered handler yet.
     * Drained automatically when the corresponding handler is registered via
     * registerActionHandler(), solving startup race conditions between
     * independently-initialized plugins.
     */
    private pendingMessages;
    /**
     * Tracks in-flight and completed script loads by URL.
     * Using a map, this prevents duplicate loads before the first resolves.
     */
    private scriptLoadPromises;
    constructor({ origin }?: {
        origin?: string;
    });
    /**
     * Send a message to the parent window (window.opener).
     */
    notify(message: {
        action: string;
        data?: any;
    } | MessageRequest): void;
    /**
     * Send an async request to the parent window with a unique ID.
     * Returns a Promise that resolves when the parent responds.
     */
    sendRequest(action: string, args: Record<string, any>, options?: {
        timeout: number;
    }): Promise<any>;
    /**
     * Handle a response to a previous request by resolving its Promise.
     */
    private handleResponse;
    /**
     * Register a handler for a specific action type.
     * Logs a warning if overwriting an existing handler.
     */
    registerActionHandler(action: string, handler: ActionHandler): void;
    /**
     * Load a script once, deduplicating by URL.
     * Safe against concurrent calls — the second call awaits the first's in-flight Promise
     * rather than triggering a duplicate load.
     */
    loadScriptOnce(url: string): Promise<void>;
    /**
     * Set up the message listener. Idempotent — safe to call multiple times.
     * Subclasses should call super.setup() and then register their own action handlers.
     */
    setup({ logger, endpoint }?: {
        logger?: ILogger;
        endpoint?: string;
    }): void;
    /**
     * Tear down the messenger: remove the message listener, clear all state.
     */
    destroy(): void;
}
/**
 * Get or create a singleton BaseWindowMessenger instance.
 * Ensures only one messenger (and one message listener) exists per page,
 * preventing duplicate script loads and double notifications.
 *
 * The singleton is stored on globalScope under the same MESSENGER_KEY.
 * The branded property check verifies the stored value is actually a messenger.
 */
export declare function getOrCreateWindowMessenger(options?: {
    origin?: string;
}): BaseWindowMessenger;
export type { BaseWindowMessenger };
//# sourceMappingURL=base-window-messenger.d.ts.map