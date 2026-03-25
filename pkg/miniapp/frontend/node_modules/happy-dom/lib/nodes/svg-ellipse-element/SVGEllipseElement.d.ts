import SVGGeometryElement from '../svg-geometry-element/SVGGeometryElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
/**
 * SVG Ellipse Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGEllipseElement
 */
export default class SVGEllipseElement extends SVGGeometryElement {
    [PropertySymbol.cx]: SVGAnimatedLength | null;
    [PropertySymbol.cy]: SVGAnimatedLength | null;
    [PropertySymbol.rx]: SVGAnimatedLength | null;
    [PropertySymbol.ry]: SVGAnimatedLength | null;
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
     * Returns rx.
     *
     * @returns Rx.
     */
    get rx(): SVGAnimatedLength;
    /**
     * Returns ry.
     *
     * @returns Ry.
     */
    get ry(): SVGAnimatedLength;
}
//# sourceMappingURL=SVGEllipseElement.d.ts.map