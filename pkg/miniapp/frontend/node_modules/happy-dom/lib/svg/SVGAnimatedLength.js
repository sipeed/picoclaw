import * as PropertySymbol from '../PropertySymbol.js';
import SVGLength from './SVGLength.js';
/**
 * SVG Animated Length.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedLength
 */
export default class SVGAnimatedLength {
    // Internal properties
    [PropertySymbol.window];
    [PropertySymbol.getAttribute];
    [PropertySymbol.setAttribute];
    [PropertySymbol.baseVal] = null;
    [PropertySymbol.animVal] = null;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param window Window.
     * @param options Options.
     * @param options.getAttribute Get attribute.
     * @param options.setAttribute Set attribute.
     */
    constructor(illegalConstructorSymbol, window, options) {
        if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
            throw new TypeError('Illegal constructor');
        }
        this[PropertySymbol.window] = window;
        this[PropertySymbol.getAttribute] = options.getAttribute;
        this[PropertySymbol.setAttribute] = options.setAttribute;
    }
    /**
     * Returns animated value.
     *
     * @returns Animated value.
     */
    get animVal() {
        if (!this[PropertySymbol.animVal]) {
            this[PropertySymbol.animVal] = new SVGLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                readOnly: true,
                getAttribute: this[PropertySymbol.getAttribute]
            });
        }
        return this[PropertySymbol.animVal];
    }
    /**
     * Returns animated value.
     *
     * @param value Animated value.
     */
    set animVal(_value) {
        // Do nothing
    }
    /**
     * Returns base value.
     *
     * @returns Base value.
     */
    get baseVal() {
        if (!this[PropertySymbol.baseVal]) {
            this[PropertySymbol.baseVal] = new SVGLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: this[PropertySymbol.getAttribute],
                setAttribute: this[PropertySymbol.setAttribute]
            });
        }
        return this[PropertySymbol.baseVal];
    }
    /**
     * Returns base value.
     *
     * @param value Base value.
     */
    set baseVal(_value) {
        // Do nothing
    }
}
//# sourceMappingURL=SVGAnimatedLength.js.map