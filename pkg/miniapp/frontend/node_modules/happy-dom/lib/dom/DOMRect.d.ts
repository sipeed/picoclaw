import DOMRectReadOnly from './DOMRectReadOnly.js';
import IDOMRectInit from './IDOMRectInit.js';
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
    set x(value: number);
    /**
     * Returns x.
     *
     * @returns X.
     */
    get x(): number;
    /**
     * Sets y.
     *
     * @param value Y.
     */
    set y(value: number);
    /**
     * Returns y.
     *
     * @returns Y.
     */
    get y(): number;
    /**
     * Sets width.
     *
     * @param value Width.
     */
    set width(value: number);
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width(): number;
    /**
     * Sets height.
     *
     * @param value Height.
     */
    set height(value: number);
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height(): number;
    /**
     * Returns a new DOMRect object.
     *
     * @param other
     * @returns Cloned object.
     */
    static fromRect(other: IDOMRectInit): DOMRect;
}
//# sourceMappingURL=DOMRect.d.ts.map