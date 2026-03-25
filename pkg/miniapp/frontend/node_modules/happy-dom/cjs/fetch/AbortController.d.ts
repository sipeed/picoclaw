import AbortSignal from './AbortSignal.cjs';
import * as PropertySymbol from '../PropertySymbol.cjs';
import BrowserWindow from '../window/BrowserWindow.cjs';
/**
 * AbortController.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/AbortController
 */
export default class AbortController {
    protected [PropertySymbol.window]: BrowserWindow;
    readonly signal: AbortSignal;
    /**
     * Aborts the signal.
     *
     * @param [reason] Reason.
     */
    abort(reason?: Error): void;
}
//# sourceMappingURL=AbortController.d.ts.map