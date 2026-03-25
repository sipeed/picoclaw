import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedNumberList from '../../svg/SVGAnimatedNumberList.js';
/**
 * SVGFECompositeElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFECompositeElement
 */
export default class SVGFECompositeElement extends SVGElement {
    // Static properties
    static SVG_FECOMPOSITE_OPERATOR_UNKNOWN = 0;
    static SVG_FECOMPOSITE_OPERATOR_OVER = 1;
    static SVG_FECOMPOSITE_OPERATOR_IN = 2;
    static SVG_FECOMPOSITE_OPERATOR_OUT = 3;
    static SVG_FECOMPOSITE_OPERATOR_ATOP = 4;
    static SVG_FECOMPOSITE_OPERATOR_XOR = 5;
    static SVG_FECOMPOSITE_OPERATOR_ARITHMETIC = 6;
    // Internal properties
    [PropertySymbol.height] = null;
    [PropertySymbol.in1] = null;
    [PropertySymbol.result] = null;
    [PropertySymbol.type] = null;
    [PropertySymbol.values] = null;
    [PropertySymbol.x] = null;
    [PropertySymbol.y] = null;
    [PropertySymbol.width] = null;
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height() {
        if (!this[PropertySymbol.height]) {
            this[PropertySymbol.height] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('height'),
                setAttribute: (value) => this.setAttribute('height', value)
            });
        }
        return this[PropertySymbol.height];
    }
    /**
     * Returns in1.
     *
     * @returns In1.
     */
    get in1() {
        if (!this[PropertySymbol.in1]) {
            this[PropertySymbol.in1] = new SVGAnimatedString(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('in'),
                setAttribute: (value) => this.setAttribute('in', value)
            });
        }
        return this[PropertySymbol.in1];
    }
    /**
     * Returns result.
     *
     * @returns Result.
     */
    get result() {
        if (!this[PropertySymbol.result]) {
            this[PropertySymbol.result] = new SVGAnimatedString(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('result'),
                setAttribute: (value) => this.setAttribute('result', value)
            });
        }
        return this[PropertySymbol.result];
    }
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
                values: ['over', 'in', 'out', 'atop', 'xor', 'arithmetic'],
                defaultValue: 'over'
            });
        }
        return this[PropertySymbol.type];
    }
    /**
     * Returns values.
     *
     * @returns Values.
     */
    get values() {
        if (!this[PropertySymbol.values]) {
            this[PropertySymbol.values] = new SVGAnimatedNumberList(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('values'),
                setAttribute: (value) => this.setAttribute('values', value)
            });
        }
        return this[PropertySymbol.values];
    }
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width() {
        if (!this[PropertySymbol.width]) {
            this[PropertySymbol.width] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('width'),
                setAttribute: (value) => this.setAttribute('width', value)
            });
        }
        return this[PropertySymbol.width];
    }
    /**
     * Returns x position.
     *
     * @returns X position.
     */
    get x() {
        if (!this[PropertySymbol.x]) {
            this[PropertySymbol.x] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('x'),
                setAttribute: (value) => this.setAttribute('x', value)
            });
        }
        return this[PropertySymbol.x];
    }
    /**
     * Returns y position.
     *
     * @returns Y position.
     */
    get y() {
        if (!this[PropertySymbol.y]) {
            this[PropertySymbol.y] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('y'),
                setAttribute: (value) => this.setAttribute('y', value)
            });
        }
        return this[PropertySymbol.y];
    }
}
//# sourceMappingURL=SVGFECompositeElement.js.map