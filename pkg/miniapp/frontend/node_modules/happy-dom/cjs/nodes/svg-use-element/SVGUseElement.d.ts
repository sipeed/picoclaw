import * as PropertySymbol from '../../PropertySymbol.cjs';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.cjs';
import SVGAnimatedString from '../../svg/SVGAnimatedString.cjs';
import SVGGraphicsElement from '../svg-graphics-element/SVGGraphicsElement.cjs';
/**
 * SVG Use Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGUseElement
 */
export default class SVGUseElement extends SVGGraphicsElement {
    [PropertySymbol.href]: SVGAnimatedString | null;
    [PropertySymbol.x]: SVGAnimatedLength | null;
    [PropertySymbol.y]: SVGAnimatedLength | null;
    [PropertySymbol.width]: SVGAnimatedLength | null;
    [PropertySymbol.height]: SVGAnimatedLength | null;
    /**
     * Returns href.
     *
     * @returns Href.
     */
    get href(): SVGAnimatedString;
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
     * Returns width.
     *
     * @returns Width.
     */
    get width(): SVGAnimatedLength;
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height(): SVGAnimatedLength;
}
//# sourceMappingURL=SVGUseElement.d.ts.map