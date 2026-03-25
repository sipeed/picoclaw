import SVGGeometryElement from '../svg-geometry-element/SVGGeometryElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
/**
 * SVG Ellipse Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGEllipseElement
 */
export default class SVGEllipseElement extends SVGGeometryElement {
    // Internal properties
    [PropertySymbol.cx] = null;
    [PropertySymbol.cy] = null;
    [PropertySymbol.rx] = null;
    [PropertySymbol.ry] = null;
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
     * Returns rx.
     *
     * @returns Rx.
     */
    get rx() {
        if (!this[PropertySymbol.rx]) {
            this[PropertySymbol.rx] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('rx'),
                setAttribute: (value) => this.setAttribute('rx', value)
            });
        }
        return this[PropertySymbol.rx];
    }
    /**
     * Returns ry.
     *
     * @returns Ry.
     */
    get ry() {
        if (!this[PropertySymbol.ry]) {
            this[PropertySymbol.ry] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('ry'),
                setAttribute: (value) => this.setAttribute('ry', value)
            });
        }
        return this[PropertySymbol.ry];
    }
}
//# sourceMappingURL=SVGEllipseElement.js.map