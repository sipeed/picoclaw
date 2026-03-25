import * as PropertySymbol from '../PropertySymbol.js';
import SVGAngle from './SVGAngle.js';
/**
 * SVG Animated Angle.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedAngle
 */
export default class SVGAnimatedAngle {
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
            this[PropertySymbol.animVal] = new SVGAngle(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                readOnly: true,
                getAttribute: this[PropertySymbol.getAttribute]
            });
        }
        return this[PropertySymbol.animVal];
    }
    /**
     * Sets animated value.
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
            this[PropertySymbol.baseVal] = new SVGAngle(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: this[PropertySymbol.getAttribute],
                setAttribute: this[PropertySymbol.setAttribute]
            });
        }
        return this[PropertySymbol.baseVal];
    }
    /**
     * Sets base value.
     *
     * @param value Base value.
     */
    set baseVal(_value) {
        // Do nothing
    }
}
//# sourceMappingURL=SVGAnimatedAngle.js.map