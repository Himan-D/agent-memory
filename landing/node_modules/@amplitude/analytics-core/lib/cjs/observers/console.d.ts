type ConsoleLogLevel = keyof Console;
type Callback = (logLevel: ConsoleLogLevel, args: any[]) => void;
/**
 * Observe a console log method (log, warn, error, etc.)
 * @param level - The console log level to observe
 * @param callback - The callback function to call when the console log level is observed
 */
declare function addListener(level: ConsoleLogLevel, callback: Callback): Error | void;
/**
 * Disconnect a callback function from a console log method
 * @param callback - The callback function to disconnect
 */
declare function removeListener(callback: Callback): void;
declare function _restoreConsole(): void;
declare const consoleObserver: {
    addListener: typeof addListener;
    removeListener: typeof removeListener;
    _restoreConsole: typeof _restoreConsole;
};
export { consoleObserver, ConsoleLogLevel };
//# sourceMappingURL=console.d.ts.map