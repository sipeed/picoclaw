import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGGradientElement from '../svg-gradient-element/SVGGradientElement.js';
/**
 * SVG LinearGradient Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGLinearGradientElement
 */
export default class SVGLinearGradientElement extends SVGGradientElement {
    // Internal properties
    [PropertySymbol.x1] = null;
    [PropertySymbol.y1] = null;
    [PropertySymbol.x2] = null;
    [PropertySymbol.y2] = null;
    /**
     * Returns x1 position.
     *
     * @returns X1 position.
     */
    get x1() {
        if (!this[PropertySymbol.x1]) {
            this[PropertySymbol.x1] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('x1'),
                setAttribute: (value) => this.setAttribute('x1', value)
            });
        }
        return this[PropertySymbol.x1];
    }
    /**
     * Returns y1 position.
     *
     * @returns Y1 position.
     */
    get y1() {
        if (!this[PropertySymbol.y1]) {
            this[PropertySymbol.y1] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('y1'),
                setAttribute: (value) => this.setAttribute('y1', value)
            });
        }
        return this[PropertySymbol.y1];
    }
    /**
     * Returns x2 position.
     *
     * @returns X2 position.
     */
    get x2() {
        if (!this[PropertySymbol.x2]) {
            this[PropertySymbol.x2] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('x2'),
                setAttribute: (value) => this.setAttribute('x2', value)
            });
        }
        return this[PropertySymbol.x2];
    }
    /**
     * Returns y2 position.
     *
     * @returns Y2 position.
     */
    get y2() {
        if (!this[PropertySymbol.y2]) {
            this[PropertySymbol.y2] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('y2'),
                setAttribute: (value) => this.setAttribute('y2', value)
            });
        }
        return this[PropertySymbol.y2];
    }
}
//# sourceMappingURL=SVGLinearGradientElement.js.map