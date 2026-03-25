import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedTransformList from '../../svg/SVGAnimatedTransformList.js';
import SVGGraphicsElement from '../svg-graphics-element/SVGGraphicsElement.js';
/**
 * SVG Gradient Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGGradientElement
 */
export default class SVGGradientElement extends SVGGraphicsElement {
    // Public static properties
    static SVG_SPREADMETHOD_UNKNOWN = 0;
    static SVG_SPREADMETHOD_PAD = 1;
    static SVG_SPREADMETHOD_REFLECT = 2;
    static SVG_SPREADMETHOD_REPEAT = 3;
    // Internal properties
    [PropertySymbol.href] = null;
    [PropertySymbol.gradientUnits] = null;
    [PropertySymbol.gradientTransform] = null;
    [PropertySymbol.spreadMethod] = null;
    /**
     * Returns href.
     *
     * @returns Href.
     */
    get href() {
        if (!this[PropertySymbol.href]) {
            this[PropertySymbol.href] = new SVGAnimatedString(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('href'),
                setAttribute: (value) => this.setAttribute('href', value)
            });
        }
        return this[PropertySymbol.href];
    }
    /**
     * Returns gradient units.
     *
     * @returns Gradient units.
     */
    get gradientUnits() {
        if (!this[PropertySymbol.gradientUnits]) {
            this[PropertySymbol.gradientUnits] = new SVGAnimatedEnumeration(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('gradientUnits'),
                setAttribute: (value) => this.setAttribute('gradientUnits', value),
                values: ['userSpaceOnUse', 'objectBoundingBox'],
                defaultValue: 'objectBoundingBox'
            });
        }
        return this[PropertySymbol.gradientUnits];
    }
    /**
     * Returns gradient transform.
     *
     * @returns Gradient transform.
     */
    get gradientTransform() {
        if (!this[PropertySymbol.gradientTransform]) {
            this[PropertySymbol.gradientTransform] = new SVGAnimatedTransformList(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('gradientTransform'),
                setAttribute: (value) => this.setAttribute('gradientTransform', value)
            });
        }
        return this[PropertySymbol.gradientTransform];
    }
    /**
     * Returns spread method.
     *
     * @returns Spread method.
     */
    get spreadMethod() {
        if (!this[PropertySymbol.spreadMethod]) {
            this[PropertySymbol.spreadMethod] = new SVGAnimatedEnumeration(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('spreadMethod'),
                setAttribute: (value) => this.setAttribute('spreadMethod', value),
                values: ['pad', 'reflect', 'repeat'],
                defaultValue: 'pad'
            });
        }
        return this[PropertySymbol.spreadMethod];
    }
}
//# sourceMappingURL=SVGGradientElement.js.map