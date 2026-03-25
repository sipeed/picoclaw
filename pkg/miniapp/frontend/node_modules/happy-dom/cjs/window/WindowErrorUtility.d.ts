import BrowserWindow from './BrowserWindow.cjs';
import Element from '../nodes/element/Element.cjs';
/**
 * Error utility.
 */
export default class WindowErrorUtility {
    /**
     * Calls a function synchronously wrapped in a try/catch block to capture errors and dispatch error events.
     * If the callback returns a Promise, it will catch errors from the promise.
     *
     * It will also output the errors to the console.
     *
     * @param elementOrWindow Element or Window.
     * @param callback Callback.
     * @param [cleanup] Cleanup callback on error.
     * @returns Result.
     */
    static captureError<T>(elementOrWindow: BrowserWindow | Element, callback: () => T, cleanup?: () => void): T | null;
    /**
     * Dispatches an error event and outputs it to the console.
     *
     * @param elementOrWindow Element or Window.
     * @param error Error.
     */
    static dispatchError(elementOrWindow: BrowserWindow | Element, error: Error): void;
}
//# sourceMappingURL=WindowErrorUtility.d.ts.map