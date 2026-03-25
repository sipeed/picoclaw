import * as PropertySymbol from '../../PropertySymbol.cjs';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.cjs';
import SVGAnimatedString from '../../svg/SVGAnimatedString.cjs';
import SVGAnimatedTransformList from '../../svg/SVGAnimatedTransformList.cjs';
import SVGGraphicsElement from '../svg-graphics-element/SVGGraphicsElement.cjs';
/**
 * SVG Gradient Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGGradientElement
 */
export default class SVGGradientElement extends SVGGraphicsElement {
    static readonly SVG_SPREADMETHOD_UNKNOWN = 0;
    static readonly SVG_SPREADMETHOD_PAD = 1;
    static readonly SVG_SPREADMETHOD_REFLECT = 2;
    static readonly SVG_SPREADMETHOD_REPEAT = 3;
    [PropertySymbol.href]: SVGAnimatedString | null;
    [PropertySymbol.gradientUnits]: SVGAnimatedEnumeration | null;
    [PropertySymbol.gradientTransform]: SVGAnimatedTransformList | null;
    [PropertySymbol.spreadMethod]: SVGAnimatedEnumeration | null;
    /**
     * Returns href.
     *
     * @returns Href.
     */
    get href(): SVGAnimatedString;
    /**
     * Returns gradient units.
     *
     * @returns Gradient units.
     */
    get gradientUnits(): SVGAnimatedEnumeration;
    /**
     * Returns gradient transform.
     *
     * @returns Gradient transform.
     */
    get gradientTransform(): SVGAnimatedTransformList;
    /**
     * Returns spread method.
     *
     * @returns Spread method.
     */
    get spreadMethod(): SVGAnimatedEnumeration;
}
//# sourceMappingURL=SVGGradientElement.d.ts.map