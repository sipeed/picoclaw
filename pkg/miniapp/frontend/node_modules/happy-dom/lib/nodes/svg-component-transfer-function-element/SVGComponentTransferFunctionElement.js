import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
import SVGAnimatedNumberList from '../../svg/SVGAnimatedNumberList.js';
/**
 * SVGComponentTransferFunctionElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGComponentTransferFunctionElement
 */
export default class SVGComponentTransferFunctionElement extends SVGElement {
    // Static properties
    static SVG_FECOMPONENTTRANSFER_TYPE_UNKNOWN = 0;
    static SVG_FECOMPONENTTRANSFER_TYPE_IDENTITY = 1;
    static SVG_FECOMPONENTTRANSFER_TYPE_TABLE = 2;
    static SVG_FECOMPONENTTRANSFER_TYPE_DISCRETE = 3;
    static SVG_FECOMPONENTTRANSFER_TYPE_LINEAR = 4;
    static SVG_FECOMPONENTTRANSFER_TYPE_GAMMA = 5;
    // Internal properties
    [PropertySymbol.type] = null;
    [PropertySymbol.tableValues] = null;
    [PropertySymbol.slope] = null;
    [PropertySymbol.intercept] = null;
    [PropertySymbol.amplitude] = null;
    [PropertySymbol.exponent] = null;
    [PropertySymbol.offset] = null;
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type() {
        if (!this[PropertySymbol.type]) {
            this[PropertySymbol.type] = new SVGAnimatedEnumeration(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('type'),
                setAttribute: (value) => this.setAttribute('type', value),
                values: ['identity', 'table', 'discrete', 'linear', 'gamma'],
                defaultValue: 'identity'
            });
        }
        return this[PropertySymbol.type];
    }
    /**
     * Returns table values.
     *
     * @returns Table values.
     */
    get tableValues() {
        if (!this[PropertySymbol.tableValues]) {
            this[PropertySymbol.tableValues] = new SVGAnimatedNumberList(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('tableValues'),
                setAttribute: (value) => this.setAttribute('tableValues', value)
            });
        }
        return this[PropertySymbol.tableValues];
    }
    /**
     * Returns slope.
     *
     * @returns Slope.
     */
    get slope() {
        if (!this[PropertySymbol.slope]) {
            this[PropertySymbol.slope] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('slope'),
                setAttribute: (value) => this.setAttribute('slope', value),
                defaultValue: 1
            });
        }
        return this[PropertySymbol.slope];
    }
    /**
     * Returns intercept.
     *
     * @returns Intercept.
     */
    get intercept() {
        if (!this[PropertySymbol.intercept]) {
            this[PropertySymbol.intercept] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('intercept'),
                setAttribute: (value) => this.setAttribute('intercept', value)
            });
        }
        return this[PropertySymbol.intercept];
    }
    /**
     * Returns amplitude.
     *
     * @returns Amplitude.
     */
    get amplitude() {
        if (!this[PropertySymbol.amplitude]) {
            this[PropertySymbol.amplitude] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('amplitude'),
                setAttribute: (value) => this.setAttribute('amplitude', value),
                defaultValue: 1
            });
        }
        return this[PropertySymbol.amplitude];
    }
    /**
     * Returns exponent.
     *
     * @returns Exponent.
     */
    get exponent() {
        if (!this[PropertySymbol.exponent]) {
            this[PropertySymbol.exponent] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('exponent'),
                setAttribute: (value) => this.setAttribute('exponent', value),
                defaultValue: 1
            });
        }
        return this[PropertySymbol.exponent];
    }
    /**
     * Returns offset.
     *
     * @returns Offset.
     */
    get offset() {
        if (!this[PropertySymbol.offset]) {
            this[PropertySymbol.offset] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('offset'),
                setAttribute: (value) => this.setAttribute('offset', value)
            });
        }
        return this[PropertySymbol.offset];
    }
}
//# sourceMappingURL=SVGComponentTransferFunctionElement.js.map