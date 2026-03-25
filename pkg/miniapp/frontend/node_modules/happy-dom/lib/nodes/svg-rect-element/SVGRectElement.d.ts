import SVGGeometryElement from '../svg-geometry-element/SVGGeometryElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
/**
 * SVG Rect Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGRectElement
 */
export default class SVGRectElement extends SVGGeometryElement {
    [PropertySymbol.x]: SVGAnimatedLength | null;
    [PropertySymbol.y]: SVGAnimatedLength | null;
    [PropertySymbol.width]: SVGAnimatedLength | null;
    [PropertySymbol.height]: SVGAnimatedLength | null;
    [PropertySymbol.rx]: SVGAnimatedLength | null;
    [PropertySymbol.ry]: SVGAnimatedLength | null;
    /**
     * Returns x position.
     *
     * @returns X position.
     */
    get x(): SVGAnimatedLength;
    /**
     * Returns y position.
     *
     * @returns Y position.
     */
    get y(): SVGAnimatedLength;
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height(): SVGAnimatedLength;
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width(): SVGAnimatedLength;
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
//# sourceMappingURL=SVGRectElement.d.ts.map