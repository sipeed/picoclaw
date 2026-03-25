import SVGGeometryElement from '../svg-geometry-element/SVGGeometryElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.cjs';
/**
 * SVG Line Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGLineElement
 */
export default class SVGLineElement extends SVGGeometryElement {
    [PropertySymbol.x1]: SVGAnimatedLength | null;
    [PropertySymbol.y1]: SVGAnimatedLength | null;
    [PropertySymbol.x2]: SVGAnimatedLength | null;
    [PropertySymbol.y2]: SVGAnimatedLength | null;
    /**
     * Returns x1 position.
     *
     * @returns X1 position.
     */
    get x1(): SVGAnimatedLength;
    /**
     * Returns y1 position.
     *
     * @returns Y1 position.
     */
    get y1(): SVGAnimatedLength;
    /**
     * Returns x2 position.
     *
     * @returns X2 position.
     */
    get x2(): SVGAnimatedLength;
    /**
     * Returns y2 position.
     *
     * @returns Y2 position.
     */
    get y2(): SVGAnimatedLength;
}
//# sourceMappingURL=SVGLineElement.d.ts.map