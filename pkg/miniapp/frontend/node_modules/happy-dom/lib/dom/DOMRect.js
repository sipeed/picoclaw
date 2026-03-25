import DOMRectReadOnly from './DOMRectReadOnly.js';
import * as PropertySymbol from '../PropertySymbol.js';
/**
 * DOM Rect.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/DOMRect
 */
export default class DOMRect extends DOMRectReadOnly {
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
     * Sets width.
     *
     * @param value Width.
     */
    set width(value) {
        this[PropertySymbol.width] = value;
    }
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width() {
        return this[PropertySymbol.width];
    }
    /**
     * Sets height.
     *
     * @param value Height.
     */
    set height(value) {
        this[PropertySymbol.height] = value;
    }
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height() {
        return this[PropertySymbol.height];
    }
    /**
     * Returns a new DOMRect object.
     *
     * @param other
     * @returns Cloned object.
     */
    static fromRect(other) {
        return new DOMRect(other.x, other.y, other.width, other.height);
    }
}
//# sourceMappingURL=DOMRect.js.map