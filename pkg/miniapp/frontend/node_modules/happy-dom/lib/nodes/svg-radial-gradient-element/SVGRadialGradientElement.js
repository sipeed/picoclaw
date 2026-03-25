import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGGradientElement from '../svg-gradient-element/SVGGradientElement.js';
/**
 * SVG Radial Gradient Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGRadialGradientElement
 */
export default class SVGRadialGradientElement extends SVGGradientElement {
    // Internal properties
    [PropertySymbol.cx] = null;
    [PropertySymbol.cy] = null;
    [PropertySymbol.r] = null;
    [PropertySymbol.fx] = null;
    [PropertySymbol.fy] = null;
    /**
     * Returns cx.
     *
     * @returns Cx.
     */
    get cx() {
        if (!this[PropertySymbol.cx]) {
            this[PropertySymbol.cx] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('cx'),
                setAttribute: (value) => this.setAttribute('cx', value)
            });
        }
        return this[PropertySymbol.cx];
    }
    /**
     * Returns cy.
     *
     * @returns Cy.
     */
    get cy() {
        if (!this[PropertySymbol.cy]) {
            this[PropertySymbol.cy] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('cy'),
                setAttribute: (value) => this.setAttribute('cy', value)
            });
        }
        return this[PropertySymbol.cy];
    }
    /**
     * Returns r.
     *
     * @returns R.
     */
    get r() {
        if (!this[PropertySymbol.r]) {
            this[PropertySymbol.r] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('r'),
                setAttribute: (value) => this.setAttribute('r', value)
            });
        }
        return this[PropertySymbol.r];
    }
    /**
     * Returns fx.
     *
     * @returns Fx.
     */
    get fx() {
        if (!this[PropertySymbol.fx]) {
            this[PropertySymbol.fx] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('fx'),
                setAttribute: (value) => this.setAttribute('fx', value)
            });
        }
        return this[PropertySymbol.fx];
    }
    /**
     * Returns fy.
     *
     * @returns Fy.
     */
    get fy() {
        if (!this[PropertySymbol.fy]) {
            this[PropertySymbol.fy] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('fy'),
                setAttribute: (value) => this.setAttribute('fy', value)
            });
        }
        return this[PropertySymbol.fy];
    }
}
//# sourceMappingURL=SVGRadialGradientElement.js.map