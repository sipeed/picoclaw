import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGElement from '../svg-element/SVGElement.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedTransformList from '../../svg/SVGAnimatedTransformList.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
/**
 * SVG Pattern Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGPatternElement
 */
export default class SVGPatternElement extends SVGElement {
    [PropertySymbol.href]: SVGAnimatedString | null;
    [PropertySymbol.patternUnits]: SVGAnimatedEnumeration | null;
    [PropertySymbol.patternContentUnits]: SVGAnimatedEnumeration | null;
    [PropertySymbol.patternTransform]: SVGAnimatedTransformList | null;
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
     * Returns pattern units.
     *
     * @returns Pattern units.
     */
    get patternUnits(): SVGAnimatedEnumeration;
    /**
     * Returns pattern content units.
     *
     * @returns Pattern content units.
     */
    get patternContentUnits(): SVGAnimatedEnumeration;
    /**
     * Returns pattern transform.
     *
     * @returns Pattern transform.
     */
    get patternTransform(): SVGAnimatedTransformList;
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
//# sourceMappingURL=SVGPatternElement.d.ts.map