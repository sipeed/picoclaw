import * as PropertySymbol from '../PropertySymbol.js';
/**
 * AbortController.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/AbortController
 */
export default class AbortController {
    // Public properties
    signal = new this[PropertySymbol.window].AbortSignal();
    /**
     * Aborts the signal.
     *
     * @param [reason] Reason.
     */
    abort(reason) {
        this.signal[PropertySymbol.abort](reason);
    }
}
//# sourceMappingURL=AbortController.js.map