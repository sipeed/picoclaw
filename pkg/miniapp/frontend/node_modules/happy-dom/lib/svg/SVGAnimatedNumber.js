import * as PropertySymbol from '../PropertySymbol.js';
/**
 * SVG Animated Number.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedNumber
 */
export default class SVGAnimatedNumber {
    // Internal properties
    [PropertySymbol.window];
    [PropertySymbol.getAttribute] = null;
    [PropertySymbol.setAttribute] = null;
    [PropertySymbol.defaultValue] = 0;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param window Window.
     * @param options Options.
     * @param options.getAttribute Get attribute.
     * @param options.setAttribute Set attribute.
     * @param options.defaultValue
     */
    constructor(illegalConstructorSymbol, window, options) {
        if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
            throw new TypeError('Illegal constructor');
        }
        this[PropertySymbol.window] = window;
        this[PropertySymbol.getAttribute] = options.getAttribute;
        this[PropertySymbol.setAttribute] = options.setAttribute;
        this[PropertySymbol.defaultValue] = options.defaultValue || 0;
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
            return this[PropertySymbol.defaultValue];
        }
        const value = parseFloat(attributeValue);
        if (isNaN(value)) {
            return this[PropertySymbol.defaultValue];
        }
        return value;
    }
    /**
     * Sets base value.
     *
     * @param value Base value.
     */
    set baseVal(value) {
        const parsedValue = typeof value !== 'number' ? parseFloat(value) : value;
        if (isNaN(parsedValue)) {
            throw new this[PropertySymbol.window].TypeError(`TypeError: Failed to set the 'baseVal' property on 'SVGAnimatedNumber': The provided float value is non-finite.`);
        }
        this[PropertySymbol.setAttribute](String(parsedValue));
    }
}
//# sourceMappingURL=SVGAnimatedNumber.js.map