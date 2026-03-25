import DOMPointReadOnly from './DOMPointReadOnly.js';
import * as PropertySymbol from '../PropertySymbol.js';
/**
 * DOM Point.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/DOMPoint
 */
export default class DOMPoint extends DOMPointReadOnly {
    /**
     * Sets x.
     *
     * @param value X.
     */
    set x(value) {
        this[PropertySymbol.x] = value;
    }
    /**
     * Returns x.
     *
     * @returns X.
     */
    get x() {
        return this[PropertySymbol.x];
    }
    /**
     * Sets y.
     *
     * @param value Y.
     */
    set y(value) {
        this[PropertySymbol.y] = value;
    }
    /**
     * Returns y.
     *
     * @returns Y.
     */
    get y() {
        return this[PropertySymbol.y];
    }
    /**
     * Sets z.
     *
     * @param value Z.
     */
    set z(value) {
        this[PropertySymbol.z] = value;
    }
    /**
     * Returns z.
     *
     * @returns Z.
     */
    get z() {
        return this[PropertySymbol.z];
    }
    /**
     * Sets w.
     *
     * @param value W.
     */
    set w(value) {
        this[PropertySymbol.w] = value;
    }
    /**
     * Returns w.
     *
     * @returns W.
     */
    get w() {
        return this[PropertySymbol.w];
    }
}
//# sourceMappingURL=DOMPoint.js.map