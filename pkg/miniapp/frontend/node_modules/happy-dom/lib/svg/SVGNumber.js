import * as PropertySymbol from '../PropertySymbol.js';
/**
 * SVG number.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGNumber
 */
export default class SVGNumber {
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
     * Returns value.
     *
     * @returns Value.
     */
    get value() {
        const attributeValue = this[PropertySymbol.getAttribute]
            ? this[PropertySymbol.getAttribute]()
            : this[PropertySymbol.attributeValue];
        return parseFloat(attributeValue || '0');
    }
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value) {
        if (this[PropertySymbol.readOnly]) {
            throw new TypeError(`Failed to set the 'value' property on 'SVGNumber': The object is read-only.`);
        }
        const parsedValue = typeof value !== 'number' ? parseFloat(String(value)) : value;
        if (isNaN(parsedValue)) {
            throw new TypeError(`Failed to set the 'value' property on 'SVGNumber': The provided value is not a number.`);
        }
        this[PropertySymbol.attributeValue] = String(value);
        if (this[PropertySymbol.setAttribute]) {
            this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue] || '');
        }
    }
}
//# sourceMappingURL=SVGNumber.js.map