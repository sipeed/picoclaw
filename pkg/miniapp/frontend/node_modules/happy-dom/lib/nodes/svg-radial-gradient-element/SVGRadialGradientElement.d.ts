import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGGradientElement from '../svg-gradient-element/SVGGradientElement.js';
/**
 * SVG Radial Gradient Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGRadialGradientElement
 */
export default class SVGRadialGradientElement extends SVGGradientElement {
    [PropertySymbol.cx]: SVGAnimatedLength | null;
    [PropertySymbol.cy]: SVGAnimatedLength | null;
    [PropertySymbol.r]: SVGAnimatedLength | null;
    [PropertySymbol.fx]: SVGAnimatedLength | null;
    [PropertySymbol.fy]: SVGAnimatedLength | null;
    /**
     * Returns cx.
     *
     * @returns Cx.
     */
    get cx(): SVGAnimatedLength;
    /**
     * Returns cy.
     *
     * @returns Cy.
     */
    get cy(): SVGAnimatedLength;
    /**
     * Returns r.
     *
     * @returns R.
     */
    get r(): SVGAnimatedLength;
    /**
     * Returns fx.
     *
     * @returns Fx.
     */
    get fx(): SVGAnimatedLength;
    /**
     * Returns fy.
     *
     * @returns Fy.
     */
    get fy(): SVGAnimatedLength;
}
//# sourceMappingURL=SVGRadialGradientElement.d.ts.map