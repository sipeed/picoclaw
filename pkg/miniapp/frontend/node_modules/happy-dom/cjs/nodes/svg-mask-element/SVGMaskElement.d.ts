import SVGElement from '../svg-element/SVGElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.cjs';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.cjs';
/**
 * SVG Mask Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGMaskElement
 */
export default class SVGMaskElement extends SVGElement {
    [PropertySymbol.maskUnits]: SVGAnimatedEnumeration | null;
    [PropertySymbol.maskContentUnits]: SVGAnimatedEnumeration | null;
    [PropertySymbol.x]: SVGAnimatedLength | null;
    [PropertySymbol.y]: SVGAnimatedLength | null;
    [PropertySymbol.width]: SVGAnimatedLength | null;
    [PropertySymbol.height]: SVGAnimatedLength | null;
    /**
     * Returns mask units.
     *
     * @returns Mask units.
     */
    get maskUnits(): SVGAnimatedEnumeration;
    /**
     * Returns mask content units.
     *
     * @returns Mask content units.
     */
    get maskContentUnits(): SVGAnimatedEnumeration;
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
}
//# sourceMappingURL=SVGMaskElement.d.ts.map