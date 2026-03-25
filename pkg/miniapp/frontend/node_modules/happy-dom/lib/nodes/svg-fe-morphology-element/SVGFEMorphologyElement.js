import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
/**
 * SVGFEMorphologyElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEMorphologyElement
 */
export default class SVGFEMorphologyElement extends SVGElement {
    // Static properties
    static SVG_MORPHOLOGY_OPERATOR_UNKNOWN = 0;
    static SVG_MORPHOLOGY_OPERATOR_ERODE = 1;
    static SVG_MORPHOLOGY_OPERATOR_DILATE = 2;
    // Internal properties
    [PropertySymbol.height] = null;
    [PropertySymbol.in1] = null;
    [PropertySymbol.operator] = null;
    [PropertySymbol.radiusX] = null;
    [PropertySymbol.radiusY] = null;
    [PropertySymbol.result] = null;
    [PropertySymbol.width] = null;
    [PropertySymbol.x] = null;
    [PropertySymbol.y] = null;
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
     * Returns operator.
     *
     * @returns Operator.
     */
    get operator() {
        if (!this[PropertySymbol.operator]) {
            this[PropertySymbol.operator] = new SVGAnimatedEnumeration(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('operator'),
                setAttribute: (value) => this.setAttribute('operator', value),
                values: ['erode', 'dilate'],
                defaultValue: 'erode'
            });
        }
        return this[PropertySymbol.operator];
    }
    /**
     * Returns radiusX.
     *
     * @returns RadiusX.
     */
    get radiusX() {
        if (!this[PropertySymbol.radiusX]) {
            this[PropertySymbol.radiusX] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('radiusX'),
                setAttribute: (value) => this.setAttribute('radiusX', value)
            });
        }
        return this[PropertySymbol.radiusX];
    }
    /**
     * Returns radiusY.
     *
     * @returns RadiusY.
     */
    get radiusY() {
        if (!this[PropertySymbol.radiusY]) {
            this[PropertySymbol.radiusY] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('radiusY'),
                setAttribute: (value) => this.setAttribute('radiusY', value)
            });
        }
        return this[PropertySymbol.radiusY];
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
     * Returns x.
     *
     * @returns X.
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
     * Returns y.
     *
     * @returns Y.
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
//# sourceMappingURL=SVGFEMorphologyElement.js.map