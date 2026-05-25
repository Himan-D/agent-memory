import type { BaseWindowMessenger } from './base-window-messenger';
/**
 * Enable background capture on a messenger instance.
 * Plugins can call this on a shared messenger instance.
 * The first call registers the handlers; subsequent calls are no-ops.
 *
 * @param messenger - The messenger to enable background capture on
 * @param options.scriptUrl - Override the background capture script URL (optional)
 */
export declare function enableBackgroundCapture(messenger: BaseWindowMessenger, options?: {
    scriptUrl?: string;
}): void;
//# sourceMappingURL=background-capture.d.ts.map