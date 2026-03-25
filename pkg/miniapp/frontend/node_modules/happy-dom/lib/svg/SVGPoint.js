import * as PropertySymbol from '../PropertySymbol.js';
const ATTRIBUTE_SEPARATOR_REGEXP = /[\t\f\n\r, ]+/;
/**
 * SVG point.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGPoint
 */
export default class SVGPoint {
    // Internal properties
    [PropertySymbol.window];
    [PropertySymbol.getAttribute] = null;
    [PropertySymbol.setAttribute] = null;
    [PropertySymbol.attributeValue] = null;
    [PropertySymbol.readOnly] = false;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param window Window.
     * @param [options] Options.
     * @param [options.readOnly] Read only.
     * @param [options.getAttribute] Get attribute.
     * @param [options.setAttribute] Set attribute
     */
    constructor(illegalConstructorSymbol, window, options) {
        if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
            throw new TypeError('Illegal constructor');
        }
        this[PropertySymbol.window] = window;
        if (options) {
            this[PropertySymbol.readOnly] = !!options.readOnly;
            this[PropertySymbol.getAttribute] = options.getAttribute || null;
            this[PropertySymbol.setAttribute] = options.setAttribute || null;
        }
    }
    /**
     * Returns x.
     *
     * @returns X.
     */
    get x() {
        const attributeValue = this[PropertySymbol.getAttribute]
            ? this[PropertySymbol.getAttribute]()
            : this[PropertySymbol.attributeValue];
        const parts = (attributeValue || '').split(ATTRIBUTE_SEPARATOR_REGEXP);
        return !!parts[0] ? parseFloat(parts[0]) : 0;
    }
    /**
     * Sets x.
     *
     * @param value X.
     */
    set x(value) {
        if (this[PropertySymbol.readOnly]) {
            throw new this[PropertySymbol.window].TypeError(`Failed to set the 'x' property on 'SVGPoint': The object is read-only.`);
        }
        this[PropertySymbol.attributeValue] = `${value} ${this.y}`;
        if (this[PropertySymbol.setAttribute]) {
            this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
        }
    }
    /**
     * Returns y.
     *
     * @returns Y.
     */
    get y() {
        const attributeValue = this[PropertySymbol.getAttribute]
            ? this[PropertySymbol.getAttribute]()
            : this[PropertySymbol.attributeValue];
        const parts = (attributeValue || '').split(ATTRIBUTE_SEPARATOR_REGEXP);
        return !!parts[1] ? parseFloat(parts[1]) : 0;
    }
    /**
     * Sets y.
     *
     * @param value Y.
     */
    set y(value) {
        if (this[PropertySymbol.readOnly]) {
            throw new this[PropertySymbol.window].TypeError(`Failed to set the 'y' property on 'SVGPoint': The object is read-only.`);
        }
        this[PropertySymbol.attributeValue] = `${this.x} ${value}`;
        if (this[PropertySymbol.setAttribute]) {
            this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
        }
    }
}
//# sourceMappingURL=SVGPoint.js.map