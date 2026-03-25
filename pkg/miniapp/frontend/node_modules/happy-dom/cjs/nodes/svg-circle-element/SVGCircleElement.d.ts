import SVGGeometryElement from '../svg-geometry-element/SVGGeometryElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.cjs';
/**
 * SVG Circle Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGCircleElement
 */
export default class SVGCircleElement extends SVGGeometryElement {
    [PropertySymbol.cx]: SVGAnimatedLength | null;
    [PropertySymbol.cy]: SVGAnimatedLength | null;
    [PropertySymbol.r]: SVGAnimatedLength | null;
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
}
//# sourceMappingURL=SVGCircleElement.d.ts.map