import * as PropertySymbol from '../PropertySymbol.js';
/**
 * SVG Animated Integer.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedInteger
 */
export default class SVGAnimatedInteger {
    // Internal properties
    [PropertySymbol.window];
    [PropertySymbol.getAttribute] = null;
    [PropertySymbol.setAttribute] = null;
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
        return this.baseVal;
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
        const attributeValue = this[PropertySymbol.getAttribute]();
        if (!attributeValue) {
            return 0;
        }
        const value = parseInt(attributeValue);
        if (isNaN(value)) {
            return 0;
        }
        return value;
    }
    /**
     * Sets base value.
     *
     * @param value Base value.
     */
    set baseVal(value) {
        const parsedValue = parseInt(String(value));
        if (isNaN(parsedValue)) {
            throw new this[PropertySymbol.window].TypeError(`TypeError: Failed to set the 'baseVal' property on 'SVGAnimatedInteger': The provided float value is non-finite.`);
        }
        this[PropertySymbol.setAttribute](String(parsedValue));
    }
}
//# sourceMappingURL=SVGAnimatedInteger.js.map