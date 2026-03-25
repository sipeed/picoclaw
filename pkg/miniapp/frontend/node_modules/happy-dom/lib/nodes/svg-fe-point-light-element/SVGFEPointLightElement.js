import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
/**
 * SVGFEPointLightElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEPointLightElement
 */
export default class SVGFEPointLightElement extends SVGElement {
    // Internal properties
    [PropertySymbol.x] = null;
    [PropertySymbol.y] = null;
    [PropertySymbol.z] = null;
    /**
     * Returns x.
     *
     * @returns X.
     */
    get x() {
        if (!this[PropertySymbol.x]) {
            this[PropertySymbol.x] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
            this[PropertySymbol.y] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('y'),
                setAttribute: (value) => this.setAttribute('y', value)
            });
        }
        return this[PropertySymbol.y];
    }
    /**
     * Returns z.
     *
     * @returns Z.
     */
    get z() {
        if (!this[PropertySymbol.z]) {
            this[PropertySymbol.z] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('z'),
                setAttribute: (value) => this.setAttribute('z', value)
            });
        }
        return this[PropertySymbol.z];
    }
}
//# sourceMappingURL=SVGFEPointLightElement.js.map