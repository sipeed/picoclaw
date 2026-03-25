import * as PropertySymbol from '../PropertySymbol.js';
import IDOMRectInit from './IDOMRectInit.js';
/**
 * DOM Rect Readonly.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/DOMRectReadOnly
 */
export default class DOMRectReadOnly implements IDOMRectInit {
    protected [PropertySymbol.x]: number;
    protected [PropertySymbol.y]: number;
    protected [PropertySymbol.width]: number;
    protected [PropertySymbol.height]: number;
    /**
     * Constructor.
     *
     * @param [x] X position.
     * @param [y] Y position.
     * @param [width] Width.
     * @param [height] Height.
     */
    constructor(x?: number | null, y?: number | null, width?: number | null, height?: number | null);
    /**
     * Returns x.
     *
     * @returns X.
     */
    get x(): number;
    /**
     * Returns y.
     *
     * @returns Y.
     */
    get y(): number;
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width(): number;
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height(): number;
    /**
     * Returns top.
     *
     * @returns Top.
     */
    get top(): number;
    /**
     * Returns right.
     *
     * @returns Right.
     */
    get right(): number;
    /**
     * Returns bottom.
     *
     * @returns Bottom.
     */
    get bottom(): number;
    /**
     * Returns left.
     *
     * @returns Left.
     */
    get left(): number;
    /**
     * Returns the JSON representation of the object.
     *
     * @returns JSON representation.
     */
    toJSON(): object;
    /**
     * Returns a new DOMRectReadOnly object.
     *
     * @param other
     * @returns Cloned object.
     */
    static fromRect(other: IDOMRectInit): DOMRectReadOnly;
}
//# sourceMappingURL=DOMRectReadOnly.d.ts.map