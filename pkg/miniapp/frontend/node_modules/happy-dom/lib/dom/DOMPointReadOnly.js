import * as PropertySymbol from '../PropertySymbol.js';
/**
 * DOM Point Readonly.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/DOMPointReadOnly
 */
export default class DOMPointReadOnly {
    [PropertySymbol.x] = 0;
    [PropertySymbol.y] = 0;
    [PropertySymbol.z] = 0;
    [PropertySymbol.w] = 1;
    /**
     * Constructor.
     *
     * @param [x] X position.
     * @param [y] Y position.
     * @param [z] Width.
     * @param [w] Height.
     */
    constructor(x, y, z, w) {
        this[PropertySymbol.x] = x !== undefined && x !== null ? Number(x) : 0;
        this[PropertySymbol.y] = y !== undefined && y !== null ? Number(y) : 0;
        this[PropertySymbol.z] = z !== undefined && z !== null ? Number(z) : 0;
        this[PropertySymbol.w] = w !== undefined && w !== null ? Number(w) : 1;
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
     * Returns y.
     *
     * @returns Y.
     */
    get y() {
        return this[PropertySymbol.y];
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
     * Returns w.
     *
     * @returns W.
     */
    get w() {
        return this[PropertySymbol.w];
    }
    /**
     * Returns the JSON representation of the object.
     *
     * @returns JSON representation.
     */
    toJSON() {
        return {
            x: this.x,
            y: this.y,
            z: this.z,
            w: this.w
        };
    }
    /**
     * Returns a new DOMPointReadOnly object.
     *
     * @param [otherPoint] Other point.
     * @returns Cloned object.
     */
    static fromPoint(otherPoint) {
        if (!otherPoint) {
            return new this();
        }
        return new this(otherPoint.x ?? null, otherPoint.y ?? null, otherPoint.z ?? null, otherPoint.w ?? null);
    }
}
//# sourceMappingURL=DOMPointReadOnly.js.map